package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// This scenario drives real authenticated HTTP routes and persisted balances.
func (f *fixture) unifiedScenario(admin string) error {
	member, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "unified@example.test", "password": "Unified-fixture-password-2026", "role": "user", "balance": 5, "concurrency": 10})
	if err != nil {
		return err
	}
	jwt, _, err := f.login("unified@example.test", "Unified-fixture-password-2026")
	if err != nil {
		return err
	}
	access, err := f.api("POST", "/api/v1/user/api-access", jwt, map[string]any{})
	if err != nil {
		return err
	}
	key, ok := access["default_key"].(map[string]any)
	if err = f.checked("Existing user receives a prepared default key", ok && key["key"] != ""); err != nil {
		return err
	}
	keyID := key["id"]
	secret, _ := key["key"].(string)
	f.secrets = append(f.secrets, secret)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, e := f.api("POST", "/api/v1/user/api-access", jwt, map[string]any{})
			if e == nil {
				d, _ := a["default_key"].(map[string]any)
				if d["id"] != keyID {
					e = fmt.Errorf("default key changed")
				}
			}
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			return e
		}
	}
	if err = f.checked("Concurrent reconciliation preserves one default key", true); err != nil {
		return err
	}
	var count int
	if err = f.db.QueryRowContext(f.ctx, "SELECT count(*) FROM api_keys WHERE user_id=$1 AND is_default", member["id"]).Scan(&count); err != nil {
		return err
	}
	if err = f.checked("Default identity is unique in PostgreSQL", count == 1); err != nil {
		return err
	}
	group, err := f.api("POST", "/api/v1/admin/groups", admin, map[string]any{"name": "Unified wallet GPT", "platform": "openai", "subscription_type": "standard", "rate_multiplier": 1, "model_pricing": []any{map[string]any{"platform": "openai", "models": []string{"gpt-4o-mini"}, "billing_mode": "per_request", "per_request_price": 1}}})
	if err != nil {
		return err
	}
	if _, err = f.api("POST", "/api/v1/admin/accounts", admin, map[string]any{"name": "Unified GPT provider", "platform": "openai", "type": "apikey", "credentials": map[string]any{"api_key": "fixture-only", "base_url": f.mock.URL, "model_mapping": map[string]string{"gpt-4o-mini": "gpt-4o-mini"}}, "group_ids": []any{group["id"]}, "concurrency": 10, "extra": map[string]any{"openai_ws_enabled": true, "openai_apikey_responses_websockets_v2_mode": "passthrough"}, "upstream_billing_probe_enabled": false}); err != nil {
		return err
	}
	r, err := f.gateway(secret)
	if err != nil {
		return err
	}
	if err = f.checked("Default key reaches the real OpenAI gateway", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Unified key spends signup/promotional funds first", jwt, int64(member["id"].(float64)), "4", "0", "4"); err != nil {
		return err
	}
	if err = f.unifiedBillingScenario(admin, jwt, int64(member["id"].(float64)), secret, group); err != nil {
		return err
	}
	bad, e := f.request("PUT", fmt.Sprintf("/api/v1/keys/%v", keyID), jwt, map[string]any{"group_id": group["id"]}, "")
	if e != nil {
		return e
	}
	if e = f.checked("Unified key cannot be rebound through legacy key editing", bad.Status == 400); e != nil {
		return e
	}
	// The immutable provisioning marker survives soft deletion and process restart.
	if _, err = f.api("DELETE", fmt.Sprintf("/api/v1/keys/%v", keyID), jwt, nil); err != nil {
		return err
	}
	access, err = f.api("POST", "/api/v1/user/api-access", jwt, map[string]any{})
	if err != nil {
		return err
	}
	if err = f.checked("Deleted default key is not recreated", access["state"] == "revoked" && access["default_key"] == nil); err != nil {
		return err
	}
	r, err = f.gateway(secret)
	if err != nil {
		return err
	}
	if err = f.checked("Deleted credential stays rejected", r.Status == 401); err != nil {
		return err
	}
	if err = f.stop(); err != nil {
		return err
	}
	if err = f.start(); err != nil {
		return err
	}
	access, err = f.api("POST", "/api/v1/user/api-access", jwt, map[string]any{})
	if err != nil {
		return err
	}
	if err = f.checked("Default-key deletion survives process restart", access["state"] == "revoked" && access["default_key"] == nil); err != nil {
		return err
	}
	if err = f.unifiedSignupScenario(admin); err != nil {
		return err
	}
	return f.migrations()
}

