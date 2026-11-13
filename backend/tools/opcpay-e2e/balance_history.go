package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

// Exercises the real admin handler after real HTTP consumption and replay above.
func verifyBalanceHistory(db *sql.DB, request func(string, string, string, any) (int, map[string]any), check func(bool, string)) map[string]any {
	history := func(user int, query string) map[string]any {
		status, body := request("one", http.MethodGet, fmt.Sprintf("/admin/payment/users/%d/balance-history%s", user, query), nil)
		check(status == 200, "balance history HTTP succeeds: "+query)
		return body["data"].(map[string]any)
	}
	actual := history(1, "?type=opcpay_balance")
	check(actual["total"] == float64(6) && actual["total_recharged"] == float64(15), "completed HTTP consumption appears once after replay and restart")
	check(len(actual["items"].([]any)) == 6, "typed history includes every completed consumption")

	// Historical data mirrors a completed 100-credit recharge plus excluded states.
	_, err := db.Exec(`
INSERT INTO users (id,email,password_hash,role,status,balance,concurrency,created_at,updated_at)
VALUES (3,'history@test.invalid','fixture','user','active',5,1,NOW(),NOW());
INSERT INTO opcpay_operations (business_id,user_id,action_id,kind,amount,credit_usd,quota_unit,status,created_at,updated_at) VALUES
('history-credit-100',3,'history-100','consume',100,100,'POINT','COMPLETED','2026-01-01Z','2026-01-06Z'),
('history-credit-20',3,'history-20','consume',20,20,'POINT','COMPLETED','2026-01-01Z','2026-01-04Z'),
('history-purchase',3,'history-100','purchase',100,100,'POINT','CLOSED','2026-01-01Z','2026-01-06Z'),
('history-ready',3,'history-ready','purchase',100,100,'POINT','READY','2026-01-01Z','2026-01-07Z'),
('history-failed',3,'history-failed','consume',100,100,'POINT','FAILED','2026-01-01Z','2026-01-07Z'),
('history-pending',3,'history-pending','consume',100,100,'POINT','CONFIRMED','2026-01-01Z','2026-01-07Z');
SELECT wallet_credit_payment(3,120,120,NULL);
INSERT INTO redeem_codes (code,type,value,status,used_by,used_at,created_at,notes) VALUES
('history-redeem','balance',7,'used',3,'2026-01-05Z','2026-01-01Z',''),
('history-admin','admin_balance',3,'used',3,'2026-01-03Z','2026-01-01Z','fixture'),
('history-debit','admin_balance',-2,'used',3,'2026-01-02Z','2026-01-01Z','fixture');
INSERT INTO user_affiliate_ledger (user_id,action,amount,created_at) VALUES (3,'transfer',2,'2026-01-04Z');`)
	must(err)
	first := history(3, "?page=1&page_size=2")
	items := first["items"].([]any)
	check(first["total"] == float64(6) && first["pages"] == float64(3) && first["total_recharged"] == float64(130), "mixed history counts completed credits and preserves existing top-up total")
	check(len(items) == 2 && items[0].(map[string]any)["code"] == "history-credit-100" && items[1].(map[string]any)["code"] == "history-redeem", "all-types history sorts by completion time before pagination")
	check(items[0].(map[string]any)["value"] == float64(100) && items[0].(map[string]any)["used_at"] == "2026-01-06T00:00:00Z", "historical 100-credit record reports amount and completion time")
	seen := map[string]bool{}
	for page := 1; page <= 3; page++ {
		body := history(3, fmt.Sprintf("?page=%d&page_size=2", page))
		check(body["total_recharged"] == float64(130), "summary remains constant across pages")
		for _, raw := range body["items"].([]any) {
			item := raw.(map[string]any)
			key := item["type"].(string) + ":" + item["code"].(string)
			check(!seen[key], "mixed-source row identity is unique: "+key)
			seen[key] = true
		}
	}
	check(len(seen) == 6, "mixed pagination neither omits nor duplicates records")
	for kind, count := range map[string]float64{"opcpay_balance": 2, "balance": 1, "admin_balance": 2, "affiliate_balance": 1, "concurrency": 0} {
		body := history(3, "?type="+kind)
		check(body["total"] == count && body["total_recharged"] == float64(130), "type filter and cumulative recharge: "+kind)
	}
	typedPage := history(3, "?type=opcpay_balance&page=2&page_size=1")
	check(typedPage["items"].([]any)[0].(map[string]any)["code"] == "history-credit-20", "OPCPay filter supports pagination")
	empty := history(3, "?page=20&page_size=2")
	check(len(empty["items"].([]any)) == 0 && empty["total"] == float64(6) && empty["total_recharged"] == float64(130), "out-of-range page preserves count and summary")
	other := history(2, "")
	check(other["total"] == float64(0) && other["total_recharged"] == float64(0), "history and summary stay isolated by user")
	var balance, paid float64
	must(db.QueryRow(`SELECT balance,balance-bonus_balance-unverified_balance FROM users WHERE id=3`).Scan(&balance, &paid))
	check(balance == 125 && paid == 120, "history queries leave balances unchanged")
	return map[string]any{"completed_http_credits": actual, "historical_first_page": first, "historical_opcpay_second_page": typedPage, "isolated_user": other}
}
