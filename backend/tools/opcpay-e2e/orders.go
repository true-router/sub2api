package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"
)

// Uses real handlers and wallet transactions. The merchant alone is simulated.
func verifyOrders(db *sql.DB, request func(string, string, string, any) (int, map[string]any), check func(bool, string)) map[string]any {
	get := func(session, path string) map[string]any {
		status, body := request(session, http.MethodGet, path, nil)
		check(status == 200, "order HTTP succeeds: "+path)
		return body["data"].(map[string]any)
	}
	list := get("one", "/admin/payment/orders?payment_type=opcpay&user_id=1")
	check(list["total"] == float64(6), "every delivered OPCPay consumption has exactly one managed order")
	var mismatches int
	must(db.QueryRow(`SELECT COUNT(*) FROM opcpay_operations o WHERE o.user_id=1 AND
 ((o.kind='consume' AND o.status='COMPLETED' AND (SELECT COUNT(*) FROM payment_orders p WHERE p.out_trade_no=o.business_id)<>1)
 OR ((o.kind='purchase' OR o.status<>'COMPLETED') AND EXISTS(SELECT 1 FROM payment_orders p WHERE p.out_trade_no=o.business_id)))`).Scan(&mismatches))
	check(mismatches == 0, "purchase, unknown and failed delivery never create completed orders; replay stays unique")
	must(db.QueryRow(`SELECT COUNT(*) FROM wallet_journal j WHERE j.user_id=1 AND j.operation='payment' AND NOT EXISTS
 (SELECT 1 FROM payment_orders p WHERE p.id=j.order_id AND p.user_id=j.user_id AND p.amount=j.delta AND p.status='COMPLETED')`).Scan(&mismatches))
	check(mismatches == 0, "every new wallet payment links to its completed order")

	var orderID int64
	must(db.QueryRow(`SELECT id FROM payment_orders WHERE out_trade_no=(SELECT business_id FROM opcpay_operations WHERE user_id=1 AND action_id='card_purchase_001' AND kind='consume')`).Scan(&orderID))
	detail := get("one", fmt.Sprintf("/admin/payment/orders/%d", orderID))["order"].(map[string]any)
	check(detail["amount"] == float64(10) && detail["pay_amount"] == 9.9 && detail["currency"] == "CNY" && detail["pay_amount_known"] == true, "managed order preserves catalog price separately from USD credit")
	check(detail["out_trade_no"] != "" && detail["provider_key"] == "opcpay" && detail["completed_at"] != nil, "order retains business identity, provider and delivery timestamp")
	user := get("one", "/payment/orders/my?payment_type=opcpay")
	check(user["total"] == float64(6), "user order history includes completed OPCPay payments")
	other := get("two", "/payment/orders/my?payment_type=opcpay")
	check(other["total"] == float64(0), "order history remains isolated by authenticated user")
	status, _ := request("two", http.MethodGet, fmt.Sprintf("/payment/orders/%d", orderID), nil)
	check(status >= 400, "another user cannot read the order")
	for _, session := range []string{"one", "two"} {
		path := fmt.Sprintf("/payment/orders/%d/refund", orderID)
		if session == "one" {
			path = fmt.Sprintf("/admin/payment/orders/%d/refund", orderID)
		}
		status, _ = request(session, http.MethodPost, path, map[string]any{"amount": 1, "reason": "fixture", "deduct_balance": true, "force": true})
		check(status >= 400, "unsupported OPCPay refund is rejected: "+session)
	}
	status, _ = request("one", http.MethodPost, fmt.Sprintf("/payment/orders/%d/refund", orderID), map[string]any{"reason": "fixture"})
	check(status >= 400, "order owner cannot request an unsupported OPCPay refund")
	before := walletState(db)
	migration, err := os.ReadFile("migrations/255_opcpay_payment_orders.sql")
	must(err)
	for i := 0; i < 2; i++ {
		tx, err := db.Begin()
		must(err)
		_, err = tx.Exec(string(migration))
		must(err)
		must(tx.Commit())
	}
	check(walletState(db) == before, "historical backfill and replay leave balances, wallet journal and operation timestamps unchanged")
	historical := get("one", "/admin/payment/orders?payment_type=opcpay&user_id=3")
	check(historical["total"] == float64(2), "backfill includes only completed historical consumption, exactly once")
	for _, raw := range historical["items"].([]any) {
		item := raw.(map[string]any)
		check(item["pay_amount_known"] == false && item["pay_amount"] == float64(0), "historical payment amount is explicitly unknown")
		check(item["completed_at"] == "2026-01-06T00:00:00Z" || item["completed_at"] == "2026-01-04T00:00:00Z", "backfill retains original delivery timestamp")
	}
	filtered := get("one", "/admin/payment/orders?keyword=history-credit-100&status=COMPLETED&order_type=balance&payment_type=opcpay")
	check(filtered["total"] == float64(1), "managed orders support business ID, status, type and method filters")
	page := get("one", "/admin/payment/orders?payment_type=opcpay&user_id=1&page=2&page_size=1")
	check(page["total"] == float64(6) && len(page["items"].([]any)) == 1, "managed orders support pagination")
	dashboard := get("one", "/admin/payment/dashboard?days=1")
	check(dashboard["total_count"] == float64(6), "dashboard counts all completed current OPCPay orders")
	check(dashboard["total_amount"].(map[string]any)["CNY"] == 9.9 && dashboard["avg_amount"].(map[string]any)["CNY"] == 9.9, "dashboard totals and averages exclude unknown payment amounts")
	// A database failure must never deliver funds without the matching order.
	_, err = db.Exec(`CREATE FUNCTION reject_opcpay_order() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated order failure'; END; $$;
CREATE TRIGGER reject_opcpay_order BEFORE INSERT ON payment_orders FOR EACH ROW EXECUTE FUNCTION reject_opcpay_order()`)
	must(err)
	var balanceBefore float64
	must(db.QueryRow(`SELECT balance FROM users WHERE id=1`).Scan(&balanceBefore))
	status, _ = request("one", http.MethodPost, "/operations", map[string]any{"kind": "purchase", "action_id": "order_failure_00001", "sku_id": "large-fixture-sku"})
	check(status == 200, "purchase entry succeeds without a paid order")
	status, _ = request("one", http.MethodPost, "/operations", map[string]any{"kind": "consume", "action_id": "order_failure_00001", "sku_id": "large-fixture-sku"})
	check(status == 409, "failed order insertion blocks wallet delivery")
	var balanceAfter float64
	var operationStatus, businessID string
	must(db.QueryRow(`SELECT balance FROM users WHERE id=1`).Scan(&balanceAfter))
	must(db.QueryRow(`SELECT status,business_id FROM opcpay_operations WHERE user_id=1 AND kind='consume' AND action_id='order_failure_00001'`).Scan(&operationStatus, &businessID))
	check(balanceBefore == balanceAfter && operationStatus == "CONFIRMED", "failed order insertion preserves confirmed consumption for recovery")
	_, err = db.Exec(`DROP TRIGGER reject_opcpay_order ON payment_orders`)
	must(err)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request("one", http.MethodPost, "/operations/"+businessID+"/resume", nil)
		}()
	}
	wg.Wait()
	must(db.QueryRow(`SELECT balance FROM users WHERE id=1`).Scan(&balanceAfter))
	must(db.QueryRow(`SELECT COUNT(*) FROM payment_orders WHERE out_trade_no=$1`, businessID).Scan(&mismatches))
	check(balanceAfter == balanceBefore+20 && mismatches == 1, "concurrent recovery delivers funds and its order exactly once after insertion failure")
	return map[string]any{"current_orders": list, "historical_orders": historical, "card_order": detail, "dashboard": dashboard}
}

func walletState(db *sql.DB) string {
	var state string
	must(db.QueryRow(`SELECT md5((SELECT jsonb_agg(to_jsonb(u) ORDER BY id)::text FROM users u) ||
 (SELECT COALESCE(jsonb_agg(to_jsonb(j) ORDER BY id),'[]')::text FROM wallet_journal j) ||
 (SELECT jsonb_agg(to_jsonb(o) ORDER BY business_id)::text FROM opcpay_operations o) ||
 (SELECT COALESCE(jsonb_agg(to_jsonb(g) ORDER BY id),'[]')::text FROM wallet_grants g) ||
 (SELECT COALESCE(jsonb_agg(to_jsonb(h) ORDER BY user_id,hold_key),'[]')::text FROM wallet_holds h))`).Scan(&state))
	return state
}