func (f *fixture) waitSubscription(id any, expected float64) error {
	deadline := time.Now().Add(8 * time.Second)
	for {
		var amount float64
		err := f.db.QueryRowContext(f.ctx, "SELECT daily_usage_usd FROM user_subscriptions WHERE id=$1", id).Scan(&amount)
		if err != nil {
			return err
		}
		if amount == expected {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("subscription %v usage %v; expected %v", id, amount, expected)
		}
		time.Sleep(30 * time.Millisecond)
	}
}
func (f *fixture) unifiedGroup(admin, name, platform, kind, model string, price, rate, limit float64) (map[string]any, error) {
	group, err := f.api("POST", "/api/v1/admin/groups", admin, map[string]any{"name": name, "platform": platform, "subscription_type": kind, "rate_multiplier": rate, "daily_limit_usd": limit, "model_pricing": []any{map[string]any{"platform": platform, "models": []string{model}, "billing_mode": "per_request", "per_request_price": price}}})
	if err != nil {
		return nil, err
	}
	_, err = f.api("POST", "/api/v1/admin/accounts", admin, map[string]any{"name": name + " provider", "platform": platform, "type": "apikey", "credentials": map[string]any{"api_key": "fixture-only", "base_url": f.mock.URL, "model_mapping": map[string]string{model: model}}, "group_ids": []any{group["id"]}, "concurrency": 10, "extra": map[string]any{"openai_ws_enabled": true, "openai_apikey_responses_websockets_v2_mode": "passthrough"}, "upstream_billing_probe_enabled": false})
	return group, err
}
func (f *fixture) unifiedBillingScenario(admin, jwt string, userID int64, key string, walletGroup map[string]any) error {
	legacy, err := f.api("POST", "/api/v1/keys", jwt, map[string]any{"name": "Legacy key stays scoped", "group_id": walletGroup["id"]})
	if err != nil {
		return err
	}
	legacySecret := legacy["key"].(string)
	f.secrets = append(f.secrets, legacySecret)
	soon, err := f.unifiedGroup(admin, "Soon subscription", "openai", "subscription", "gpt-4o-mini", 2, .5, 1)
	if err != nil {
		return err
	}
	later, err := f.unifiedGroup(admin, "Later subscription", "openai", "subscription", "gpt-4o-mini", 4, .5, 1)
	if err != nil {
		return err
	}
	sub1, err := f.api("POST", "/api/v1/admin/subscriptions/assign", admin, map[string]any{"user_id": userID, "group_id": soon["id"], "validity_days": 1})
	if err != nil {
		return err
	}
	sub2, err := f.api("POST", "/api/v1/admin/subscriptions/assign", admin, map[string]any{"user_id": userID, "group_id": later["id"], "validity_days": 2})
	if err != nil {
		return err
	}
	r, err := f.gateway(legacySecret)
	if err != nil {
		return err
	}
	if err = f.checked("Legacy key does not acquire unified subscription billing", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Legacy key retains wallet rate", jwt, userID, "3", "0", "3"); err != nil {
		return err
	}
	f.mu.Lock()
	f.upstreamFailed = true
	f.mu.Unlock()
	failed, e := f.gateway(key)
	f.mu.Lock()
	f.upstreamFailed = false
	f.mu.Unlock()
	if e != nil {
		return e
	}
	if e = f.checked("Provider rejection does not turn into a paid fallback", failed.Status >= 400); e != nil {
		return e
	}
	if e = f.wallet("Failed subscribed request leaves wallet unchanged", jwt, userID, "3", "0", "3"); e != nil {
		return e
	}
	r, err = f.gateway(key)
	if err != nil {
		return err
	}
	if err = f.checked("Unified request chooses a covered subscription", r.Status == 200); err != nil {
		return err
	}
	if err = f.waitSubscription(sub1["id"], 1); err != nil {
		return err
	}
	if err = f.wallet("Earliest-expiry subscription leaves wallet untouched", jwt, userID, "3", "0", "3"); err != nil {
		return err
	}
	if err = f.checked("Earliest subscription reaches its allowance", true); err != nil {
		return err
	}
	r, err = f.gateway(key)
	if err != nil {
		return err
	}
	if err = f.checked("Exhausted subscription yields to another eligible subscription", r.Status == 200); err != nil {
		return err
	}
	if err = f.waitSubscription(sub2["id"], 2); err != nil {
		return err
	}
	if err = f.wallet("Admitted request settles its entire cost to the chosen subscription", jwt, userID, "3", "0", "3"); err != nil {
		return err
	}
	r, err = f.gateway(key)
	if err != nil {
		return err
	}
	if err = f.checked("Exhausted subscriptions automatically continue with wallet", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Wallet continuation uses wallet price, not subscription rate", jwt, userID, "2", "0", "2"); err != nil {
		return err
	}
	// A second provider and API protocol must accept the identical credential.
	_, err = f.unifiedGroup(admin, "Claude wallet", "anthropic", "standard", "claude-sonnet-4-5", .25, 1, 0)
	if err != nil {
		return err
	}
	r, err = f.request("POST", "/v1/messages", key, map[string]any{"model": "claude-sonnet-4-5", "max_tokens": 20, "messages": []any{map[string]any{"role": "user", "content": "fixture"}}}, "")
	if err != nil {
		return err
	}
	if err = f.checked("Identical key reaches Claude Messages on the same API origin", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Claude request uses its own wallet model price", jwt, userID, "1.75", "0", "1.75"); err != nil {
		return err
	}
	r, err = f.request("POST", "/v1/responses", key, map[string]any{"model": "gpt-4o-mini", "input": "fixture", "stream": false}, "")
	if err != nil {
		return err
	}
	if err = f.checked("Identical key reaches GPT Responses after Claude", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("GPT request returns to the correct wallet route", jwt, userID, "0.75", "0", "0.75"); err != nil {
		return err
	}
	// Do not use a real payment provider; verified manual credit exercises the real wallet ledger.
	_, err = f.api("POST", fmt.Sprintf("/api/v1/admin/users/%d/balance", userID), admin, map[string]any{"operation": "add_paid", "balance": 2, "notes": "Verified synthetic E2E payment; no real funds"})
	if err != nil {
		return err
	}
	r, err = f.gateway(key)
	if err != nil {
		return err
	}
	if err = f.checked("One wallet request can spend gifts and purchased credit in order", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Gift exhaustion consumes only the remaining cost from paid credit", jwt, userID, "1.75", "1.75", "0"); err != nil {
		return err
	}
	var receiptCount int
	err = f.db.QueryRowContext(f.ctx, `SELECT count(*) FROM unified_billing_sources WHERE user_id=$1 AND bonus_cost=.75 AND paid_cost=.25`, userID).Scan(&receiptCount)
	if err != nil {
		return err
	}
	if err = f.checked("Settlement receipt records actual gift and paid portions", receiptCount == 1); err != nil {
		return err
	}
	usage, err := f.api("GET", "/api/v1/usage?page_size=100", jwt, nil)
	if err != nil {
		return err
	}
	items, _ := usage["items"].([]any)
	sourceCount := 0
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row["billing_source"] != nil {
			sourceCount++
		}
	}
	if err = f.checked("User usage API returns correlated billing sources", sourceCount >= 6); err != nil {
		return err
	}
	zero, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "zero-unified@example.test", "password": "Zero-user-password-2026", "role": "user", "concurrency": 5})
	if err != nil {
		return err
	}
	zeroJWT, _, err := f.login("zero-unified@example.test", "Zero-user-password-2026")
	if err != nil {
		return err
	}
	zaccess, err := f.api("POST", "/api/v1/user/api-access", zeroJWT, map[string]any{})
	if err != nil {
		return err
	}
	zk := zaccess["default_key"].(map[string]any)["key"].(string)
	f.secrets = append(f.secrets, zk)
	zs, err := f.api("POST", "/api/v1/admin/subscriptions/assign", admin, map[string]any{"user_id": zero["id"], "group_id": soon["id"], "validity_days": 1})
	if err != nil {
		return err
	}
	var beforeWindow, afterWindow string
	windowSQL := `SELECT COALESCE(daily_window_start::text,'')||'|'||COALESCE(weekly_window_start::text,'')||'|'||COALESCE(monthly_window_start::text,'') FROM user_subscriptions WHERE id=$1`
	if err = f.db.QueryRowContext(f.ctx, windowSQL, zs["id"]).Scan(&beforeWindow); err != nil {
		return err
	}
	catalog, e := f.request("GET", "/api/v1/user/api-access/models", zeroJWT, nil, "")
	if e != nil {
		return e
	}
	if err = f.db.QueryRowContext(f.ctx, windowSQL, zs["id"]).Scan(&afterWindow); err != nil {
		return err
	}
	if err = f.checked("Browsing models does not activate or change subscription windows", catalog.Status == 200 && beforeWindow == afterWindow); err != nil {
		return err
	}
	r, err = f.gateway(zk)
	if err != nil {
		return err
	}
	if err = f.checked("Zero-wallet subscriber can call a covered model", r.Status == 200); err != nil {
		return err
	}
	if err = f.waitSubscription(zs["id"], 1); err != nil {
		return err
	}
	r, err = f.gateway(zk)
	if err != nil {
		return err
	}
	if err = f.checked("No subscription allowance and no wallet rejects a new request", r.Status == 403); err != nil {
		return err
	}
	discounted, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "discount-unified@example.test", "password": "Discount-user-password-2026", "role": "user", "balance": 2, "concurrency": 5})
	if err != nil {
		return err
	}
	negotiated, err := f.unifiedGroup(admin, "Negotiated wallet route", "openai", "standard", "gpt-4o-mini", 1, 1, 0)
	if err != nil {
		return err
	}
	if _, err = f.db.ExecContext(f.ctx, `INSERT INTO user_group_rate_multipliers(user_id,group_id,rate_multiplier) VALUES($1,$2,.5)`, discounted["id"], negotiated["id"]); err != nil {
		return err
	}
	dj, _, err := f.login("discount-unified@example.test", "Discount-user-password-2026")
	if err != nil {
		return err
	}
	da, err := f.api("POST", "/api/v1/user/api-access", dj, map[string]any{})
	if err != nil {
		return err
	}
	dk := da["default_key"].(map[string]any)["key"].(string)
	f.secrets = append(f.secrets, dk)
	r, err = f.gateway(dk)
	if err != nil {
		return err
	}
	if err = f.checked("Unified key retains negotiated wallet discount", r.Status == 200); err != nil {
		return err
	}
	if err = f.wallet("Discounted request uses existing user-specific rate", dj, int64(discounted["id"].(float64)), "1.5", "0", "1.5"); err != nil {
		return err
	}
	if err = f.unifiedPriceAndSocketChecks(admin, dj, dk, int64(discounted["id"].(float64))); err != nil {
		return err
	}

	if err = f.unifiedAliasAndPermissionChecks(admin, jwt, key, userID); err != nil {
		return err
	}
	return nil
}

