// Run only against a disposable PostgreSQL database; never use a production DSN.
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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/opcpay"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type redirectTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "ai6666.com" {
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
func main() {
	gin.SetMode(gin.ReleaseMode)
	dsn := os.Getenv("OPCPAY_E2E_DSN")
	if !strings.Contains(dsn, "opcpay_e2e") {
		panic("use a disposable opcpay_e2e database")
	}
	db, err := sql.Open("postgres", dsn)
	must(err)
	defer db.Close()
	ctx := context.Background()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	must(client.Schema.Create(ctx))
	migration, err := os.ReadFile("migrations/251_opcpay_operations.sql")
	must(err)
	_, err = db.Exec(string(migration))
	must(err)
	binding, err := os.ReadFile("migrations/252_opcpay_provider_binding.sql")
	must(err)
	_, err = db.Exec(string(binding))
	must(err)
	auditSchema, err := os.ReadFile("migrations/180_audit_logs.sql")
	must(err)
	_, err = db.Exec(string(auditSchema))
	must(err)
	// Payment credits now share the wallet accounting migration.
	for _, name := range []string{"071_add_usage_billing_dedup.sql", "073_add_usage_billing_dedup_archive.sql", "131_affiliate_rebate_hardening.sql", "253_wallet_balance_sources.sql", "255_opcpay_payment_orders.sql"} {
		data, err := os.ReadFile("migrations/" + name)
		must(err)
		_, err = db.Exec(string(data))
		must(err)
	}
	settingRepo := repository.NewSettingRepository(client)
	configSvc := service.NewPaymentConfigService(client, settingRepo, []byte(strings.Repeat("e", 32)))
	auditSvc := service.NewAuditLogService(repository.NewAuditLogRepository(db), nil)
	auditSvc.Start()
	defer auditSvc.Stop()

	_, err = db.Exec(`INSERT INTO users (id,email,password_hash,role,status,balance,concurrency,created_at,updated_at) VALUES (1,'one@test.invalid','fixture','user','active',0,1,NOW(),NOW()),(2,'two@test.invalid','fixture','user','active',0,1,NOW(),NOW())`)
	must(err)
	os.Setenv("opcpay_api_key", "must-not-use-this-environment-key")
	os.Setenv("OPCPAY_PUBLIC_ORIGIN", "https://app.example.test")
	var mu sync.Mutex
	results := map[string]map[string]any{}
	calls := 0
	deductions := 0
	providerConfigured := true
	quantity := true
	cardPrice := "9.90"
	allow := true
	includeDecision := true
	opened := true
	lose := false
	mismatch := false
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if r.Header.Get("Authorization") != "Bearer opcpay-e2e-synthetic-key" {
			panic("missing server credential")
		}
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		if r.Method == "POST" {
			must(json.NewDecoder(r.Body).Decode(&body))
			if body["user_id"] != "1" && body["user_id"] != "2" {
				panic("untrusted user")
			}
		}
		send := func(v any) { must(json.NewEncoder(w).Encode(v)) }
		switch strings.TrimPrefix(r.URL.Path, "/opcpay/api/pay/v1") {
		case "/capabilities":
			send(map[string]any{"provider_configured": providerConfigured, "capabilities": map[string]bool{"purchase_entry": opened, "entitlement_query": true, "consumption": opened}})
		case "/product":
			mode := "QUANTITY"
			unit := "POINT"
			if !quantity {
				mode = "ONE_TIME"
				unit = ""
			}
			send(map[string]any{"capability_status": "ENABLED", "billing_mode": mode, "quota_unit": unit, "charging_options": []map[string]any{{"sku_id": "actual-fixture-sku", "display_name": "10 积分充值卡", "price": cardPrice, "currency": "CNY", "billing_mode": "QUANTITY", "quota_unit": "POINT", "quota_amount": 10}, {"sku_id": "large-fixture-sku", "display_name": "20 积分充值卡", "price": "18.80", "currency": "CNY", "billing_mode": "QUANTITY", "quota_unit": "POINT", "quota_amount": 20}, {"sku_id": "count-fixture", "display_name": "次数套餐", "billing_mode": "QUANTITY", "quota_unit": "COUNT", "quota_amount": 10, "price": "1.00", "currency": "CNY"}}})
		case "/access":
			decision := "ALLOW"
			if !allow {
				decision = "PURCHASE_REQUIRED"
			}
			access := map[string]any{"access_granted": allow, "requires_consumption": quantity, "quota_unit": "POINT", "remaining": 50}
			if includeDecision {
				access["decision"] = decision
			}
			send(access)
		case "/purchase":
			if (body["sku_id"] != "actual-fixture-sku" && body["sku_id"] != "large-fixture-sku") || body["callback_url"] != "https://app.example.test/purchase" {
				panic("purchase parameters changed")
			}
			send(map[string]any{"status": "READY", "idempotency_key": body["idempotency_key"], "purchase_url": "https://example.test/pay"})
		case "/consumption":
			if v := results[body["idempotency_key"].(string)]; v != nil {
				send(v)
			} else {
				w.WriteHeader(404)
				send(map[string]any{"error": "not found"})
			}
		case "/consume":
			key := body["idempotency_key"].(string)
			if results[key] == nil {
				if (body["amount"] != float64(1) && body["amount"] != float64(10) && body["amount"] != float64(20)) || body["reason"] != "sub2api balance credit" {
					panic("unexpected debit")
				}
				deductions++
				v := map[string]any{"status": "SUCCEEDED", "confirmed": true, "idempotency_key": key, "consumed": body["amount"], "quota_unit": "POINT"}
				if mismatch {
					v["quota_unit"] = "COUNT"
				}
				results[key] = v
			}
			if lose {
				lose = false
				w.WriteHeader(503)
				send(map[string]any{"error": "response lost"})
				return
			}
			send(results[key])
		default:
			panic("unsupported capability called")
		}
	}))
	defer mock.Close()
	original := http.DefaultTransport
	target, _ := url.Parse(mock.URL)
	http.DefaultTransport = redirectTransport{target, original}
	defer func() { http.DefaultTransport = original }()
	httpClient := &http.Client{Transport: original, Timeout: 10 * time.Second}
	newAPI := func() *httptest.Server {
		paymentSvc := service.NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, configSvc, nil, nil, nil)
		h := handler.NewPaymentHandler(paymentSvc, configSvc)
		adminH := adminhandler.NewPaymentHandler(paymentSvc, configSvc)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			if c.GetHeader("X-Fixture-Session") == "one" {
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
			} else if c.GetHeader("X-Fixture-Session") == "two" {
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 2})
			}
		})
		admin := router.Group("/admin/payment")
		admin.Use(func(c *gin.Context) {
			if c.GetHeader("X-Fixture-Session") != "one" {
				c.AbortWithStatusJSON(403, gin.H{"message": "forbidden"})
				return
			}
		})
		admin.Use(gin.HandlerFunc(middleware.NewAuditLogMiddleware(auditSvc)))
		admin.GET("/orders", adminH.ListOrders)
		admin.GET("/orders/:id", adminH.GetOrderDetail)
		admin.GET("/dashboard", adminH.GetDashboard)
		admin.POST("/orders/:id/refund", adminH.ProcessRefund)
		router.GET("/payment/orders/my", h.GetMyOrders)
		router.GET("/payment/orders/:id", h.GetOrder)
		router.POST("/payment/orders/:id/refund", h.RequestRefund)
		admin.GET("/providers", adminH.ListProviders)
		admin.POST("/providers", adminH.CreateProvider)
		admin.PUT("/providers/:id", adminH.UpdateProvider)
		admin.DELETE("/providers/:id", adminH.DeleteProvider)
		admin.PUT("/config", adminH.UpdateConfig)
		adminSvc := service.NewAdminService(nil, nil, nil, nil, nil, nil, repository.NewRedeemCodeRepository(client), nil, nil, nil, nil, nil, nil, client, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		userH := adminhandler.NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
		admin.GET("/users/:id/balance-history", userH.GetBalanceHistory)
		router.GET("/payment/opcpay", h.OPCPayInfo)
		router.POST("/payment/opcpay/operations", h.OPCPayStart)
		router.POST("/payment/opcpay/operations/:business_id/resume", h.OPCPayResume)
		router.POST("/payment/opcpay/operations/:business_id/close", h.OPCPayClosePurchase)
		return httptest.NewServer(router)
	}
	api := newAPI()
	defer func() { api.Close() }()
	request := func(session, method, path string, body any) (int, map[string]any) {
		data, _ := json.Marshal(body)
		fullPath := "/payment/opcpay" + path
		if strings.HasPrefix(path, "/admin/") || strings.HasPrefix(path, "/payment/") {
			fullPath = path
		}
		req, _ := http.NewRequest(method, api.URL+fullPath, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Fixture-Session", session)
		resp, e := httpClient.Do(req)
		must(e)
		defer resp.Body.Close()
		raw, e := io.ReadAll(resp.Body)
		must(e)
		if bytes.Contains(raw, []byte("synthetic-key")) {
			panic("credential leaked")
		}
		var result map[string]any
		must(json.Unmarshal(raw, &result))
		return resp.StatusCode, result
	}
	checks := []string{}
	check := func(ok bool, name string) {
		if !ok {
			panic(name)
		}
		checks = append(checks, name)
		fmt.Println("PASS", name)
	}
	balance := func() float64 {
		var v float64
		must(db.QueryRow(`SELECT balance FROM users WHERE id=1`).Scan(&v))
		return v
	}
	start := func(kind, action string) (int, map[string]any) {
		sku := ""
		if kind == "purchase" {
			sku = "actual-fixture-sku"
		}
		return request("one", "POST", "/operations", map[string]any{"kind": kind, "action_id": action, "sku_id": sku})
	}
	data := func(m map[string]any) map[string]any { return m["data"].(map[string]any) }
	status, unconfigured := request("one", "GET", "", nil)
	check(status == 200 && data(unconfigured)["configured"] == false && calls == 0, "忽略环境变量中的 API Key")
	status, _ = request("two", "POST", "/admin/payment/providers", map[string]any{"provider_key": "opcpay"})
	check(status == 403, "普通用户不能配置支付服务商")
	status, _ = request("one", "PUT", "/admin/payment/config", map[string]any{"enabled": true, "enabled_payment_types": []string{"opcpay"}})
	check(status == 200, "通过管理端启用一人付支付类型")
	providerBody := map[string]any{"provider_key": "opcpay", "name": "一人付充值卡", "supported_types": []string{"opcpay"}, "enabled": true, "config": map[string]string{}}
	status, _ = request("one", "POST", "/admin/payment/providers", providerBody)
	check(status == 400, "启用服务商前必须填写 API Key")
	providerBody["config"] = map[string]string{"apiKey": "opcpay-e2e-synthetic-key", "publicOrigin": "https://app.example.test"}
	status, created := request("one", "POST", "/admin/payment/providers", providerBody)
	check(status == 201, "在现有服务商接口添加一人付")
	providerID := int64(data(created)["id"].(float64))
	providerPath := fmt.Sprintf("/admin/payment/providers/%d", providerID)
	var stored string
	must(db.QueryRow(`SELECT config FROM payment_provider_instances WHERE id=$1`, providerID).Scan(&stored))
	check(strings.HasPrefix(stored, "opcpay:v1:") && !strings.Contains(stored, "synthetic-key"), "API Key 加密保存且写响应不回显")
	status, listed := request("one", "GET", "/admin/payment/providers", nil)
	listedConfig := listed["data"].([]any)[0].(map[string]any)["config"].(map[string]any)
	check(status == 200 && listedConfig["apiKey"] == nil && listedConfig["apiKeyConfigured"] == "true", "服务商列表仅返回密钥配置状态")
	status, _ = request("one", "PUT", providerPath, map[string]any{"config": map[string]string{"apiKey": ""}})
	check(status == 200, "编辑时留空保留原密钥")
	status, _ = request("one", "POST", "/admin/payment/providers", providerBody)
	check(status == 409, "阻止同时启用多个一人付项目")
	status, _ = request("one", "PUT", providerPath, map[string]any{"refund_enabled": true})
	check(status == 400, "不开放一人付退款开关")
	limits, err := configSvc.GetAvailableMethodLimits(ctx)
	must(err)
	check(len(limits.Methods) == 0, "充值卡不进入自定义金额支付方式")
	status, _ = request("", "GET", "", nil)
	check(status == 401, "拒绝匿名请求")
	status, info := request("one", "GET", "", nil)
	check(status == 200 && data(info)["credit_usd_per_unit"] == float64(1), "读取模拟能力、套餐与积分兑换规则")
	cards := data(info)["product"].(map[string]any)["charging_options"].([]any)
	check(len(cards) == 2 && cards[0].(map[string]any)["quota_amount"] == float64(10), "读取真实接口格式的充值卡并排除次数套餐")
	mu.Lock()
	providerConfigured = false
	mu.Unlock()
	beforeUnavailable := calls
	status, unavailable := request("one", "GET", "", nil)
	check(status == 200 && data(unavailable)["access"] == nil && calls-beforeUnavailable == 2, "商户能力未开放时只读能力和套餐")
	mu.Lock()
	providerConfigured = true
	mu.Unlock()

	status, _ = request("one", "POST", "/operations", map[string]any{"kind": "consume", "action_id": "forgery_000000001", "user_id": "2", "amount": 100})
	check(status == 400, "拒绝伪造用户与扣减量")
	status, _ = request("one", "POST", "/operations", map[string]any{"kind": "purchase", "action_id": "badsku_0000000001", "sku_id": "invented"})
	check(status == 409, "拒绝非商户套餐")
	status, purchase := start("purchase", "purchase_00000001")
	check(status == 200 && data(purchase)["status"] == "READY" && balance() == 0, "购买入口与回跳不增加余额")
	status, _ = request("one", "PUT", providerPath, map[string]any{"config": map[string]string{"apiKey": "other-synthetic-key"}})
	check(status == 409, "未完成购买入口阻止更换项目密钥")
	status, _ = request("one", "DELETE", providerPath, nil)
	check(status == 409, "未完成操作阻止删除原服务商")
	mu.Lock()
	lose = true
	mu.Unlock()
	status, lost := start("consume", "consume_lost_0001")
	id := data(lost)["business_id"].(string)
	check(status == 200 && data(lost)["status"] == "UNKNOWN" && balance() == 0, "核销成功但响应丢失时不交付")
	status, _ = start("consume", "consume_new_00001")
	check(status == 409, "未完成核销阻止新编号")
	status, _ = request("two", "POST", "/operations/"+id+"/resume", nil)
	check(status == 409, "阻止跨用户恢复")
	api.Close()
	api = newAPI()
	status, recovered := request("one", "POST", "/operations/"+id+"/resume", nil)
	check(status == 200 && data(recovered)["status"] == "COMPLETED" && balance() == 1 && deductions == 1, "重建服务后按原编号恢复且仅交付一次")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); request("one", "POST", "/operations/"+id+"/resume", nil) }()
	}
	wg.Wait()
	check(balance() == 1 && deductions == 1, "并发恢复不重复扣减或交付")
	mu.Lock()
	allow = false
	mu.Unlock()
	status, _ = start("consume", "noquota_00000001")
	check(status == 409 && deductions == 1, "额度不足时拒绝核销")
	unpaidAction := "unpaid_purchase_0001"
	status, unpaidPurchase := start("purchase", unpaidAction)
	unpaidID := data(unpaidPurchase)["business_id"].(string)
	check(status == 200, "未付款场景创建原购买入口")
	status, unverified := request("one", "POST", "/operations", map[string]any{"kind": "consume", "action_id": unpaidAction, "sku_id": "actual-fixture-sku"})
	check(status == 409 && unverified["reason"] == "OPCPAY_PAYMENT_UNVERIFIED" && unverified["message"] == "验证失败，请重试", "未付款确认返回明确的可重试验证失败")
	var unpaidConsumes int
	must(db.QueryRow(`SELECT COUNT(*) FROM opcpay_operations WHERE user_id=1 AND kind='consume' AND action_id=$1`, unpaidAction).Scan(&unpaidConsumes))
	check(unpaidConsumes == 0 && balance() == 1 && deductions == 1, "验证失败不创建核销、不扣款且不增加余额")
	status, _ = request("one", "POST", "/operations/"+unpaidID+"/close", nil)
	check(status == 200, "验证失败后可以关闭原购买入口")
	status, retryPurchase := start("purchase", "unpaid_retry_00001")
	check(status == 200 && data(retryPurchase)["business_id"] != unpaidID, "关闭后可以创建新的支付入口")
	status, _ = request("one", "POST", "/operations/"+data(retryPurchase)["business_id"].(string)+"/close", nil)
	check(status == 200, "重新发起的支付入口也可以取消")
	mu.Lock()
	allow = true
	opened = false
	mu.Unlock()
	status, _ = start("purchase", "disabled_0000001")
	check(status == 409, "未开放能力不可使用")
	mu.Lock()
	opened = true
	quantity = false
	mu.Unlock()
	status, _ = start("consume", "duration_0000001")
	check(status == 409, "非积分权益不兑换余额")
	mu.Lock()
	quantity = true
	mu.Unlock()
	_, err = db.Exec(`CREATE FUNCTION reject_opcpay_credit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated delivery failure'; END; $$; CREATE TRIGGER reject_opcpay_credit BEFORE UPDATE OF balance ON users FOR EACH ROW EXECUTE FUNCTION reject_opcpay_credit()`)
	must(err)
	status, _ = start("consume", "delivery_0000001")
	check(status == 409 && balance() == 1 && deductions == 2, "交付失败时回滚回执与余额")
	_, err = db.Exec(`DROP TRIGGER reject_opcpay_credit ON users`)
	must(err)
	beforeDeliveryResume := calls
	status, recovered = start("consume", "delivery_0000001")
	check(status == 200 && data(recovered)["status"] == "COMPLETED" && balance() == 2 && deductions == 2 && calls == beforeDeliveryResume, "恢复已核销的交付且不再次扣减")
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); start("consume", "concurrent_00001") }()
	}
	wg.Wait()
	check(balance() == 3 && deductions == 3, "并发创建同一操作仅扣减与交付一次")
	mu.Lock()
	mismatch = true
	mu.Unlock()
	status, bad := start("consume", "mismatch_0000001")
	check(status == 200 && data(bad)["status"] == "UNKNOWN" && balance() == 3, "拒绝单位不一致的核销确认")
	badID := data(bad)["business_id"].(string)
	mu.Lock()
	mismatch = false
	results[badID]["quota_unit"] = "POINT"
	results[badID]["consumed"] = 2
	mu.Unlock()
	status, bad = request("one", "POST", "/operations/"+badID+"/resume", nil)
	check(status == 200 && data(bad)["status"] == "UNKNOWN" && balance() == 3, "拒绝数量不一致的核销确认")
	mu.Lock()
	results[badID]["consumed"] = 1
	results[badID]["idempotency_key"] = "other-request"
	mu.Unlock()
	status, bad = request("one", "POST", "/operations/"+badID+"/resume", nil)
	check(status == 200 && data(bad)["status"] == "UNKNOWN" && balance() == 3, "拒绝编号不一致的核销确认")
	mu.Lock()
	results[badID]["idempotency_key"] = badID
	results[badID]["confirmed"] = false
	mu.Unlock()
	status, bad = request("one", "POST", "/operations/"+badID+"/resume", nil)
	check(status == 200 && data(bad)["status"] == "UNKNOWN" && balance() == 3, "拒绝未确认的核销结果")
	mu.Lock()
	results[badID]["status"] = "PROCESSING"
	mu.Unlock()
	status, bad = request("one", "POST", "/operations/"+badID+"/resume", nil)
	check(status == 200 && data(bad)["status"] == "UNKNOWN" && balance() == 3 && deductions == 4, "处理中不交付且不创建新核销")
	mu.Lock()
	results[badID]["status"] = "SUCCEEDED"
	results[badID]["confirmed"] = true
	mu.Unlock()
	status, bad = request("one", "POST", "/operations/"+badID+"/resume", nil)
	check(status == 200 && data(bad)["status"] == "COMPLETED" && balance() == 4 && deductions == 4, "恢复原编号的有效确认")
	status, _ = request("one", "POST", "/operations/"+data(purchase)["business_id"].(string)+"/close", nil)
	check(status == 200, "允许结束未核销的购买入口")
	status, cardPurchase := start("purchase", "card_purchase_001")
	check(status == 200 && data(cardPurchase)["amount"] == float64(10) && data(cardPurchase)["credit_usd"] == float64(10), "所选充值卡数量由服务端套餐确定")
	mu.Lock()
	cardPrice = "99.90"
	mu.Unlock()
	status, cardCredit := request("one", "POST", "/operations", map[string]any{"kind": "consume", "action_id": "card_purchase_001", "sku_id": "actual-fixture-sku"})
	check(status == 200 && data(cardCredit)["status"] == "COMPLETED" && balance() == 14 && deductions == 5, "按所选充值卡核销 10 积分并充值 10 USD")
	status, _ = request("one", "POST", "/operations", map[string]any{"kind": "consume", "action_id": "card_purchase_001", "sku_id": "actual-fixture-sku"})
	check(status == 200 && balance() == 14 && deductions == 5, "重复确认充值卡不重复扣减或到账")
	status, _ = request("one", "PUT", providerPath, map[string]any{"enabled": false})
	beforeDisabled := calls
	status, disabledInfo := request("one", "GET", "", nil)
	check(status == 200 && data(disabledInfo)["enabled"] == false && calls == beforeDisabled, "后台停用后不查询上游或展示充值卡")
	status, _ = start("purchase", "disabled_new_001")
	check(status == 409 && calls == beforeDisabled, "后台停用后禁止创建购买入口")
	status, _ = request("one", "PUT", providerPath, map[string]any{"enabled": true})
	check(status == 200, "后台重新启用立即生效")
	mu.Lock()
	includeDecision = false
	mu.Unlock()
	beforeOptionalDecision := balance()
	beforeOptionalDeductions := deductions
	status, optionalDecision := start("consume", "optional_decision_01")
	check(status == 200 && data(optionalDecision)["status"] == "COMPLETED" && balance() == beforeOptionalDecision+1 && deductions == beforeOptionalDeductions+1, "额度响应省略买断决策字段时仍可正常充值")
	beforeHistoryCalls, beforeHistoryDeductions := calls, deductions
	historyReceipt := verifyBalanceHistory(db, request, check)
	check(calls == beforeHistoryCalls && deductions == beforeHistoryDeductions, "历史记录查询不调用上游或再次核销")
	ordersReceipt := verifyOrders(db, request, check)
	auditSvc.Stop()
	var auditRows, leaks int
	must(db.QueryRow(`SELECT COUNT(*),COUNT(*) FILTER(WHERE request_body LIKE '%synthetic-key%') FROM audit_logs`).Scan(&auditRows, &leaks))
	check(auditRows > 0 && leaks == 0, "真实审计中间件不记录 API Key 明文")
	sources := map[string]string{}
	for _, path := range []string{"migrations/131_affiliate_rebate_hardening.sql", "internal/service/domain_constants.go", "internal/handler/admin/user_handler.go", "internal/service/admin_user.go", "internal/service/admin_user_opcpay.go", "tools/opcpay-e2e/balance_history.go", "../frontend/src/api/admin/users.ts", "../frontend/src/components/admin/user/UserBalanceHistoryModal.vue", "../frontend/src/i18n/locales/zh/admin/overview.ts", "../frontend/src/i18n/locales/en/admin/overview.ts", "../frontend/src/i18n/locales/zh/dashboard.ts", "../frontend/src/i18n/locales/en/dashboard.ts"} {
		raw, err := os.ReadFile(path)
		must(err)
		hash := sha256.Sum256(raw)
		sources[path] = hex.EncodeToString(hash[:])
	}
	for _, path := range []string{"migrations/255_opcpay_payment_orders.sql", "tools/opcpay-e2e/orders.go", "internal/service/payment_currency.go", "internal/service/payment_stats.go", "migrations/251_opcpay_operations.sql", "migrations/252_opcpay_provider_binding.sql", "internal/payment/crypto.go", "internal/service/payment_order.go", "internal/opcpay/client.go", "internal/service/opcpay_provider_config.go", "internal/service/payment_config_providers.go", "../frontend/src/components/payment/providerConfig.ts", "../frontend/src/components/payment/PaymentProviderDialog.vue", "../frontend/src/views/admin/SettingsView.vue", "internal/service/opcpay_service.go", "internal/handler/opcpay_handler.go", "internal/server/routes/payment.go", "tools/opcpay-e2e/main.go", "../frontend/src/components/payment/OPCPayPanel.vue", "../frontend/src/api/opcpay.ts", "../frontend/src/views/user/PaymentView.vue", "../frontend/e2e/opcpay.browser.mjs", "../frontend/e2e/opcpay-preview.mjs", "../frontend/e2e/opcpay.mjs"} {
		raw, err := os.ReadFile(path)
		must(err)
		hash := sha256.Sum256(raw)
		sources[path] = hex.EncodeToString(hash[:])
	}
	for _, path := range []string{"internal/handler/payment_handler.go", "internal/handler/admin/payment_handler.go", "internal/service/payment_refund.go", "../frontend/src/components/payment/OrderTable.vue", "../frontend/src/components/admin/payment/AdminOrderTable.vue", "../frontend/src/components/admin/payment/AdminOrderDetail.vue", "../frontend/src/views/admin/orders/AdminOrdersView.vue", "../frontend/src/views/user/UserOrdersView.vue", "../frontend/src/types/payment.ts", "../frontend/src/i18n/locales/zh/misc.ts", "../frontend/src/i18n/locales/en/misc.ts", "../docs/PAYMENT.md"} {
		raw, err := os.ReadFile(path)
		must(err)
		hash := sha256.Sum256(raw)
		sources[path] = hex.EncodeToString(hash[:])
	}
	if os.Getenv("OPCPAY_E2E_PREVIEW") == "1" {
		must(os.WriteFile("../output/opcpay-e2e/preview.json", []byte(fmt.Sprintf(`{"api_url":%q,"provider_id":%d}`, api.URL, providerID)), 0644))
		fmt.Println("模拟接口保持运行，供浏览器验收使用")
		for {
			time.Sleep(time.Second)
		}
	}
	hash := sha256.Sum256(migration)
	report := map[string]any{"mode": "simulated-provider-real-postgresql-http", "source_sha256": sources, "passed": len(checks), "failed": 0, "checks": checks, "migration_sha256": hex.EncodeToString(hash[:]), "balance_usd": balance(), "provider_consumptions": deductions, "provider_requests": calls, "real_payments": false, "real_jwt_login": false, "os_process_restart": false, "api_base": opcpay.BaseURL}
	report["balance_history"] = historyReceipt
	report["orders"] = ordersReceipt
	raw, _ := json.MarshalIndent(report, "", "  ")
	out := os.Getenv("OPCPAY_E2E_REPORT")
	if out == "" {
		out = "../output/opcpay-e2e/checks.json"
	}
	must(os.MkdirAll(strings.TrimSuffix(out, "/checks.json"), 0755))
	must(os.WriteFile(out, raw, 0644))
}
