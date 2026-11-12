// Acceptance against disposable PostgreSQL, real handlers and repositories.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type fixtureTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "api-demo.airwallex.com" {
		r = r.Clone(r.Context())
		u := *r.URL
		u.Scheme = t.target.Scheme
		u.Host = t.target.Host
		r.URL = &u
	}
	return t.base.RoundTrip(r)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func check(ok bool, msg string) {
	if !ok {
		panic(msg)
	}
}
func main() {
	ctx := context.Background()
	dsn := os.Getenv("WALLET_E2E_DSN")
	check(strings.Contains(dsn, "/wallet_e2e?"), "requires disposable wallet_e2e database")
	db, err := sql.Open("postgres", dsn)
	must(err)
	defer db.Close()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	must(client.Schema.Create(ctx))
	// The SQL idempotency repository relies on database timestamp defaults from
	// migration 057. Replace only this empty Ent-created fixture table so the
	// production migration supplies its exact schema.
	_, err = db.ExecContext(ctx, `DROP TABLE idempotency_records`)
	must(err)
	legacy, err := client.User.Create().SetEmail("legacy@wallet.invalid").SetPasswordHash("fixture").SetBalance(60).SetFrozenBalance(40).Save(ctx)
	must(err)
	for _, name := range []string{"057_add_idempotency_records.sql", "071_add_usage_billing_dedup.sql", "073_add_usage_billing_dedup_archive.sql"} {
		data, err := os.ReadFile("migrations/" + name)
		must(err)
		_, err = db.ExecContext(ctx, string(data))
		must(err)
	}
	_, err = client.BatchImageJob.Create().SetBatchID("legacy-batch").SetUserID(legacy.ID).SetAPIKeyID(legacy.ID).SetProvider("fixture").SetModel("fixture").SetItemCount(1).SetHoldAmount(40).SetEstimatedCost(40).Save(ctx)
	must(err)
	_, err = db.ExecContext(ctx, `INSERT INTO usage_billing_dedup(request_id,api_key_id,request_fingerprint) VALUES ('batch_image_hold:legacy-batch',$1,'legacy-fixture')`, legacy.ID)
	must(err)
	migration, err := os.ReadFile("migrations/253_wallet_balance_sources.sql")
	must(err)
	// Match the production migration runner's transaction boundary, including
	// rollback when frozen provenance cannot be reconciled.
	_, err = db.ExecContext(ctx, `UPDATE users SET frozen_balance=41 WHERE id=$1`, legacy.ID)
	must(err)
	rejected, err := db.BeginTx(ctx, nil)
	must(err)
	_, err = rejected.ExecContext(ctx, string(migration))
	check(err != nil && strings.Contains(err.Error(), "historical frozen balance"), "migration accepted unmatched frozen funds")
	must(rejected.Rollback())
	var grantTable sql.NullString
	must(db.QueryRowContext(ctx, `SELECT to_regclass('wallet_grants')::text`).Scan(&grantTable))
	check(!grantTable.Valid, "failed migration left wallet tables behind")
	_, err = db.ExecContext(ctx, `UPDATE users SET frozen_balance=40 WHERE id=$1`, legacy.ID)
	must(err)
	migrated, err := db.BeginTx(ctx, nil)
	must(err)
	_, err = migrated.ExecContext(ctx, string(migration))
	must(err)
	must(migrated.Commit())
	conversionMigration, err := os.ReadFile("migrations/254_wallet_paid_to_bonus.sql")
	must(err)
	_, err = db.ExecContext(ctx, string(conversionMigration))
	must(err)
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repository.NewIdempotencyRepository(client, db), service.DefaultIdempotencyConfig()))
	defer service.SetDefaultIdempotencyCoordinator(nil)
	users := repository.NewUserRepository(client, db)
	redeems := service.NewRedeemService(repository.NewRedeemCodeRepository(client), users, nil, nil, nil, client, nil, nil)
	cfg := service.NewPaymentConfigService(client, repository.NewSettingRepository(client), []byte(strings.Repeat("w", 32)))
	var fail atomic.Bool
	var calls atomic.Int64
	var money atomic.Value
	money.Store("")
	var pendingState atomic.Value
	pendingState.Store("ACCEPTED")
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			if strings.HasSuffix(r.URL.Path, "/authentication/login") {
				io.WriteString(w, `{"token":"fixture-token","expires_at":"2099-01-01T00:00:00Z"}`)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/pa/refunds/create") {
				calls.Add(1)
				io.WriteString(w, `{"id":"refund-fixture","status":"ACCEPTED"}`)
				return
			}
			if strings.Contains(r.URL.Path, "/pa/refunds/") {
				fmt.Fprintf(w, `{"id":"refund-fixture","status":%q}`, pendingState.Load())
				return
			}
			panic("unexpected Airwallex fixture endpoint")
		}
		must(r.ParseForm())
		calls.Add(1)
		money.Store(r.FormValue("money"))
		if fail.Load() {
			io.WriteString(w, `{"code":0,"msg":"declined"}`)
		} else {
			io.WriteString(w, `{"code":1,"msg":"ok"}`)
		}
	}))
	defer gateway.Close()
	originalTransport := http.DefaultTransport
	target, err := url.Parse(gateway.URL)
	must(err)
	http.DefaultTransport = fixtureTransport{target, originalTransport}
	defer func() { http.DefaultTransport = originalTransport }()
	configJSON, _ := json.Marshal(map[string]string{"apiBase": gateway.URL, "notifyUrl": "https://fixture.invalid/notify", "returnUrl": "https://fixture.invalid/return", "pid": "wallet-fixture", "pkey": "wallet-fixture"})
	instance, err := client.PaymentProviderInstance.Create().SetProviderKey(payment.TypeEasyPay).SetName("wallet-fixture").SetConfig(string(configJSON)).SetSupportedTypes("alipay").SetEnabled(true).SetAllowUserRefund(true).SetRefundEnabled(true).Save(ctx)
	must(err)
	pay := service.NewPaymentService(client, payment.NewRegistry(), payment.NewDefaultLoadBalancer(client, nil), redeems, nil, cfg, users, nil, nil)
	bill := repository.NewUsageBillingRepository(client, db)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	authConfig := &config.Config{}
	authConfig.JWT.Secret = strings.Repeat("signup-fixture", 4)
	authConfig.JWT.ExpireHour = 1
	authConfig.Default.UserConcurrency = 5
	settingsRepo := repository.NewSettingRepository(client)
	must(settingsRepo.Set(ctx, service.SettingKeyRegistrationEnabled, "true"))
	must(settingsRepo.Set(ctx, service.SettingKeyDefaultBalance, "5"))
	settings := service.NewSettingService(settingsRepo, authConfig)
	auth := service.NewAuthService(client, users, nil, nil, authConfig, settings, nil, nil, nil, nil, nil, nil, nil)
	authHandler := handler.NewAuthHandler(authConfig, auth, nil, settings, nil, nil, nil, nil)
	r.POST("/auth/register", authHandler.Register)
	adminSvc := service.NewAdminService(nil, users, nil, nil, nil, nil, repository.NewRedeemCodeRepository(client), nil, nil, nil, nil, nil, nil, client, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	userHandler := adminhandler.NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
	r.POST("/admin/users/:id/balance", userHandler.UpdateBalance)
	ah := adminhandler.NewPaymentHandler(pay, cfg)
	r.POST("/orders/:id/fulfill", ah.RetryFulfillment)
	r.POST("/orders/:id/refund", ah.ProcessRefund)
	r.POST("/orders/:id/refund/query", func(c *gin.Context) {
		recovered := service.NewPaymentService(client, payment.NewRegistry(), payment.NewDefaultLoadBalancer(client, nil), redeems, nil, cfg, users, nil, nil)
		adminhandler.NewPaymentHandler(recovered, cfg).QueryAndFinalizeRefund(c)
	})
	r.GET("/users/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		must(err)
		u, err := users.GetByID(c.Request.Context(), id)
		must(err)
		c.JSON(200, dto.UserFromService(u))
	})
	api := httptest.NewServer(r)
	defer api.Close()
	request := func(method, path string, body any, keys ...string) map[string]any {
		data, err := json.Marshal(body)
		must(err)
		req, err := http.NewRequest(method, api.URL+path, bytes.NewReader(data))
		must(err)
		req.Header.Set("Content-Type", "application/json")
		if len(keys) > 0 {
			req.Header.Set("Idempotency-Key", keys[0])
		}
		resp, err := http.DefaultClient.Do(req)
		must(err)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		must(err)
		var out map[string]any
		must(json.Unmarshal(raw, &out))
		out["http_status"] = float64(resp.StatusCode)
		return out
	}
	snapshots := []map[string]any{}
	expect := func(label string, id int64, total, paid, bonus, unverified float64) {
		u := request("GET", fmt.Sprintf("/users/%d", id), nil)
		for key, want := range map[string]float64{"balance": total, "paid_balance": paid, "bonus_balance": bonus, "unverified_balance": unverified} {
			got, ok := u[key].(float64)
			check(ok && math.Abs(got-want) < 1e-8, fmt.Sprintf("%s %s got %v want %v", label, key, u[key], want))
		}
		snapshots = append(snapshots, map[string]any{"scenario": label, "total": total, "paid": paid, "bonus": bonus, "unverified": unverified})
	}
	newUser := func(name string) *dbent.User {
		u, err := client.User.Create().SetEmail(name + "@wallet.invalid").SetPasswordHash("fixture").Save(ctx)
		must(err)
		return u
	}
	orderSeq := 0
	order := func(uid int64, principal, bonus, cash float64) *dbent.PaymentOrder {
		orderSeq++
		ref := fmt.Sprintf("wallet-%d", orderSeq)
		o, err := client.PaymentOrder.Create().SetUserID(uid).SetUserEmail("wallet@fixture.invalid").SetUserName("wallet fixture").SetAmount(principal + bonus).SetBonusAmount(bonus).SetPayAmount(cash).SetFeeRate(0).SetRechargeCode(ref).SetOutTradeNo(ref).SetPaymentType(payment.TypeAlipay).SetPaymentTradeNo(ref).SetOrderType(payment.OrderTypeBalance).SetStatus(service.OrderStatusPaid).SetPaidAt(time.Now()).SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("fixture.invalid").SetProviderInstanceID(strconv.FormatInt(instance.ID, 10)).Save(ctx)
		must(err)
		out := request("POST", fmt.Sprintf("/orders/%d/fulfill", o.ID), map[string]any{})
		check(out["http_status"] == float64(200), fmt.Sprint(out))
		return o
	}
	refund := func(o *dbent.PaymentOrder, amount float64) map[string]any {
		return request("POST", fmt.Sprintf("/orders/%d/refund", o.ID), map[string]any{"amount": amount, "deduct_balance": true, "force": false})
	}
	success := func(out map[string]any) bool { d, _ := out["data"].(map[string]any); return d["success"] == true }
	spend := func(uid int64, id string, amount float64) {
		_, err := bill.Apply(ctx, &service.UsageBillingCommand{RequestID: id, APIKeyID: uid, UserID: uid, BalanceCost: amount})
		must(err)
	}
	// Check the first registration response, before a subsequent database read
	// can hide a missing refresh of trigger-maintained accounting fields.
	for _, amount := range []float64{5, 0} {
		must(settingsRepo.Set(ctx, service.SettingKeyDefaultBalance, strconv.FormatFloat(amount, 'f', -1, 64)))
		out := request("POST", "/auth/register", map[string]any{"email": fmt.Sprintf("registered-%g@wallet.invalid", amount), "password": "WalletFixture123!"})
		check(out["http_status"] == float64(200), fmt.Sprint(out))
		data := out["data"].(map[string]any)
		user := data["user"].(map[string]any)
		check(user["balance"] == amount && user["bonus_balance"] == amount && user["paid_balance"] == float64(0) && user["unverified_balance"] == float64(0), fmt.Sprint("registration response misclassifies signup credit: ", user))
		uid := int64(user["id"].(float64))
		expect(fmt.Sprintf("registration response and persisted signup gift %g", amount), uid, amount, 0, amount, 0)
		if amount > 0 {
			spend(uid, "signup-gift-spend", 2)
			expect("signup gift remains promotional after consumption", uid, 3, 0, 3, 0)
		}
	}
	expect("legacy preserved", legacy.ID, 60, 0, 0, 60)
	_, err = bill.ReleaseBatchImageBalance(ctx, &service.BatchImageBalanceHoldCommand{RequestID: service.BatchImageReleaseRequestID("legacy-batch"), APIKeyID: legacy.ID, UserID: legacy.ID, BatchID: "legacy-batch", HoldAmount: 40})
	must(err)
	expect("historical frozen funds remain unverified", legacy.ID, 100, 0, 0, 100)
	spend(legacy.ID, "legacy-spend", 10)
	expect("legacy spendable", legacy.ID, 90, 0, 0, 90)
	u := newUser("primary")
	a := order(u.ID, 100, 20, 70)
	expect("payment split", u.ID, 120, 100, 20, 0)
	beforeInvalid := calls.Load()
	for _, amount := range []float64{120, 0.000000001, 0.00000001} {
		check(!success(refund(a, amount)), "invalid refund amount accepted")
	}
	check(calls.Load() == beforeInvalid, "invalid amount reached provider")
	must(pay.ExecuteBalanceFulfillment(ctx, a.ID))
	expect("fulfillment replay", u.ID, 120, 100, 20, 0)
	must(users.UpdateBalance(ctx, u.ID, 10))
	spend(u.ID, "gift-first", 15)
	expect("gift first", u.ID, 115, 100, 15, 0)
	out := refund(a, 100)
	check(success(out), fmt.Sprint("full refund failed: ", out))
	check(money.Load() == "70.00", "gateway must refund actual cash")
	expect("reclaim only order remainder", u.ID, 10, 0, 10, 0)
	count := calls.Load()
	check(!success(refund(a, 100)), "duplicate refund accepted")
	check(calls.Load() == count, "duplicate gateway refund")
	v := newUser("failure")
	b := order(v.ID, 100, 20, 100)
	fail.Store(true)
	check(!success(refund(b, 100)), "failed gateway marked successful")
	expect("failed refund exact restore", v.ID, 120, 100, 20, 0)
	fail.Store(false)
	check(success(refund(b, 40)), "partial refund failed")
	expect("partial refund reclaims remaining order gift", v.ID, 60, 60, 0, 0)
	check(money.Load() == "40.00", "partial cash denominator includes gift")
	g := newUser("gift-only")
	must(users.UpdateBalance(ctx, g.ID, 50))
	c := order(g.ID, 10, 0, 10)
	spend(g.ID, "spend-gift-and-paid", 55)
	before := calls.Load()
	check(!success(refund(c, 10)), "gift funded refund")
	check(calls.Load() == before, "invalid refund reached gateway")
	expect("gift cannot refund", g.ID, 5, 5, 0, 0)
	check(!success(request("POST", fmt.Sprintf("/orders/%d/refund", c.ID), map[string]any{"amount": 10, "deduct_balance": true, "force": true})), "forced gift refund accepted")
	check(!success(request("POST", fmt.Sprintf("/orders/%d/refund", c.ID), map[string]any{"amount": 10, "deduct_balance": false, "force": true})), "nondeducting refund accepted")
	check(calls.Load() == before, "refund bypass reached provider")
	h := newUser("batch")
	d := order(h.ID, 100, 20, 100)
	hold := &service.BatchImageBalanceHoldCommand{RequestID: service.BatchImageHoldRequestID("one"), APIKeyID: h.ID, UserID: h.ID, BatchID: "one", HoldAmount: 40}
	_, err = bill.ReserveBatchImageBalance(ctx, hold)
	must(err)
	expect("batch source reserve", h.ID, 80, 80, 0, 0)
	check(!success(refund(d, 50)), "refunded a gift reserved by a batch")
	hold.RequestID = service.BatchImageReleaseRequestID("one")
	_, err = bill.ReleaseBatchImageBalance(ctx, hold)
	must(err)
	expect("batch restores exact sources", h.ID, 120, 100, 20, 0)
	hold.BatchID = "two"
	hold.RequestID = service.BatchImageHoldRequestID("two")
	_, err = bill.ReserveBatchImageBalance(ctx, hold)
	must(err)
	hold.RequestID = service.BatchImageCaptureRequestID("two")
	hold.ActualAmount = 10
	_, err = bill.CaptureBatchImageBalance(ctx, hold)
	must(err)
	expect("batch settles gifts first", h.ID, 110, 100, 10, 0)
	_, err = bill.CaptureBatchImageBalance(ctx, hold)
	must(err)
	expect("batch settlement replay", h.ID, 110, 100, 10, 0)
	concurrent := newUser("concurrent")
	e := order(concurrent.ID, 100, 20, 100)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := bill.Apply(ctx, &service.UsageBillingCommand{RequestID: fmt.Sprintf("parallel-%d", i), APIKeyID: concurrent.ID, UserID: concurrent.ID, BalanceCost: 1})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(err)
	}
	expect("concurrent gifts exhausted exactly once", concurrent.ID, 100, 100, 0, 0)
	check(success(refund(e, 100)), "concurrent user refund failed")
	expect("consumed gift not reclaimed twice", concurrent.ID, 0, 0, 0, 0)

	// Legacy reconciliation preserves totals and writes the operator's evidence.
	historical, err := client.PaymentOrder.Create().SetUserID(legacy.ID).SetUserEmail(legacy.Email).SetUserName("historical").SetAmount(40).SetBonusAmount(10).SetPayAmount(30).SetFeeRate(0).SetRechargeCode("historical-reviewed").SetOutTradeNo("historical-reviewed").SetPaymentType(payment.TypeAlipay).SetPaymentTradeNo("historical-reviewed").SetOrderType(payment.OrderTypeBalance).SetStatus(service.OrderStatusCompleted).SetPaidAt(time.Now()).SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("fixture.invalid").SetProviderInstanceID(strconv.FormatInt(instance.ID, 10)).Save(ctx)
	must(err)
	check(!success(refund(historical, 30)), "unverified funds funded historical refund")
	_, err = db.ExecContext(ctx, `SELECT wallet_verify_legacy($1,30,10,'fixture funding evidence',$2)`, legacy.ID, historical.ID)
	must(err)
	expect("legacy explicitly verified", legacy.ID, 90, 30, 10, 50)
	_, err = db.ExecContext(ctx, `SELECT wallet_verify_legacy($1,60,0,'invalid excess',NULL)`, legacy.ID)
	check(err != nil, "excess legacy verification accepted")
	expect("invalid verification rolled back", legacy.ID, 90, 30, 10, 50)
	check(success(refund(historical, 30)), "verified historical funds could not refund")
	expect("verified historical order refund reclaims its gift", legacy.ID, 50, 0, 0, 50)
	// Public/admin vouchers, initial signup credit and SQL writers remain gifts.
	signup, err := client.User.Create().SetEmail("signup@wallet.invalid").SetPasswordHash("fixture").SetBalance(8).Save(ctx)
	must(err)
	expect("signup credit is promotional", signup.ID, 8, 0, 8, 0)
	rc := &service.RedeemCode{Code: "wallet-admin-gift", Type: service.RedeemTypeBalance, Value: 7, Status: service.StatusUnused}
	must(redeems.CreateCode(ctx, rc))
	_, err = redeems.RedeemForAdminFulfillment(ctx, signup.ID, rc.Code)
	must(err)
	expect("admin voucher is promotional", signup.ID, 15, 0, 15, 0)
	out = request("POST", fmt.Sprintf("/admin/users/%d/balance", signup.ID), map[string]any{"operation": "add", "balance": 5, "notes": "campaign gift"})
	check(out["http_status"] == float64(200), fmt.Sprint(out))
	expect("manual campaign credit", signup.ID, 20, 0, 20, 0)
	out = request("POST", fmt.Sprintf("/admin/users/%d/balance", signup.ID), map[string]any{"operation": "add_paid", "balance": 10, "notes": ""})
	check(out["http_status"] == float64(400), fmt.Sprint(out))
	expect("manual paid credit requires evidence", signup.ID, 20, 0, 20, 0)
	out = request("POST", fmt.Sprintf("/admin/users/%d/balance", signup.ID), map[string]any{"operation": "add_paid", "balance": 10, "notes": "bank receipt fixture"})
	check(out["http_status"] == float64(200), fmt.Sprint(out))
	expect("manual verified paid credit", signup.ID, 30, 10, 20, 0)
	tiny := &service.BatchImageBalanceHoldCommand{RequestID: service.BatchImageHoldRequestID("tiny"), APIKeyID: signup.ID, UserID: signup.ID, BatchID: "tiny", HoldAmount: 0.000000001}
	_, err = bill.ReserveBatchImageBalance(ctx, tiny)
	must(err)
	tiny.RequestID = service.BatchImageReleaseRequestID("tiny")
	_, err = bill.ReleaseBatchImageBalance(ctx, tiny)
	must(err)
	expect("sub-scale batch reservation is stable", signup.ID, 30, 10, 20, 0)
	// Two orders compete for one remaining paid balance. Only one gateway refund
	// may be created, even if both preflight reads initially observe enough money.
	race := newUser("refund-race")
	raceA := order(race.ID, 100, 20, 100)
	raceB := order(race.ID, 100, 20, 100)
	spend(race.ID, "race-spend", 140)
	beforeRace := calls.Load()
	raceResults := make(chan bool, 2)
	for _, o := range []*dbent.PaymentOrder{raceA, raceB} {
		wg.Add(1)
		go func(o *dbent.PaymentOrder) { defer wg.Done(); raceResults <- success(refund(o, 100)) }(o)
	}
	wg.Wait()
	close(raceResults)
	wins := 0
	for ok := range raceResults {
		if ok {
			wins++
		}
	}
	check(wins == 1 && calls.Load()-beforeRace == 1, "concurrent refunds exceeded paid credit")
	expect("concurrent refund uses paid balance once", race.ID, 0, 0, 0, 0)
	airConfig, _ := json.Marshal(map[string]string{"apiBase": "https://api-demo.airwallex.com/api/v1", "clientId": "wallet-fixture", "apiKey": "wallet-fixture", "webhookSecret": "wallet-fixture"})
	air, err := client.PaymentProviderInstance.Create().SetProviderKey(payment.TypeAirwallex).SetName("wallet-pending").SetConfig(string(airConfig)).SetSupportedTypes("airwallex").SetEnabled(true).SetAllowUserRefund(true).SetRefundEnabled(true).Save(ctx)
	must(err)
	for _, settled := range []string{"SETTLED", "FAILED"} {
		pendingUser := newUser("pending-" + strings.ToLower(settled))
		po := order(pendingUser.ID, 100, 20, 70)
		_, err = client.PaymentOrder.UpdateOneID(po.ID).SetProviderInstanceID(strconv.FormatInt(air.ID, 10)).SetPaymentType(payment.TypeAirwallex).Save(ctx)
		must(err)
		beforePending := calls.Load()
		out := refund(po, 100)
		check(!success(out) && out["http_status"] == float64(200), fmt.Sprint("pending refund failed: ", out))
		expect("pending reserves both sources "+settled, pendingUser.ID, 0, 0, 0, 0)
		pendingState.Store("ACCEPTED")
		out = request("POST", fmt.Sprintf("/orders/%d/refund/query", po.ID), map[string]any{})
		check(!success(out), "pending query settled too early")
		check(calls.Load() == beforePending+1, "pending query created another refund")
		pendingState.Store(settled)
		out = request("POST", fmt.Sprintf("/orders/%d/refund/query", po.ID), map[string]any{})
		if settled == "SETTLED" {
			check(success(out), fmt.Sprint(out))
			expect("restart finalizes held refund", pendingUser.ID, 0, 0, 0, 0)
		} else {
			check(!success(out), fmt.Sprint(out))
			expect("restart restores failed pending refund", pendingUser.ID, 120, 100, 20, 0)
		}
		request("POST", fmt.Sprintf("/orders/%d/refund/query", po.ID), map[string]any{})
		check(calls.Load() == beforePending+1, "replayed finalization created another refund")
	}
	separate := newUser("separate-orders")
	firstOrder := order(separate.ID, 100, 10, 100)
	order(separate.ID, 50, 5, 50)
	spend(separate.ID, "separate-order-spend", 8)
	check(success(refund(firstOrder, 100)), "order reclamation failed")
	expect("refund preserves another orders gift", separate.ID, 55, 50, 5, 0)
	// Conversion preserves total funds and permanently reduces refundable paid
	// credit. Requests use the real PostgreSQL idempotency repository.
	conversion := newUser("conversion")
	conversionOrder := order(conversion.ID, 100, 20, 100)
	conversionPath := fmt.Sprintf("/admin/users/%d/balance", conversion.ID)
	convert := func(id int64, amount float64, reason string, keys ...string) map[string]any {
		return request("POST", fmt.Sprintf("/admin/users/%d/balance", id), map[string]any{"operation": "convert_paid_to_bonus", "balance": amount, "notes": reason}, keys...)
	}
	cashBeforeConversion := calls.Load()
	converted := convert(conversion.ID, 40, "campaign reclassification", "conversion-once")
	check(converted["http_status"] == float64(200), fmt.Sprint(converted))
	expect("conversion preserves total", conversion.ID, 120, 60, 60, 0)
	converted = convert(conversion.ID, 40, "campaign reclassification", "conversion-once")
	check(converted["http_status"] == float64(200), fmt.Sprint(converted))
	expect("conversion replay applies once", conversion.ID, 120, 60, 60, 0)
	converted = request("POST", conversionPath, map[string]any{"operation": "convert_paid_to_bonus", "balance": 10, "notes": "changed request"}, "conversion-once")
	check(converted["http_status"] == float64(409), "conversion key accepted a different request")
	for _, invalid := range []struct {
		amount float64
		reason string
	}{{61, "exceeds paid"}, {1, " "}, {0, "zero"}, {-1, "negative"}, {0.000000001, "sub-scale"}} {
		check(convert(conversion.ID, invalid.amount, invalid.reason)["http_status"] == float64(400), "invalid conversion accepted")
	}
	expect("invalid conversions leave sources unchanged", conversion.ID, 120, 60, 60, 0)
	check(calls.Load() == cashBeforeConversion, "conversion called the payment provider")
	var journalCount int
	var recharged float64
	var convertedAmount float64
	var convertedDelta float64
	var evidence string
	must(db.QueryRow(`SELECT COUNT(*) FROM wallet_journal WHERE user_id=$1 AND operation='convert_paid_to_bonus'`, conversion.ID).Scan(&journalCount))
	check(journalCount == 1, "conversion produced duplicate journal records")
	must(db.QueryRow(`SELECT (allocations->0->>'amount')::numeric,delta,reference FROM wallet_journal WHERE user_id=$1 AND operation='convert_paid_to_bonus'`, conversion.ID).Scan(&convertedAmount, &convertedDelta, &evidence))
	check(convertedAmount == 40 && convertedDelta == 0 && evidence == "campaign reclassification", "conversion audit missing amount or reason")
	must(db.QueryRow(`SELECT total_recharged FROM users WHERE id=$1`, conversion.ID).Scan(&recharged))
	check(recharged == 100, "conversion changed cumulative recharge")
	check(!success(refund(conversionOrder, 100)), "converted credit was refundable")
	check(success(refund(conversionOrder, 60)), "remaining paid funds could not refund")
	expect("refund excludes converted gift", conversion.ID, 40, 0, 40, 0)
	spend(conversion.ID, "converted-gift-spend", 10)
	expect("converted gift remains spendable", conversion.ID, 30, 0, 30, 0)
	check(convert(legacy.ID, 1, "unverified funds")["http_status"] == float64(400), "unverified credit converted as paid")
	frozenConversion := newUser("frozen-conversion")
	order(frozenConversion.ID, 100, 20, 100)
	frozenCmd := &service.BatchImageBalanceHoldCommand{RequestID: service.BatchImageHoldRequestID("conversion-held"), APIKeyID: frozenConversion.ID, UserID: frozenConversion.ID, BatchID: "conversion-held", HoldAmount: 50}
	_, err = bill.ReserveBatchImageBalance(ctx, frozenCmd)
	must(err)
	check(convert(frozenConversion.ID, 71, "frozen funds excluded")["http_status"] == float64(400), "conversion used frozen paid funds")
	check(convert(frozenConversion.ID, 60, "convert available funds")["http_status"] == float64(200), "available paid conversion failed")
	expect("conversion excludes frozen funds", frozenConversion.ID, 70, 10, 60, 0)
	var frozenAmount float64
	must(db.QueryRow(`SELECT frozen_balance FROM users WHERE id=$1`, frozenConversion.ID).Scan(&frozenAmount))
	check(frozenAmount == 50, "conversion changed frozen balance")
	frozenCmd.RequestID = service.BatchImageReleaseRequestID("conversion-held")
	_, err = bill.ReleaseBatchImageBalance(ctx, frozenCmd)
	must(err)
	expect("release retains converted source", frozenConversion.ID, 120, 40, 80, 0)
	concurrentConversion := newUser("concurrent-conversion")
	order(concurrentConversion.ID, 100, 0, 100)
	conversionResults := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conversionResults <- convert(concurrentConversion.ID, 75, "parallel conversion")["http_status"] == float64(200)
		}()
	}
	wg.Wait()
	close(conversionResults)
	conversionWins := 0
	for ok := range conversionResults {
		if ok {
			conversionWins++
		}
	}
	check(conversionWins == 1, "parallel conversions overspent paid funds")
	expect("concurrent conversions use paid funds once", concurrentConversion.ID, 100, 25, 75, 0)
	refundConversion := newUser("refund-conversion")
	refundConversionOrder := order(refundConversion.ID, 100, 0, 100)
	races := make(chan bool, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		races <- convert(refundConversion.ID, 80, "race refund")["http_status"] == float64(200)
	}()
	go func() { defer wg.Done(); races <- success(refund(refundConversionOrder, 100)) }()
	wg.Wait()
	close(races)
	raceWins := 0
	for ok := range races {
		if ok {
			raceWins++
		}
	}
	check(raceWins == 1, "conversion and refund spent the same funds")
	raceUser, err := users.GetByID(ctx, refundConversion.ID)
	must(err)
	check((raceUser.Balance == 100 && raceUser.BonusBalance == 80) || (raceUser.Balance == 0 && raceUser.BonusBalance == 0), "conversion refund race broke conservation")
	// Inject a journal failure after grant creation and summary refresh. The
	// function statement must roll every money mutation back together.
	journalFailure := newUser("conversion-journal-failure")
	order(journalFailure.ID, 100, 0, 100)
	_, err = db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE wallet_journal ADD CONSTRAINT conversion_fixture_failure CHECK(operation<>'convert_paid_to_bonus' OR user_id<>%d)`, journalFailure.ID))
	must(err)
	check(convert(journalFailure.ID, 40, "journal must be atomic")["http_status"] == float64(500), "journal failure did not stop conversion")
	_, err = db.ExecContext(ctx, `ALTER TABLE wallet_journal DROP CONSTRAINT conversion_fixture_failure`)
	must(err)
	expect("failed journal rolls back conversion", journalFailure.ID, 100, 100, 0, 0)
	var failedGrants int
	must(db.QueryRow(`SELECT COUNT(*) FROM wallet_grants WHERE user_id=$1`, journalFailure.ID).Scan(&failedGrants))
	check(failedGrants == 0, "failed conversion left a promotional grant")
	usageConversion := newUser("usage-conversion")
	order(usageConversion.ID, 100, 0, 100)
	usageConverted := make(chan bool, 1)
	wg.Add(2)
	go func() {
		defer wg.Done()
		usageConverted <- convert(usageConversion.ID, 80, "race usage")["http_status"] == float64(200)
	}()
	go func() { defer wg.Done(); spend(usageConversion.ID, "conversion-usage-race", 60) }()
	wg.Wait()
	if <-usageConverted {
		expect("conversion and consumption conserve funds", usageConversion.ID, 40, 20, 20, 0)
	} else {
		expect("consumption limits available conversion", usageConversion.ID, 40, 40, 0, 0)
	}
	var mismatch int
	must(db.QueryRow(`SELECT COUNT(*) FROM users u WHERE bonus_balance<>(SELECT COALESCE(SUM(remaining),0) FROM wallet_grants g WHERE g.user_id=u.id AND source='bonus') OR unverified_balance<>(SELECT COALESCE(SUM(remaining),0) FROM wallet_grants g WHERE g.user_id=u.id AND source='unverified')`).Scan(&mismatch))
	check(mismatch == 0, "grant summary mismatch")
	sources := map[string]string{}
	for _, name := range []string{"migrations/253_wallet_balance_sources.sql", "migrations/254_wallet_paid_to_bonus.sql", "../frontend/src/components/admin/user/UserBalanceModal.vue", "../frontend/src/api/admin/users.ts", "../frontend/src/views/admin/UsersView.vue", "internal/service/payment_wallet.go", "internal/service/payment_refund.go", "internal/service/redeem_service.go", "internal/repository/usage_billing_repo.go", "internal/service/admin_user.go", "internal/service/user_service.go", "internal/repository/user_repo.go", "internal/handler/admin/user_handler.go", "internal/service/auth_service.go", "internal/handler/auth_handler.go", "internal/handler/dto/mappers.go", "../frontend/src/i18n/locales/zh/admin/settings.ts", "../frontend/src/i18n/locales/en/admin/settings.ts", "tools/wallet-e2e/main.go"} {
		raw, err := os.ReadFile(name)
		must(err)
		sum := sha256.Sum256(raw)
		sources[name] = hex.EncodeToString(sum[:])
	}
	receipt := map[string]any{"source_sha256": sources, "passed": true, "migration_mismatch_rejected_transactionally": true, "database": "PostgreSQL disposable fixture", "gateway": "local HTTP fixture", "scenarios": snapshots, "gateway_refund_calls": calls.Load(), "limits": []string{"synthetic user identity", "no live payment provider", "no browser UI validation"}}
	data, err := json.MarshalIndent(receipt, "", "  ")
	must(err)
	must(os.MkdirAll("../.planning/2026-11-11-balance-sources/artifacts", 0755))
	must(os.WriteFile("../.planning/2026-11-11-balance-sources/artifacts/wallet-e2e.json", append(data, '\n'), 0644))
	fmt.Println(string(data))
}