func (f *fixture) unifiedExternal(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasSuffix(r.URL.Path, "/messages") && !strings.HasSuffix(r.URL.Path, "/responses") {
		return false
	}
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return true
		}
		defer conn.CloseNow()
		for {
			_, raw, err := conn.Read(r.Context())
			if err != nil {
				return true
			}
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			f.mu.Lock()
			f.modelCalls++
			id := f.modelCalls
			f.upstreamSuccess++
			f.mu.Unlock()
			event, _ := json.Marshal(map[string]any{"type": "response.completed", "response": map[string]any{"id": fmt.Sprintf("resp-ws-%d", id), "object": "response", "status": "completed", "model": body["model"], "output": []any{}, "usage": map[string]any{"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}}})
			if err = conn.Write(r.Context(), websocket.MessageText, event); err != nil {
				return true
			}
		}
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	f.mu.Lock()
	f.modelCalls++
	id := f.modelCalls
	f.upstreamSuccess++
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-request-id", fmt.Sprintf("unified-upstream-%d", id))
	if strings.HasSuffix(r.URL.Path, "/messages") {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("msg-%d", id), "type": "message", "role": "assistant", "model": body["model"], "content": []any{map[string]any{"type": "text", "text": "fixture answer"}}, "stop_reason": "end_turn", "usage": map[string]any{"input_tokens": 10, "output_tokens": 5}})
	} else {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("resp-%d", id), "object": "response", "status": "completed", "model": body["model"], "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": "fixture answer"}}}}, "usage": map[string]any{"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}})
	}
	return true
}

