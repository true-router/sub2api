// Acceptance runs the real server process. Only external HTTP providers are fake.
package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
}
type response struct {
	Status int
	Data   map[string]any
	Raw    []byte
}
type fixture struct {
	env                []string
	starts             int
	upstreamSuccess    int
	trace              []string
	migrationsVerified int
	wallets            []map[string]any

	ctx                                  context.Context
	dir, binary, origin, postgres, redis string
	db                                   *sql.DB
	process                              *exec.Cmd
	processDone                          chan error
	log                                  *os.File
	checks                               []check
	secrets                              []string
	mock                                 *httptest.Server
	mu                                   sync.Mutex
	modelCalls                           int
	upstreamFailed                       bool
	refunds                              []string
	orders                               map[string]url.Values
	jwtSecret, adminPassword             string
}

func token() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func (f *fixture) checked(name string, ok bool) error {
	f.checks = append(f.checks, check{name, ok})
	if !ok {
		return fmt.Errorf("acceptance failed: %s", name)
	}
	fmt.Println("PASS " + name)
	return nil
}
func run(args ...string) (string, error) {
	c := exec.Command(args[0], args[1:]...)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", args[0], err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
func (f *fixture) request(method, path, bearer string, body any, key string) (response, error) {
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return response{}, err
		}
	}
	req, err := http.NewRequestWithContext(f.ctx, method, f.origin+path, bytes.NewReader(encoded))
	if err != nil {
		return response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return response{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return response{}, err
	}
	var obj map[string]any
	_ = json.Unmarshal(raw, &obj)
	data, _ := obj["data"].(map[string]any)
	if data == nil {
		data = obj
	}
	return response{res.StatusCode, data, raw}, nil
}
func (f *fixture) api(method, path, bearer string, body any) (map[string]any, error) {
	r, err := f.request(method, path, bearer, body, token())
	if err != nil {
		return nil, err
	}
	if r.Status < 200 || r.Status >= 300 {
		return nil, fmt.Errorf("%s %s returned %d: %s", method, path, r.Status, r.Raw)
	}
	return r.Data, nil
}
func (f *fixture) login(email, password string) (string, map[string]any, error) {
	data, err := f.api("POST", "/api/v1/auth/login", "", map[string]any{"email": email, "password": password})
	if err != nil {
		return "", nil, err
	}
	access, _ := data["access_token"].(string)
	if access == "" {
		return "", nil, errors.New("login returned no access token")
	}
	f.secrets = append(f.secrets, access)
	user, _ := data["user"].(map[string]any)
	return access, user, nil
}
func (f *fixture) bootstrap() error {
	var err error
	f.dir, err = os.MkdirTemp("", "sub2api-production-billing-")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(f.dir, "data"), 0700); err != nil {
		return err
	}
	f.binary = filepath.Join(f.dir, "server")
	if _, err = run("go", "build", "-o", f.binary, "./cmd/server"); err != nil {
		return err
	}
	id := fmt.Sprintf("sub2api-production-billing-%d", os.Getpid())
	f.postgres = id + "-postgres"
	f.redis = id + "-redis"
	password := token()
	f.adminPassword = "Production-fixture-" + token()
	f.jwtSecret = token() + token()
	f.secrets = append(f.secrets, password, f.adminPassword, f.jwtSecret)
	if _, err = run("docker", "run", "-d", "--name", f.postgres, "-e", "POSTGRES_PASSWORD="+password, "-e", "POSTGRES_DB=production_billing_e2e", "-p", "127.0.0.1::5432", "postgres:18-alpine"); err != nil {
		return err
	}
	if _, err = run("docker", "run", "-d", "--name", f.redis, "-p", "127.0.0.1::6379", "redis:8-alpine"); err != nil {
		return err
	}
	pg, err := run("docker", "port", f.postgres, "5432/tcp")
	if err != nil {
		return err
	}
	rd, err := run("docker", "port", f.redis, "6379/tcp")
	if err != nil {
		return err
	}
	_, pgPort, err := net.SplitHostPort(pg)
	if err != nil {
		return err
	}
	_, redisPort, err := net.SplitHostPort(rd)
	if err != nil {
		return err
	}
	for i := 0; i < 80; i++ {
		if _, err = run("docker", "exec", f.postgres, "pg_isready", "-U", "postgres", "-d", "production_billing_e2e"); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	f.db, err = sql.Open("postgres", fmt.Sprintf("postgres://postgres:%s@127.0.0.1:%s/production_billing_e2e?sslmode=disable", password, pgPort))
	if err != nil {
		return err
	}
	f.orders = map[string]url.Values{}
	f.mock = httptest.NewServer(http.HandlerFunc(f.external))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	f.origin = fmt.Sprintf("http://127.0.0.1:%d", port)
	f.log, err = os.Create(filepath.Join(f.dir, "server.log"))
	if err != nil {
		return err
	}
	env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + f.dir, "DATA_DIR=" + filepath.Join(f.dir, "data"), "AUTO_SETUP=true", "RUN_MODE=standard", "SERVER_HOST=127.0.0.1", "SERVER_PORT=" + strconv.Itoa(port), "SERVER_MODE=release", "DATABASE_HOST=127.0.0.1", "DATABASE_PORT=" + pgPort, "DATABASE_USER=postgres", "DATABASE_PASSWORD=" + password, "DATABASE_DBNAME=production_billing_e2e", "DATABASE_SSLMODE=disable", "REDIS_HOST=127.0.0.1", "REDIS_PORT=" + redisPort, "ADMIN_EMAIL=owner@production-billing.invalid", "ADMIN_PASSWORD=" + f.adminPassword, "JWT_SECRET=" + f.jwtSecret, "TOTP_ENCRYPTION_KEY=" + token() + token()[:16], "PRICING_REMOTE_URL=" + f.mock.URL + "/prices.json", "PRICING_HASH_URL=" + f.mock.URL + "/prices.sha256", "SETUP_MIGRATION_TIMEOUT_SECONDS=300", "LOG_OUTPUT_TO_FILE=false"}
	f.env = env
	if err := f.start(); err != nil {
		return err
	}
	return f.checked("Production binary boots with real migrations and Redis", true)
}
func (f *fixture) start() error {
	f.process = exec.Command(f.binary)
	f.process.Dir = f.dir
	f.process.Env = f.env
	f.process.Stdout = f.log
	f.process.Stderr = f.log
	if err := f.process.Start(); err != nil {
		return err
	}
	f.starts++
	f.processDone = make(chan error, 1)
	proc, done := f.process, f.processDone
	go func() { done <- proc.Wait() }()
	for i := 0; i < 600; i++ {
		select {
		case err := <-f.processDone:
			f.process = nil
			return fmt.Errorf("server exited during bootstrap: %v", err)
		default:
		}
		r, e := f.request("GET", "/health", "", nil, "")
		if e == nil && r.Status == 200 {
			return nil
		}
		select {
		case <-f.ctx.Done():
			return f.ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return errors.New("server bootstrap timed out")
}
func (f *fixture) external(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.trace = append(f.trace, r.Method+" "+r.URL.Path)
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/prices.json" {
		io.WriteString(w, `{"gpt-4o-mini":{"input_cost_per_token":0.000001,"output_cost_per_token":0.000002,"max_input_tokens":128000,"max_output_tokens":16000,"mode":"chat"}}`)
		return
	}
	if r.URL.Path == "/prices.sha256" {
		w.WriteHeader(404)
		return
	}
	if r.URL.Path == "/v1/chat/completions" || r.URL.Path == "/chat/completions" {
		f.mu.Lock()
		f.modelCalls++
		callID := f.modelCalls
		failed := f.upstreamFailed
		f.mu.Unlock()
		if failed {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":{"message":"synthetic upstream rejected request","type":"invalid_request_error"}}`)
			return
		}
		f.mu.Lock()
		f.upstreamSuccess++
		f.mu.Unlock()
		w.Header().Set("x-request-id", fmt.Sprintf("upstream-%d", callID))
		fmt.Fprintf(w, `{"id":"chatcmpl-production-fixture-%d","object":"chat.completion","created":1794400000,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"fixture answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`, callID)
		return
	}
	if r.URL.Path == "/mapi.php" {
		_ = r.ParseForm()
		f.mu.Lock()
		f.orders[r.Form.Get("out_trade_no")] = r.Form
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 1, "trade_no": "trade-" + r.Form.Get("out_trade_no"), "qrcode": "https://example.test/fixture-only"})
		return
	}
	if r.URL.Path == "/api.php" && r.URL.Query().Get("act") == "order" {
		_ = r.ParseForm()
		name := strings.TrimPrefix(r.Form.Get("out_trade_no"), "trade-")
		f.mu.Lock()
		o := f.orders[name]
		paid := 0
		if o.Get("fixture_paid") == "true" {
			paid = 1
		}
		amount := o.Get("money")
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 1, "status": paid, "money": amount, "out_trade_no": r.Form.Get("out_trade_no"), "trade_no": "trade-" + name})
		return
	}
	if r.URL.Path == "/api.php" && r.URL.Query().Get("act") == "refund" {
		_ = r.ParseForm()
		f.mu.Lock()
		f.refunds = append(f.refunds, r.Form.Get("money"))
		f.mu.Unlock()
		io.WriteString(w, `{"code":1,"msg":"ok"}`)
		return
	}
	w.WriteHeader(404)
	io.WriteString(w, `{"error":"unsupported fixture endpoint"}`)
}
func id(value any) int64 { n, _ := value.(float64); return int64(n) }
func sign(values url.Values, key string) string {
	keys := make([]string, 0, len(values))
	for name := range values {
		if name != "sign" && name != "sign_type" && values.Get(name) != "" {
			keys = append(keys, name)
		}
	}
	sort.Strings(keys)
	var parts []string
	for _, name := range keys {
		parts = append(parts, name+"="+values.Get(name))
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + key))
	return hex.EncodeToString(sum[:])
}
func (f *fixture) callback(order map[string]any, amount string, valid bool) (response, error) {
	name := order["out_trade_no"].(string)
	values := url.Values{"pid": {"fixture-merchant"}, "out_trade_no": {name}, "trade_no": {"trade-" + name}, "trade_status": {"TRADE_SUCCESS"}, "money": {amount}, "type": {"alipay"}, "sign_type": {"MD5"}}
	values.Set("sign", sign(values, "fixture-merchant-key"))
	if !valid {
		values.Set("sign", "invalid-fixture-signature")
	}
	return f.request("GET", "/api/v1/payment/webhook/easypay?"+values.Encode(), "", nil, "")
}
func (f *fixture) markPaid(order map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orders[order["out_trade_no"].(string)].Set("fixture_paid", "true")
}
func money(value any) decimal.Decimal {
	d, err := decimal.NewFromString(fmt.Sprint(value))
	if err != nil {
		panic(err)
	}
	return d
}
func (f *fixture) wallet(name, bearer string, uid int64, total, paid, bonus string) error {
	deadline := time.Now().Add(8 * time.Second)
	var balance, gifts, unknown string
	for {
		err := f.db.QueryRowContext(f.ctx, `SELECT balance::text,bonus_balance::text,unverified_balance::text FROM users WHERE id=$1`, uid).Scan(&balance, &gifts, &unknown)
		if err != nil {
			return err
		}
		if money(balance).Equal(money(total)) && money(gifts).Equal(money(bonus)) && money(unknown).IsZero() {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s: database wallet total=%s gifts=%s unverified=%s, expected total=%s gifts=%s", name, balance, gifts, unknown, total, bonus)
		}
		time.Sleep(30 * time.Millisecond)
	}
	profile, err := f.api("GET", "/api/v1/user/profile", bearer, nil)
	if err != nil {
		return err
	}
	f.wallets = append(f.wallets, map[string]any{"name": name, "user_id": uid, "balance": balance, "bonus_balance": gifts, "paid_balance": money(balance).Sub(money(gifts)).Sub(money(unknown)).String(), "unverified_balance": unknown})
	return f.checked(name, money(balance).Sub(money(gifts)).Sub(money(unknown)).Equal(money(paid)) && money(profile["balance"]).Equal(money(total)) && money(profile["paid_balance"]).Equal(money(paid)) && money(profile["bonus_balance"]).Equal(money(bonus)) && money(profile["unverified_balance"]).IsZero())
}
func (f *fixture) gateway(key string) (response, error) {
	return f.request("POST", "/v1/chat/completions", key, map[string]any{"model": "gpt-4o-mini", "messages": []any{map[string]any{"role": "user", "content": "synthetic billing acceptance"}}, "stream": false}, "")
}
func (f *fixture) migrations() error {
	rows, err := f.db.QueryContext(f.ctx, `SELECT filename,checksum FROM schema_migrations ORDER BY filename`)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var name, digest string
		if err := rows.Scan(&name, &digest); err != nil {
			return err
		}
		raw, err := os.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(strings.TrimSpace(string(raw))))
		if hex.EncodeToString(sum[:]) != digest {
			return fmt.Errorf("migration checksum mismatch: %s", name)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	expected, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return err
	}
	f.migrationsVerified = count
	return f.checked(fmt.Sprintf("Production migration runner applied %d source-verified migrations", count), count == len(expected))
}

