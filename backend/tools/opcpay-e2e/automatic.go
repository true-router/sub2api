package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Only the merchant is simulated. Handlers, scheduling and wallet transactions are real.
type automaticMerchant struct {
	mu             sync.Mutex
	paid           map[string]string
	quota          map[string]float64
	results        map[string]map[string]any
	debits         int
	lose, mismatch bool
	invalidProof   string
	forceTradeNo   string
}

func newAutomaticMerchant() *automaticMerchant {
	return &automaticMerchant{paid: map[string]string{}, quota: map[string]float64{}, results: map[string]map[string]any{}}
}
func (m *automaticMerchant) serve(w http.ResponseWriter, r *http.Request, body map[string]any) bool {
	path := strings.TrimPrefix(r.URL.Path, "/opcpay/api/pay/v1")
	if path != "/transactions/query" && path != "/access" && path != "/consumption" && path != "/consume" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	user, _ := body["user_id"].(string)
	key, _ := body["idempotency_key"].(string)
	send := func(v any) { must(json.NewEncoder(w).Encode(v)) }
	switch path {
	case "/transactions/query":
		status := m.paid[key]
		if status == "" {
			status = "PENDING"
		}
		result := map[string]any{"user_id": user, "idempotency_key": key, "status": status, "trade_no": "fixture-trade-" + key, "checked_at": time.Now().UTC().Format(time.RFC3339), "status_source": "ALIPAY_PURCHASE_QUERY_V3"}
		if m.mismatch {
			result["user_id"] = "unrelated-user"
		}
		if m.invalidProof != "" {
			result[m.invalidProof] = ""
		}
		if m.forceTradeNo != "" {
			result["trade_no"] = m.forceTradeNo
		}
		send(result)
	case "/access":
		amount, _ := body["amount"].(float64)
		send(map[string]any{"access_granted": m.quota[user] >= amount, "requires_consumption": true, "remaining": m.quota[user], "quota_unit": "POINT"})
	case "/consumption":
		if result := m.results[key]; result != nil {
			send(result)
		} else {
			send(map[string]any{"status": "UNKNOWN", "confirmed": false, "idempotency_key": key})
		}
	case "/consume":
		if m.results[key] == nil {
			amount := body["amount"].(float64)
			if amount > m.quota[user] {
				panic("consumed unavailable quota")
			}
			m.quota[user] -= amount
			m.debits++
			m.results[key] = map[string]any{"status": "SUCCEEDED", "confirmed": true, "idempotency_key": key, "consumed": amount, "quota_unit": "POINT"}
		}
		if m.lose {
			m.lose = false
			w.WriteHeader(503)
			send(map[string]string{"error": "lost response"})
			return true
		}
		send(m.results[key])
	}
	return true
}
func verifyAutomaticRecharge(db *sql.DB, client *dbent.Client, cfg *service.PaymentConfigService, request func(string, string, string, any) (int, map[string]any), check func(bool, string), merchant *automaticMerchant) {
	ctx := context.Background()
	var checks []string
	originalCheck := check
	check = func(ok bool, name string) { originalCheck(ok, name); checks = append(checks, name) }
	paymentSvc := service.NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, cfg, nil, nil, nil)
	create := func(session, action string) string {
		status, body := request(session, "POST", "/operations", map[string]any{"kind": "purchase", "action_id": action, "sku_id": "actual-fixture-sku"})
		check(status == 200, "create purchase: "+action)
		return body["data"].(map[string]any)["business_id"].(string)
	}
	balance := func(user int) float64 {
		var n float64
		must(db.QueryRow(`SELECT balance FROM users WHERE id=$1`, user).Scan(&n))
		return n
	}
	due := func() { _, err := db.Exec(`UPDATE opcpay_operations SET next_reconcile_at=NOW()`); must(err) }
	cycle := func() { due(); _, err := paymentSvc.OPCPay().Reconcile(ctx); must(err) }
	paid := func(id, user string) {
		merchant.mu.Lock()
		merchant.paid[id] = "PAID"
		merchant.quota[user] += 10
		merchant.mu.Unlock()
	}
	first := create("one", "auto_closed_000001")
	status, _ := request("one", "POST", "/operations/"+first+"/close", nil)
	check(status == 200, "closing the dialog succeeds before payment")
	paid(first, "1")
	// No user request follows the payment. Only the actual scheduled worker runs.
	worker := service.NewPaymentOrderExpiryService(paymentSvc, 20*time.Millisecond)
	worker.SetLeaderLock(nil, db)
	worker.Start()
	deadline := time.Now().Add(8 * time.Second)
	for balance(1) != 10 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	worker.Stop()
	check(balance(1) == 10 && balance(2) == 0, "background scheduler credits a paid closed purchase without any user request")
	status, body := request("one", "GET", "/operations/"+first, nil)
	check(status == 200 && body["data"].(map[string]any)["status"] == "COMPLETED", "original purchase ID reports automatic delivery")
	status, _ = request("two", "GET", "/operations/"+first, nil)
	check(status != 200, "another user cannot inspect the purchase")
	second := create("two", "auto_lost_00000001")
	paid(second, "2")
	merchant.mu.Lock()
	merchant.lose = true
	merchant.mu.Unlock()
	cycle()
	check(balance(2) == 0, "lost consumption response does not guess delivery")
	var originalID string
	must(db.QueryRow(`SELECT business_id FROM opcpay_operations WHERE user_id=2 AND kind='consume' AND action_id='auto_lost_00000001'`).Scan(&originalID))
	paymentSvc = service.NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, cfg, nil, nil, nil)
	cycle()
	check(balance(2) == 10, "recreated service recovers offline consumption")
	merchant.mu.Lock()
	debits := merchant.debits
	merchant.mu.Unlock()
	check(debits == 2, "lost response recovery does not debit twice")
	unpaid := create("one", "auto_unpaid_000001")
	merchant.mu.Lock()
	merchant.quota["1"] = 10
	merchant.mu.Unlock()
	cycle()
	check(balance(1) == 10, "available quota does not prove this purchase was paid")
	merchant.mu.Lock()
	merchant.paid[unpaid] = "UNKNOWN"
	merchant.mu.Unlock()
	cycle()
	check(balance(1) == 10, "unknown transaction is left for reconciliation")
	merchant.mu.Lock()
	merchant.paid[unpaid] = "REFUNDED"
	merchant.mu.Unlock()
	cycle()
	check(balance(1) == 10, "refunded transaction is not credited")
	merchant.mu.Lock()
	merchant.paid[unpaid] = "PAID"
	merchant.mismatch = true
	merchant.mu.Unlock()
	due()
	_, mismatchErr := paymentSvc.OPCPay().Reconcile(ctx)
	check(mismatchErr != nil, "mismatched payment proof reports a deferred reconciliation")
	check(balance(1) == 10, "mismatched transaction identity cannot consume quota")
	merchant.mu.Lock()
	merchant.mismatch = false
	merchant.mu.Unlock()
	for _, field := range []string{"idempotency_key", "checked_at", "status_source", "trade_no"} {
		merchant.mu.Lock()
		merchant.invalidProof = field
		merchant.mu.Unlock()
		due()
		_, proofErr := paymentSvc.OPCPay().Reconcile(ctx)
		check(proofErr != nil && balance(1) == 10, "incomplete payment proof cannot credit: "+field)
	}
	merchant.mu.Lock()
	merchant.invalidProof = ""
	merchant.mu.Unlock()
	_, err := db.Exec(`CREATE FUNCTION reject_auto_credit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'delivery unavailable'; END; $$; CREATE TRIGGER reject_auto_credit BEFORE UPDATE OF balance ON users FOR EACH ROW EXECUTE FUNCTION reject_auto_credit()`)
	must(err)
	due()
	paymentSvc.OPCPay().Reconcile(ctx)
	check(balance(1) == 10, "wallet failure does not expose uncommitted credit")
	var confirmed string
	must(db.QueryRow(`SELECT status FROM opcpay_operations WHERE user_id=1 AND action_id='auto_unpaid_000001' AND kind='consume'`).Scan(&confirmed))
	check(confirmed == "CONFIRMED", "upstream confirmation persists for background recovery")
	_, err = db.Exec(`DROP TRIGGER reject_auto_credit ON users`)
	must(err)
	due()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); paymentSvc.OPCPay().Reconcile(ctx) }()
	}
	wg.Wait()
	check(balance(1) == 20 && balance(2) == 10, "concurrent recovery credits each user exactly once")
	cycle()
	cycle()
	var orders, journal int
	must(db.QueryRow(`SELECT COUNT(*) FROM payment_orders WHERE status='COMPLETED'`).Scan(&orders))
	must(db.QueryRow(`SELECT COUNT(*) FROM wallet_journal WHERE operation='payment'`).Scan(&journal))
	merchant.mu.Lock()
	debits = merchant.debits
	merchant.mu.Unlock()
	check(orders == 3 && journal == 3 && debits == 3, "repeated sweeps produce one order, credit and debit per payment")
	must(db.QueryRow(`SELECT COUNT(*) FROM opcpay_operations WHERE business_id=$1`, originalID).Scan(&orders))
	check(orders == 1, "recovery preserves the original consumption ID")
	duplicate := create("one", "auto_duplicate_001")
	paid(duplicate, "1")
	merchant.mu.Lock()
	merchant.forceTradeNo = "fixture-trade-" + first
	merchant.mu.Unlock()
	due()
	_, duplicateErr := paymentSvc.OPCPay().Reconcile(ctx)
	merchant.mu.Lock()
	debits = merchant.debits
	merchant.forceTradeNo = ""
	merchant.mu.Unlock()
	check(duplicateErr != nil && balance(1) == 20 && debits == 3, "one upstream transaction cannot fund two purchase entries")
	var missingTrade int
	must(db.QueryRow(`SELECT COUNT(*) FROM payment_orders WHERE payment_trade_no=''`).Scan(&missingTrade))
	check(missingTrade == 0, "delivered orders retain the verified provider transaction ID")

	hashes := map[string]string{}
	for _, path := range []string{"internal/service/opcpay_reconciliation.go", "internal/service/opcpay_service.go", "internal/service/payment_order_expiry_service.go", "internal/opcpay/client.go", "internal/handler/opcpay_handler.go", "internal/server/routes/payment.go", "migrations/259_opcpay_reconciliation.sql", "tools/opcpay-e2e/automatic.go", "tools/opcpay-e2e/main.go", "../frontend/src/components/payment/OPCPayPanel.vue", "../frontend/src/api/opcpay.ts"} {
		raw, err := os.ReadFile(path)
		must(err)
		hash := sha256.Sum256(raw)
		hashes[path] = hex.EncodeToString(hash[:])
	}
	out := os.Getenv("OPCPAY_E2E_REPORT")
	if out == "" {
		out = "../output/opcpay-auto/checks.json"
	}
	must(os.MkdirAll(filepath.Dir(out), 0755))
	receipt := map[string]any{"mode": "real-postgresql-http-background-scheduler-simulated-merchant", "passed": len(checks), "failed": 0, "checks": checks, "source_sha256": hashes, "provider_debits": debits, "user_1_balance": balance(1), "user_2_balance": balance(2), "real_payment": false, "production_modified": false}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	must(err)
	must(os.WriteFile(out, raw, 0644))
}