func (f *fixture) buildArgs() []string {
	args := []string{}
	if f.browserReady != "" {
		args = append(args, "-tags", "embed")
	}
	return append(args, "-o", f.binary, "./cmd/server")
}
func (f *fixture) waitBrowser() error {
	admin, _, err := f.login("owner@production-billing.invalid", f.adminPassword)
	if err != nil {
		return err
	}
	_, err = f.api("POST", "/api/v1/admin/payment/providers", admin, map[string]any{"provider_key": "easypay", "name": "Browser payment fixture", "enabled": true, "supported_types": []string{"alipay"}, "refund_enabled": true, "allow_user_refund": true, "config": map[string]string{"pid": "fixture-merchant", "pkey": "fixture-merchant-key", "apiBase": f.mock.URL, "notifyUrl": f.origin + "/api/v1/payment/webhook/easypay", "returnUrl": f.origin + "/purchase"}})
	if err != nil {
		return err
	}
	_, err = f.api("PUT", "/api/v1/admin/payment/config", admin, map[string]any{"enabled": true, "enabled_payment_types": []string{"alipay"}, "min_amount": 1, "max_amount": 100, "daily_limit": 1000, "balance_recharge_multiplier": 1, "recharge_bonus_mode": "bonus", "recharge_bonus_tiers": []any{map[string]any{"min_amount": 1, "bonus_percent": 20}}})
	if err != nil {
		return err
	}
	group, err := f.api("POST", "/api/v1/admin/groups", admin, map[string]any{"name": "Browser subscription allowance", "platform": "openai", "subscription_type": "subscription", "rate_multiplier": 1, "daily_limit_usd": 5})
	if err != nil {
		return err
	}
	_, err = f.api("POST", "/api/v1/admin/payment/plans", admin, map[string]any{"group_id": group["id"], "name": "Browser monthly plan", "price": 10, "currency": "USD", "validity_days": 30, "validity_unit": "day", "for_sale": true})
	if err != nil {
		return err
	}
	codes, err := f.request("POST", "/api/v1/admin/redeem-codes/generate", admin, map[string]any{"count": 1, "type": "balance", "value": 2}, token())
	if err != nil {
		return err
	}
	var generated struct {
		Data []struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err = json.Unmarshal(codes.Raw, &generated); err != nil || len(generated.Data) != 1 {
		return fmt.Errorf("browser redemption fixture failed: status %d", codes.Status)
	}
	body, _ := json.Marshal(map[string]string{"origin": f.origin, "email": "discount-unified@example.test", "password": "Discount-user-password-2026", "redeem_code": generated.Data[0].Code})
	if err := os.WriteFile(f.browserReady, body, 0600); err != nil {
		return err
	}
	defer os.Remove(f.browserReady)
	defer os.Remove(f.browserReady + ".done")
	fmt.Println("READY browser fixture (private credentials file)")
	deadline := time.Now().Add(15 * time.Minute)
	for {
		if _, err := os.Stat(f.browserReady + ".done"); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("browser fixture timed out")
		}
		select {
		case <-f.ctx.Done():
			return f.ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}
func (f *fixture) unifiedPriceAndSocketChecks(admin, jwt, key string, userID int64) error {
	r, err := f.request("GET", "/api/v1/user/api-access/models", jwt, nil, "")
	if err != nil {
		return err
	}
	var catalog struct {
		Data []struct {
			ID     string `json:"id"`
			Offers []struct {
				Endpoint string  `json:"endpoint"`
				Rate     float64 `json:"rate_multiplier"`
				Pricing  struct {
					PerRequest float64 `json:"per_request_price"`
				} `json:"pricing"`
			} `json:"wallet_offers"`
		}
	}
	if err = json.Unmarshal(r.Raw, &catalog); err != nil {
		return err
	}
	matched := false
	for _, m := range catalog.Data {
		for _, o := range m.Offers {
			if m.ID == "gpt-4o-mini" && o.Endpoint == "responses" && o.Rate == .5 && o.Pricing.PerRequest == 1 {
				matched = true
			}
		}
	}
	if err = f.checked("Displayed wallet price and negotiated rate match settlement", r.Status == 200 && matched); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(f.ctx, 12*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(f.origin, "http://", "ws://", 1)+"/v1/responses", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Bearer " + key}}})
	if err != nil {
		return fmt.Errorf("unified WebSocket handshake: %w", err)
	}
	defer conn.CloseNow()
	for turn := 1; turn <= 2; turn++ {
		body, _ := json.Marshal(map[string]any{"type": "response.create", "model": "gpt-4o-mini", "input": "fixture", "store": false})
		if err = conn.Write(ctx, websocket.MessageText, body); err != nil {
			return err
		}
		complete := false
		for i := 0; i < 20; i++ {
			_, raw, e := conn.Read(ctx)
			if e != nil {
				return fmt.Errorf("WebSocket turn %d: %w", turn, e)
			}
			var event map[string]any
			_ = json.Unmarshal(raw, &event)
			if event["type"] == "response.completed" {
				complete = true
				break
			}
		}
		if err = f.checked(fmt.Sprintf("Unified key supports Responses WebSocket turn %d", turn), complete); err != nil {
			return err
		}
		expected := "1"
		if turn == 2 {
			expected = "0.5"
		}
		if err = f.wallet(fmt.Sprintf("WebSocket turn %d uses negotiated wallet rate", turn), jwt, userID, expected, "0", expected); err != nil {
			return err
		}
	}
	if err = f.unifiedWebSocketTransition(admin); err != nil {
		return err
	}
	return nil
}

func (f *fixture) unifiedSignupScenario(admin string) error {
	_, err := f.api("PUT", "/api/v1/admin/settings", admin, map[string]any{"registration_enabled": true, "email_verify_enabled": false, "default_balance": 5})
	if err != nil {
		return err
	}
	registered, err := f.api("POST", "/api/v1/auth/register", "", map[string]any{"email": "signup-unified@example.test", "password": "Signup-user-password-2026"})
	if err != nil {
		return err
	}
	user, ok := registered["user"].(map[string]any)
	if !ok {
		return fmt.Errorf("registration has no user")
	}
	if err = f.checked("Registration response classifies signup gift without paid funds", money(user["bonus_balance"]).Equal(money("5")) && money(user["paid_balance"]).IsZero()); err != nil {
		return err
	}
	var count int
	err = f.db.QueryRowContext(f.ctx, `SELECT count(*) FROM api_keys WHERE user_id=$1 AND is_default AND access_mode='unified' AND deleted_at IS NULL`, user["id"]).Scan(&count)
	if err != nil {
		return err
	}
	if err = f.checked("Registration provisions the default key before console reconciliation", count == 1); err != nil {
		return err
	}
	if access, _ := registered["access_token"].(string); access != "" {
		f.secrets = append(f.secrets, access)
	}
	return nil
}

// Include untracked implementation files: git diff alone cannot identify a build.
func acceptanceSourceManifest(includeAssets bool) (map[string]string, error) {
	raw, err := run("git", "ls-files", "--cached", "--others", "--exclude-standard", "--", "*.go", "*.sql", "go.mod", "go.sum", "cmd/server/FORK_VERSION", "cmd/server/VERSION")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	hashFile := func(path string) error {
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(data)
		out[path] = hex.EncodeToString(sum[:])
		return nil
	}
	for _, path := range strings.Split(raw, "\n") {
		if path != "" {
			if err = hashFile(path); err != nil {
				return nil, err
			}
		}
	}
	if includeAssets {
		err = filepath.WalkDir("internal/web/dist", func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			return hashFile(path)
		})
	}
	return out, err
}

func (f *fixture) unifiedWebSocketTransition(admin string) error {
	member, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "ws-transition@example.test", "password": "Transition-password-2026", "role": "user", "balance": 2, "concurrency": 5})
	if err != nil {
		return err
	}
	jwt, _, err := f.login("ws-transition@example.test", "Transition-password-2026")
	if err != nil {
		return err
	}
	access, err := f.api("POST", "/api/v1/user/api-access", jwt, map[string]any{})
	if err != nil {
		return err
	}
	key := access["default_key"].(map[string]any)["key"].(string)
	f.secrets = append(f.secrets, key)
	var gid int64
	if err = f.db.QueryRowContext(f.ctx, `SELECT id FROM groups WHERE name='Soon subscription'`).Scan(&gid); err != nil {
		return err
	}
	sub, err := f.api("POST", "/api/v1/admin/subscriptions/assign", admin, map[string]any{"user_id": member["id"], "group_id": gid, "validity_days": 1})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(f.ctx, 12*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, strings.Replace(f.origin, "http://", "ws://", 1)+"/v1/responses", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Bearer " + key}}})
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	body := []byte(`{"type":"response.create","model":"gpt-4o-mini","input":"fixture","store":false}`)
	if err = conn.Write(ctx, websocket.MessageText, body); err != nil {
		return err
	}
	complete := false
	for i := 0; i < 20; i++ {
		_, raw, e := conn.Read(ctx)
		if e != nil {
			return e
		}
		var event map[string]any
		_ = json.Unmarshal(raw, &event)
		if event["type"] == "response.completed" {
			complete = true
			break
		}
	}
	if err = f.checked("WebSocket first turn uses the eligible subscription", complete); err != nil {
		return err
	}
	if err = f.waitSubscription(sub["id"], 1); err != nil {
		return err
	}
	if err = conn.Write(ctx, websocket.MessageText, body); err != nil {
		return err
	}
	_, _, err = conn.Read(ctx)
	if e := f.checked("WebSocket source transition requests a reconnect before another upstream turn", websocket.CloseStatus(err) == websocket.StatusTryAgainLater); e != nil {
		return fmt.Errorf("%w: %v", e, err)
	}
	r, err := f.gateway(key)
	if err != nil {
		return err
	}
	if err = f.checked("Same credential continues with wallet after WebSocket subscription exhaustion", r.Status == 200); err != nil {
		return err
	}
	return f.wallet("Source transition charges the next request once to wallet", jwt, int64(member["id"].(float64)), "1", "0", "1")
}

func (f *fixture) unifiedAliasAndPermissionChecks(admin, jwt, key string, userID int64) error {
	group, err := f.api("POST", "/api/v1/admin/groups", admin, map[string]any{"name": "Public composite alias", "platform": "composite", "subscription_type": "standard", "rate_multiplier": 1, "model_pricing": []any{map[string]any{"platform": "openai", "models": []string{"gpt-4o-mini"}, "billing_mode": "per_request", "per_request_price": .1}}})
	if err != nil {
		return err
	}
	_, err = f.api("POST", fmt.Sprintf("/api/v1/admin/groups/%v/composite-routes", group["id"]), admin, map[string]any{"public_model": "public-fixture-model", "match_type": "exact", "target_platform": "openai", "upstream_model": "gpt-4o-mini", "endpoint": "responses", "enabled": true})
	if err != nil {
		return err
	}
	_, err = f.api("POST", "/api/v1/admin/accounts", admin, map[string]any{"name": "Public alias provider", "platform": "openai", "type": "apikey", "credentials": map[string]any{"api_key": "fixture-only", "base_url": f.mock.URL, "model_mapping": map[string]string{"gpt-4o-mini": "gpt-4o-mini"}}, "group_ids": []any{group["id"]}, "concurrency": 5, "upstream_billing_probe_enabled": false})
	if err != nil {
		return err
	}
	r, err := f.request("POST", "/v1/responses", key, map[string]any{"model": "public-fixture-model", "input": "fixture", "stream": false}, "")
	if err != nil {
		return err
	}
	if err = f.checked("Unified key reuses configured Composite aliases", r.Status == 200); err != nil {
		return err
	}
	catalog, e := f.request("GET", "/api/v1/user/api-access/models", jwt, nil, "")
	if e != nil {
		return e
	}
	var aliasCatalog struct {
		Data []struct {
			ID     string `json:"id"`
			Offers []struct {
				Endpoint string  `json:"endpoint"`
				Rate     float64 `json:"rate_multiplier"`
				Pricing  struct {
					Price float64 `json:"per_request_price"`
				} `json:"pricing"`
			} `json:"wallet_offers"`
		} `json:"data"`
	}
	if err = json.Unmarshal(catalog.Raw, &aliasCatalog); err != nil {
		return err
	}
	aliasPrice := false
	for _, row := range aliasCatalog.Data {
		if row.ID == "public-fixture-model" {
			for _, offer := range row.Offers {
				if offer.Endpoint == "responses" && math.Abs(offer.Rate*offer.Pricing.Price-.1) < 1e-8 {
					aliasPrice = true
				}
			}
		}
	}
	if err = f.checked("Composite alias display uses the actual concrete-model wallet price", aliasPrice); err != nil {
		return err
	}
	if err = f.wallet("Composite alias retains its configured wallet price", jwt, userID, "1.65", "1.65", "0"); err != nil {
		return err
	}
	private, err := f.api("POST", "/api/v1/admin/groups", admin, map[string]any{"name": "Private model fixture", "platform": "openai", "subscription_type": "standard", "rate_multiplier": 1, "is_exclusive": true})
	if err != nil {
		return err
	}
	_, err = f.api("POST", "/api/v1/admin/channels", admin, map[string]any{"name": "Private catalog fixture", "group_ids": []any{private["id"]}, "model_pricing": []any{map[string]any{"platform": "openai", "models": []string{"gpt-private-fixture"}, "billing_mode": "per_request", "per_request_price": 1}}})
	if err != nil {
		return err
	}
	r, err = f.request("GET", "/api/v1/user/api-access/models", jwt, nil, "")
	if err != nil {
		return err
	}
	if err = f.checked("Unified catalog does not disclose private group model names", r.Status == 200 && !strings.Contains(string(r.Raw), "gpt-private-fixture")); err != nil {
		return err
	}
	r, err = f.request("POST", "/v1/responses", key, map[string]any{"model": "gpt-private-fixture", "input": "fixture"}, "")
	if err != nil {
		return err
	}
	if err = f.checked("Unified key does not grant private model access", r.Status == 404); err != nil {
		return err
	}
	r, err = f.request("POST", "/v1/responses", key, map[string]any{"model": "gpt-4o-mini", "Model": "claude-sonnet-4-5", "input": "fixture"}, "")
	if err != nil {
		return err
	}
	return f.checked("Conflicting model fields are rejected before routing", r.Status == 400)
}