func (f *fixture) scenario() error {
	admin, user, err := f.login("owner@production-billing.invalid", f.adminPassword)
	if err != nil {
		return err
	}
	if err = f.checked("Real password login returns an administrator JWT", user["role"] == "admin"); err != nil {
		return err
	}
	r, err := f.request("GET", "/api/v1/user/profile", "", nil, "")
	if err != nil {
		return err
	}
	if err = f.checked("Anonymous user profile request is denied", r.Status == 401); err != nil {
		return err
	}
	compliance, err := f.api("GET", "/api/v1/admin/compliance", admin, nil)
	if err != nil {
		return err
	}
	if _, err = f.api("POST", "/api/v1/admin/compliance/accept", admin, map[string]any{"phrase": compliance["ack_phrase_en"], "language": "en"}); err != nil {
		return err
	}
	group, err := f.api("POST", "/api/v1/admin/groups", admin, map[string]any{"name": "Production billing fixture", "platform": "openai", "rate_multiplier": 1, "subscription_type": "standard", "model_pricing": []any{map[string]any{"platform": "openai", "models": []string{"gpt-4o-mini"}, "billing_mode": "per_request", "per_request_price": 1}}})
	if err != nil {
		return err
	}
	groupID := group["id"]
	if _, err = f.api("POST", "/api/v1/admin/accounts", admin, map[string]any{"name": "Local model fixture", "platform": "openai", "type": "apikey", "credentials": map[string]any{"api_key": "production-fixture-upstream-key", "base_url": f.mock.URL}, "concurrency": 10, "priority": 1, "group_ids": []any{groupID}, "upstream_billing_probe_enabled": false}); err != nil {
		return err
	}
	member, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "member@production-billing.invalid", "password": "Production-member-fixture-2026", "role": "user", "concurrency": 5, "allowed_groups": []any{groupID}})
	if err != nil {
		return err
	}
	memberToken, _, err := f.login("member@production-billing.invalid", "Production-member-fixture-2026")
	if err != nil {
		return err
	}
	if err = f.checked("Normal user logs in through the production authentication route", member["id"] != nil); err != nil {
		return err
	}
	parts := strings.Split(memberToken, ".")
	rawClaims, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return e
	}
	claims := jwt.MapClaims{}
	if e = json.Unmarshal(rawClaims, &claims); e != nil {
		return e
	}
	claims["exp"] = time.Now().Add(-time.Minute).Unix()
	expired, e := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(f.jwtSecret))
	if e != nil {
		return e
	}
	f.secrets = append(f.secrets, expired)
	r, err = f.request("GET", "/api/v1/user/profile", expired, nil, "")
	if err != nil {
		return err
	}
	if err = f.checked("Production JWT middleware rejects an expired signed token", r.Status == 401 && strings.Contains(string(r.Raw), "TOKEN_EXPIRED")); err != nil {
		return err
	}
	parts[2] = "forged-fixture-signature"
	r, err = f.request("GET", "/api/v1/user/profile", strings.Join(parts, "."), nil, "")
	if err != nil {
		return err
	}
	if err = f.checked("Production JWT middleware rejects a forged signature", r.Status == 401); err != nil {
		return err
	}

	r, err = f.request("POST", fmt.Sprintf("/api/v1/admin/users/%.0f/balance", member["id"]), memberToken, map[string]any{"operation": "add_paid", "balance": 10, "notes": "forbidden"}, token())
	if err != nil {
		return err
	}
	if err = f.checked("Normal user cannot invoke admin balance adjustment", r.Status == 403); err != nil {
		return err
	}
	key, err := f.api("POST", "/api/v1/keys", memberToken, map[string]any{"name": "Production gateway key", "group_id": groupID})
	if err != nil {
		return err
	}
	if err = f.checked("Real API key is created through the user route", key["key"] != nil); err != nil {
		return err
	}
	if err = f.migrations(); err != nil {
		return err
	}
	memberID := id(member["id"])
	apiKey := key["key"].(string)
	f.secrets = append(f.secrets, apiKey)
	if err = f.wallet("Created user has no paid or promotional funds", memberToken, memberID, "0", "0", "0"); err != nil {
		return err
	}
	r, err = f.gateway(apiKey)
	if err != nil {
		return err
	}
	if err = f.checked("Gateway denies unfunded API key before model forwarding", r.Status == 402 || r.Status == 403); err != nil {
		return err
	}
	provider, err := f.api("POST", "/api/v1/admin/payment/providers", admin, map[string]any{"provider_key": "easypay", "name": "Production payment fixture", "enabled": true, "supported_types": []string{"alipay"}, "refund_enabled": true, "allow_user_refund": true, "config": map[string]string{"pid": "fixture-merchant", "pkey": "fixture-merchant-key", "apiBase": f.mock.URL, "notifyUrl": f.origin + "/api/v1/payment/webhook/easypay", "returnUrl": "https://example.test/fixture-only"}})
	if err != nil {
		return err
	}
	_ = provider
	if _, err = f.api("PUT", "/api/v1/admin/payment/config", admin, map[string]any{"enabled": true, "enabled_payment_types": []string{"alipay"}, "min_amount": 1, "max_amount": 100, "daily_limit": 1000, "balance_recharge_multiplier": 1, "recharge_bonus_mode": "bonus", "recharge_bonus_tiers": []any{map[string]any{"min_amount": 1, "bonus_percent": 20}}}); err != nil {
		return err
	}
	order, err := f.api("POST", "/api/v1/payment/orders", memberToken, map[string]any{"amount": 10, "payment_type": "alipay", "order_type": "balance"})
	if err != nil {
		return err
	}
	if err = f.checked("Checkout uses real bonus quote and original cash amount", money(order["amount"]).Equal(money("12")) && money(order["bonus_amount"]).Equal(money("2")) && money(order["pay_amount"]).Equal(money("10"))); err != nil {
		return err
	}
	if err = f.wallet("Creating an unpaid order does not credit funds", memberToken, memberID, "0", "0", "0"); err != nil {
		return err
	}
	r, err = f.callback(order, "10.00", false)
	if err != nil {
		return err
	}
	if err = f.checked("Invalid payment signature is rejected", r.Status == 400); err != nil {
		return err
	}
	if err = f.wallet("Invalid callback leaves wallet unchanged", memberToken, memberID, "0", "0", "0"); err != nil {
		return err
	}
	f.markPaid(order)
	r, err = f.callback(order, "11.00", true)
	if err != nil {
		return err
	}
	if err = f.checked("Signed amount mismatch is rejected against provider order", r.Status >= 400); err != nil {
		return err
	}
	if err = f.wallet("Mismatched callback leaves wallet unchanged", memberToken, memberID, "0", "0", "0"); err != nil {
		return err
	}
	r, err = f.callback(order, "10.00", true)
	if err != nil {
		return err
	}
	if err = f.checked("Verified callback credits through the production fulfillment route", r.Status == 200 && strings.TrimSpace(string(r.Raw)) == "success"); err != nil {
		return err
	}
	if err = f.wallet("Verified payment splits principal and promotional credit", memberToken, memberID, "12", "10", "2"); err != nil {
		return err
	}
	r, err = f.callback(order, "10.00", true)
	if err != nil {
		return err
	}
	if err = f.checked("Replayed payment callback is acknowledged", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Callback replay cannot credit the wallet twice", memberToken, memberID, "12", "10", "2"); err != nil {
		return err
	}
	r, err = f.gateway(apiKey)
	if err != nil {
		return err
	}
	if err = f.checked("Authenticated gateway schedules and calls the model upstream", r.Status == 200); err != nil {
		return fmt.Errorf("%w; gateway response: %s", err, r.Raw)
	}
	if err = f.wallet("Production gateway pricing and billing consume promotional funds first", memberToken, memberID, "11", "10", "1"); err != nil {
		return err
	}
	orderPath := fmt.Sprintf("/api/v1/payment/orders/%d", id(order["order_id"]))
	if _, err = f.api("POST", orderPath+"/refund-request", memberToken, map[string]any{"reason": "fixture refund"}); err != nil {
		return err
	}
	refunded, err := f.api("POST", fmt.Sprintf("/api/v1/admin/payment/orders/%d/refund", id(order["order_id"])), admin, map[string]any{"amount": 10, "deduct_balance": true, "reason": "production route fixture", "force": false})
	if err != nil {
		return err
	}
	if err = f.checked("Admin refund processes the user's real refund request", refunded["success"] == true); err != nil {
		return err
	}
	if err = f.wallet("Refund removes principal and only remaining order gifts", memberToken, memberID, "0", "0", "0"); err != nil {
		return err
	}
	f.mu.Lock()
	callsBefore := f.modelCalls
	refunds := append([]string{}, f.refunds...)
	f.mu.Unlock()
	if err = f.checked("Payment adapter refunds actual cash only once", len(refunds) == 1 && refunds[0] == "10.00"); err != nil {
		return err
	}
	r, err = f.gateway(apiKey)
	if err != nil {
		return err
	}
	if err = f.checked("Warmed production caches deny spending immediately after refund", r.Status == 402 || r.Status == 403); err != nil {
		return err
	}
	f.mu.Lock()
	callsAfter := f.modelCalls
	f.mu.Unlock()
	if err = f.checked("Refunded wallet cannot reach the upstream model", callsAfter == callsBefore); err != nil {
		return err
	}

	// A second real user traverses gift exhaustion, principal use and audited
	// conversion through authenticated production handlers.
	second, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "second@production-billing.invalid", "password": "Production-second-fixture-2026", "role": "user", "concurrency": 5, "allowed_groups": []any{groupID}})
	if err != nil {
		return err
	}
	secondID := id(second["id"])
	secondToken, _, err := f.login("second@production-billing.invalid", "Production-second-fixture-2026")
	if err != nil {
		return err
	}
	secondKey, err := f.api("POST", "/api/v1/keys", secondToken, map[string]any{"name": "Second production key", "group_id": groupID})
	if err != nil {
		return err
	}
	secondAPIKey := secondKey["key"].(string)
	f.secrets = append(f.secrets, secondAPIKey)
	for _, path := range []string{orderPath, fmt.Sprintf("/api/v1/keys/%d", id(key["id"]))} {
		r, err = f.request("GET", path, secondToken, nil, "")
		if err != nil {
			return err
		}
		if err = f.checked("Cross-user resource access is denied: "+path, r.Status == 403 || r.Status == 404); err != nil {
			return err
		}
	}
	r, err = f.request("POST", orderPath+"/refund-request", secondToken, map[string]any{"reason": "cross-user forbidden"}, token())
	if err != nil {
		return err
	}
	if err = f.checked("Cross-user refund request is denied", r.Status == 403 || r.Status == 404); err != nil {
		return err
	}
	r, err = f.request("POST", fmt.Sprintf("/api/v1/admin/users/%d/balance", secondID), secondToken, map[string]any{"operation": "convert_paid_to_bonus", "balance": 1, "notes": "forbidden"}, token())
	if err != nil {
		return err
	}
	if err = f.checked("Non-admin cannot reclassify funds", r.Status == 403); err != nil {
		return err
	}
	secondOrder, err := f.api("POST", "/api/v1/payment/orders", secondToken, map[string]any{"amount": 10, "payment_type": "alipay", "order_type": "balance"})
	if err != nil {
		return err
	}
	f.markPaid(secondOrder)
	r, err = f.callback(secondOrder, "10.00", true)
	if err != nil {
		return err
	}
	if err = f.checked("Second user's signed callback is accepted", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Second user receives independent paid and gift funds", secondToken, secondID, "12", "10", "2"); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		r, err = f.gateway(secondAPIKey)
		if err != nil {
			return err
		}
		if r.Status != 200 {
			return fmt.Errorf("gift exhaustion gateway status %d: %s", r.Status, r.Raw)
		}
	}
	if err = f.wallet("Real gateway exhausts gifts before consuming principal", secondToken, secondID, "9", "9", "0"); err != nil {
		return err
	}
	conversionPath := fmt.Sprintf("/api/v1/admin/users/%d/balance", secondID)
	conversion := map[string]any{"operation": "convert_paid_to_bonus", "balance": 2, "notes": "production-path source conversion"}
	conversionKey := token()
	r, err = f.request("POST", conversionPath, admin, conversion, conversionKey)
	if err != nil {
		return err
	}
	if err = f.checked("Admin can convert paid funds through real authenticated route", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Production conversion preserves total and changes both sources", secondToken, secondID, "9", "7", "2"); err != nil {
		return err
	}
	r, err = f.request("POST", conversionPath, admin, conversion, conversionKey)
	if err != nil {
		return err
	}
	if err = f.checked("Persistent HTTP idempotency replays conversion", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Conversion replay leaves source amounts unchanged", secondToken, secondID, "9", "7", "2"); err != nil {
		return err
	}
	badConversion := map[string]any{"operation": "convert_paid_to_bonus", "balance": 1, "notes": "different request"}
	r, err = f.request("POST", conversionPath, admin, badConversion, conversionKey)
	if err != nil {
		return err
	}
	if err = f.checked("Real idempotency guard rejects conflicting conversion body", r.Status == 409); err != nil {
		return err
	}
	// Stop and start a real executable while retaining only its dedicated data,
	// database and Redis. This is graceful OS-process restart, not fault injection.
	previousPID := f.process.Process.Pid
	if err = f.stop(); err != nil {
		return err
	}
	if err = f.start(); err != nil {
		return err
	}
	if err = f.checked("Actual server PID changes after restart", f.process.Process.Pid != previousPID && f.starts == 2); err != nil {
		return err
	}
	if err = f.wallet("Existing JWT and persisted source balances survive process restart", secondToken, secondID, "9", "7", "2"); err != nil {
		return err
	}
	r, err = f.request("POST", conversionPath, admin, conversion, conversionKey)
	if err != nil {
		return err
	}
	if err = f.checked("Idempotent conversion receipt survives real process restart", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Replayed conversion after restart cannot change funds", secondToken, secondID, "9", "7", "2"); err != nil {
		return err
	}
	r, err = f.gateway(secondAPIKey)
	if err != nil {
		return err
	}
	if err = f.checked("Existing API key still traverses gateway after process restart", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Gateway spends converted gift after process restart", secondToken, secondID, "8", "7", "1"); err != nil {
		return err
	}
	r, err = f.request("POST", fmt.Sprintf("/api/v1/admin/payment/orders/%d/refund", id(secondOrder["order_id"])), admin, map[string]any{"amount": 10, "deduct_balance": true, "force": true, "reason": "cannot refund gifts"}, token())
	if err != nil {
		return err
	}
	if err = f.checked("Forced refund cannot substitute converted gifts for spent principal", r.Status == 200 && r.Data["success"] == false); err != nil {
		return err
	}
	if err = f.wallet("Rejected refund leaves real wallet and gift sources unchanged", secondToken, secondID, "8", "7", "1"); err != nil {
		return err
	}
	// Failed model requests must not write successful usage or debit sources.
	f.mu.Lock()
	f.upstreamFailed = true
	f.mu.Unlock()
	r, err = f.gateway(secondAPIKey)
	if err != nil {
		return err
	}
	if err = f.checked("Model error propagates through production gateway", r.Status >= 400); err != nil {
		return err
	}
	if err = f.wallet("Failed upstream request cannot debit the wallet", secondToken, secondID, "8", "7", "1"); err != nil {
		return err
	}
	if _, err = f.api("DELETE", fmt.Sprintf("/api/v1/keys/%d", id(secondKey["id"])), secondToken, nil); err != nil {
		return err
	}
	f.mu.Lock()
	beforeDelete := f.modelCalls
	f.mu.Unlock()
	r, err = f.gateway(secondAPIKey)
	if err != nil {
		return err
	}
	if err = f.checked("Revoked API key is rejected despite warmed caches", r.Status == 401 || r.Status == 403); err != nil {
		return err
	}
	f.mu.Lock()
	afterDelete := f.modelCalls
	expectedUsage := f.upstreamSuccess
	f.mu.Unlock()
	if err = f.checked("Revoked key cannot reach upstream", beforeDelete == afterDelete); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		var count int
		var total string
		if err = f.db.QueryRowContext(f.ctx, `SELECT COUNT(*),COALESCE(SUM(actual_cost),0)::text FROM usage_logs WHERE user_id IN ($1,$2)`, memberID, secondID).Scan(&count, &total); err != nil {
			return err
		}
		if count == expectedUsage && money(total).Equal(money(expectedUsage)) {
			break
		}
		if i == 199 {
			return fmt.Errorf("usage journal mismatch: count=%d actual_cost=%s expected=%d", count, total, expectedUsage)
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err = f.checked("Real usage logs account for every successful gateway charge", true); err != nil {
		return err
	}
	var journalCount, auditCount int
	if err = f.db.QueryRowContext(f.ctx, `SELECT COUNT(*) FROM wallet_journal WHERE user_id=$1 AND operation='convert_paid_to_bonus'`, secondID).Scan(&journalCount); err != nil {
		return err
	}
	if err = f.db.QueryRowContext(f.ctx, `SELECT COUNT(*) FROM audit_logs WHERE actor_user_id=$1`, id(user["id"])).Scan(&auditCount); err != nil {
		return err
	}
	if err = f.checked("Single conversion journal and production admin audit records persist", journalCount == 1 && auditCount > 0); err != nil {
		return err
	}
	return nil
}
func (f *fixture) stop() error {
	if f.process == nil {
		return nil
	}
	proc := f.process
	f.process = nil
	if err := proc.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	select {
	case err := <-f.processDone:
		return err
	case <-time.After(8 * time.Second):
		_ = proc.Process.Kill()
		<-f.processDone
		return errors.New("server did not stop gracefully")
	}
}
func (f *fixture) close() error {
	cleanupErr := f.stop()
	if f.db != nil {
		f.db.Close()
	}
	if f.mock != nil {
		f.mock.Close()
	}
	if f.log != nil {
		f.log.Close()
	}
	for _, name := range []string{f.postgres, f.redis} {
		if name != "" {
			_, err := run("docker", "rm", "-fv", name)
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}
func main() {
	output := flag.String("output", "../.local/production-billing-e2e/receipt.json", "JSON acceptance receipt path, relative to backend")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	f := &fixture{ctx: ctx}
	err := f.bootstrap()
	if err == nil {
		err = f.scenario()
	}
	cleanupErr := f.close()
	err = errors.Join(err, cleanupErr)
	report := map[string]any{"passed": err == nil, "checks": f.checks, "mode": "production OS process, real routing/authentication/migrations/database/cache; synthetic external providers", "live_payments": false, "fixture_services_removed": cleanupErr == nil, "completed_at_utc": time.Now().UTC().Format(time.RFC3339), "os_process_starts": f.starts, "verified_migrations": f.migrationsVerified, "wallet_snapshots": f.wallets, "external_requests": f.trace}
	if f.binary != "" {
		if data, e := os.ReadFile(f.binary); e == nil {
			s := sha256.Sum256(data)
			report["binary_sha256"] = hex.EncodeToString(s[:])
		}
	}
	if head, e := run("git", "rev-parse", "HEAD"); e == nil {
		report["source_commit"] = head
	}
	if err != nil {
		report["error"] = err.Error()
		if data, e := os.ReadFile(filepath.Join(f.dir, "server.log")); e == nil {
			text := string(data)
			for _, secret := range f.secrets {
				text = strings.ReplaceAll(text, secret, "[REDACTED]")
			}
			lines := strings.Split(text, "\n")
			if len(lines) > 45 {
				lines = lines[len(lines)-45:]
			}
			fmt.Fprintln(os.Stderr, strings.Join(lines, "\n"))
		}
	}
	if diff, e := run("git", "diff", "HEAD", "--", "."); e == nil {
		s := sha256.Sum256([]byte(diff))
		report["backend_diff_sha256"] = hex.EncodeToString(s[:])
	}
	harnessHashes := map[string]string{}
	for _, path := range []string{"tools/production-billing-e2e/main.go", "tools/production-billing-e2e/run.sh"} {
		if data, e := os.ReadFile(path); e == nil {
			s := sha256.Sum256(data)
			harnessHashes[path] = hex.EncodeToString(s[:])
		}
	}
	report["harness_sha256"] = harnessHashes
	if f.dir != "" {
		removeErr := os.RemoveAll(f.dir)
		err = errors.Join(err, removeErr)
		report["temporary_credentials_removed"] = removeErr == nil
	}
	report["passed"] = err == nil
	if err != nil {
		report["error"] = err.Error()
	}
	data, marshalErr := json.MarshalIndent(report, "", "  ")
	err = errors.Join(err, marshalErr)
	if e := os.MkdirAll(filepath.Dir(*output), 0755); e != nil {
		err = errors.Join(err, e)
	} else if marshalErr == nil {
		err = errors.Join(err, os.WriteFile(*output, append(data, '\n'), 0644))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("PASS production billing acceptance")
}
