package main

import (
	"encoding/json"
	"fmt"
)

type giftNotice struct {
	ID     int64   `json:"id"`
	Type   string  `json:"type"`
	Amount float64 `json:"gift_amount"`
	Source string  `json:"gift_source"`
	ReadAt *string `json:"read_at"`
}

func (f *fixture) giftInbox(jwt string) ([]giftNotice, error) {
	r, err := f.request("GET", "/api/v1/notifications", jwt, nil, "")
	if err != nil {
		return nil, err
	}
	if r.Status != 200 {
		return nil, fmt.Errorf("typed inbox status=%d", r.Status)
	}
	var envelope struct {
		Data []giftNotice `json:"data"`
	}
	err = json.Unmarshal(r.Raw, &envelope)
	return envelope.Data, err
}
func (f *fixture) giftScenario(admin string) error {
	user, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "discount-unified@example.test", "password": "Discount-user-password-2026", "role": "user", "balance": 0, "concurrency": 5})
	if err != nil {
		return err
	}
	uid := int64(user["id"].(float64))
	jwt, _, err := f.login("discount-unified@example.test", "Discount-user-password-2026")
	if err != nil {
		return err
	}
	notices, err := f.giftInbox(jwt)
	if err != nil {
		return err
	}
	if err = f.checked("Typed notification inbox starts empty", len(notices) == 0); err != nil {
		return err
	}
	body := map[string]any{"operation": "add", "balance": 5, "notes": "Synthetic gift acceptance"}
	idem := token()
	for i := 0; i < 2; i++ {
		r, e := f.request("POST", fmt.Sprintf("/api/v1/admin/users/%d/balance", uid), admin, body, idem)
		if e != nil {
			return e
		}
		if r.Status != 200 {
			return fmt.Errorf("gift credit status=%d", r.Status)
		}
	}
	if err = f.wallet("Idempotent administrator gift credits promotional funds once", jwt, uid, "5", "0", "5"); err != nil {
		return err
	}
	notices, err = f.giftInbox(jwt)
	if err != nil {
		return err
	}
	if err = f.checked("Gift notice is typed, unread, and records the real credit once", len(notices) == 1 && notices[0].Type == "gift" && notices[0].Amount == 5 && notices[0].Source == "credit" && notices[0].ReadAt == nil); err != nil {
		return err
	}
	giftID := notices[0].ID
	other, err := f.api("POST", "/api/v1/admin/users", admin, map[string]any{"email": "other-gift@example.test", "password": "Other-gift-password-2026", "role": "user", "balance": 0})
	if err != nil {
		return err
	}
	_ = other
	otherJWT, _, err := f.login("other-gift@example.test", "Other-gift-password-2026")
	if err != nil {
		return err
	}
	others, err := f.giftInbox(otherJWT)
	if err != nil {
		return err
	}
	if err = f.checked("Another user's inbox excludes the gift", len(others) == 0); err != nil {
		return err
	}
	r, err := f.request("POST", fmt.Sprintf("/api/v1/notifications/gift/%d/read", giftID), otherJWT, map[string]any{}, "")
	if err != nil {
		return err
	}
	if err = f.checked("Another user cannot claim the gift", r.Status == 404); err != nil {
		return err
	}
	r, err = f.request("GET", "/api/v1/notifications", "", nil, "")
	if err != nil {
		return err
	}
	if err = f.checked("Anonymous notification access is denied", r.Status == 401); err != nil {
		return err
	}
	// A gift event and a public announcement can have the same numeric ID.
	_, err = f.api("POST", "/api/v1/admin/announcements", admin, map[string]any{"title": "Gift fixture announcement", "content": "Synthetic system announcement. This is not a gift.", "status": "active", "notify_mode": "popup"})
	if err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		r, err = f.request("POST", fmt.Sprintf("/api/v1/notifications/gift/%d/read", giftID), jwt, map[string]any{}, "")
		if err != nil {
			return err
		}
		if r.Status != 200 {
			return fmt.Errorf("claim status=%d", r.Status)
		}
	}
	if err = f.wallet("Claim replay never credits funds again", jwt, uid, "5", "0", "5"); err != nil {
		return err
	}
	notices, err = f.giftInbox(jwt)
	if err != nil {
		return err
	}
	giftRead, announcementUnread := false, false
	for _, n := range notices {
		if n.Type == "gift" && n.ID == giftID {
			giftRead = n.ReadAt != nil
		}
		if n.Type == "announcement" {
			announcementUnread = n.ReadAt == nil
		}
	}
	if err = f.checked("Claim reads only the typed gift and preserves the announcement", giftRead && announcementUnread); err != nil {
		return err
	}
	for _, adjust := range []map[string]any{{"operation": "subtract", "balance": 1, "notes": "Synthetic deduction"}, {"operation": "add_paid", "balance": 2, "notes": "Verified synthetic paid credit"}, {"operation": "convert_paid_to_bonus", "balance": 1, "notes": "Synthetic reclassification"}} {
		if _, err = f.api("POST", fmt.Sprintf("/api/v1/admin/users/%d/balance", uid), admin, adjust); err != nil {
			return err
		}
	}
	var count int
	if err = f.db.QueryRow(`SELECT count(*) FROM gift_notifications WHERE user_id=$1`, uid).Scan(&count); err != nil {
		return err
	}
	if err = f.checked("Debits, purchased credit, and conversion create no gift notices", count == 1); err != nil {
		return err
	}
	_, err = f.api("POST", "/api/v1/admin/payment/providers", admin, map[string]any{"provider_key": "easypay", "name": "Gift payment fixture", "enabled": true, "supported_types": []string{"alipay"}, "config": map[string]string{"pid": "fixture-merchant", "pkey": "fixture-merchant-key", "apiBase": f.mock.URL, "notifyUrl": f.origin + "/api/v1/payment/webhook/easypay", "returnUrl": f.origin + "/purchase"}})
	if err != nil {
		return err
	}
	_, err = f.api("PUT", "/api/v1/admin/payment/config", admin, map[string]any{"enabled": true, "enabled_payment_types": []string{"alipay"}, "min_amount": 1, "max_amount": 100, "balance_recharge_multiplier": 1, "recharge_bonus_mode": "bonus", "recharge_bonus_tiers": []any{map[string]any{"min_amount": 1, "bonus_percent": 20}}})
	if err != nil {
		return err
	}
	order, err := f.api("POST", "/api/v1/payment/orders", jwt, map[string]any{"amount": 10, "payment_type": "alipay", "order_type": "balance"})
	if err != nil {
		return err
	}
	f.markPaid(order)
	for i := 0; i < 2; i++ {
		r, err = f.callback(order, "10.00", true)
		if err != nil {
			return err
		}
		if r.Status != 200 {
			return fmt.Errorf("payment callback status=%d", r.Status)
		}
	}
	if err = f.wallet("Payment credits principal and gift without callback duplication", jwt, uid, "18", "11", "7"); err != nil {
		return err
	}
	notices, err = f.giftInbox(jwt)
	if err != nil {
		return err
	}
	paymentCount := 0
	for _, n := range notices {
		if n.Type == "gift" && n.Source == "payment_bonus" && n.Amount == 2 {
			paymentCount++
		}
	}
	if err = f.checked("Payment gift notification contains only the bonus portion once", paymentCount == 1); err != nil {
		return err
	}
	_, err = f.api("PUT", "/api/v1/admin/settings", admin, map[string]any{"registration_enabled": true, "email_verify_enabled": false, "default_balance": 3})
	if err != nil {
		return err
	}
	signup, err := f.api("POST", "/api/v1/auth/register", "", map[string]any{"email": "signup-gift@example.test", "password": "Signup-gift-password-2026"})
	if err != nil {
		return err
	}
	signupJWT, _ := signup["access_token"].(string)
	initial, err := f.giftInbox(signupJWT)
	if err != nil {
		return err
	}
	initialCount := 0
	for _, n := range initial {
		if n.Type == "gift" && n.Source == "initial" && n.Amount == 3 && n.ReadAt == nil {
			initialCount++
		}
	}
	if err = f.checked("Registration gift creates one unread personal gift notification", initialCount == 1); err != nil {
		return err
	}
	// A failed wallet transaction must not leave a notice behind.
	_, err = f.db.Exec(`DO $$ BEGIN UPDATE users SET balance=balance+9 WHERE id=` + fmt.Sprint(uid) + `; RAISE EXCEPTION 'synthetic rollback'; END $$`)
	if err = f.checked("Synthetic rolled-back credit transaction fails", err != nil); err != nil {
		return err
	}
	if err = f.db.QueryRow(`SELECT count(*) FROM gift_notifications WHERE user_id=$1`, uid).Scan(&count); err != nil {
		return err
	}
	if err = f.checked("Rollback also removes its gift notice", count == 2); err != nil {
		return err
	}
	if err = f.stop(); err != nil {
		return err
	}
	if err = f.start(); err != nil {
		return err
	}
	notices, err = f.giftInbox(jwt)
	if err != nil {
		return err
	}
	giftRead = false
	for _, n := range notices {
		if n.Type == "gift" && n.ID == giftID && n.ReadAt != nil {
			giftRead = true
		}
	}
	if err = f.checked("Gift acknowledgement survives a real process restart", giftRead); err != nil {
		return err
	}
	_, err = f.api("POST", "/api/v1/admin/announcements", admin, map[string]any{"title": "Second gift fixture announcement", "content": "Another synthetic system announcement.", "status": "active", "notify_mode": "silent"})
	if err != nil {
		return err
	}
	_, err = f.api("POST", "/api/v1/admin/announcements", admin, map[string]any{"title": "Third gift fixture announcement", "content": "Synthetic bulk-read fixture.", "status": "active", "notify_mode": "silent"})
	if err != nil {
		return err
	}
	// Leave one unread administrator gift for the real browser animation scenario.
	_, err = f.api("POST", fmt.Sprintf("/api/v1/admin/users/%d/balance", uid), admin, map[string]any{"operation": "add", "balance": 7, "notes": "Synthetic browser gift"})
	if err != nil {
		return err
	}
	return f.migrations()
}
