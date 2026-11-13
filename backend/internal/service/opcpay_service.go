package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/opcpay"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var opcpayAction = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,80}$`)

// This rejection occurs before a consumption operation is persisted or submitted.
var ErrOPCPayPaymentUnverified = infraerrors.Conflict("OPCPAY_PAYMENT_UNVERIFIED", "验证失败，请重试")

type OPCPayService struct {
	DB     *dbent.Client
	Config *PaymentConfigService
	redeem *RedeemService
}

func (s *PaymentService) OPCPay() *OPCPayService {
	return &OPCPayService{DB: s.entClient, Config: s.configService, redeem: s.redeemService}
}

type OPCPayInfo struct {
	Configured   bool                `json:"configured"`
	Enabled      bool                `json:"enabled"`
	ProviderID   int64               `json:"provider_id"`
	ProviderName string              `json:"provider_name"`
	Capabilities opcpay.Capabilities `json:"capabilities"`
	Product      opcpay.Product      `json:"product"`
	Access       *opcpay.Access      `json:"access,omitempty"`
	CreditUSD    float64             `json:"credit_usd_per_unit"`
	Pending      *OPCPayOperation    `json:"pending,omitempty"`
	keyHash      string
	origin       string
}
type OPCPayOperation struct {
	ID               string         `json:"business_id"`
	UserID           int64          `json:"-"`
	ProviderID       int64          `json:"provider_id"`
	KeyHash          string         `json:"-"`
	Action           string         `json:"action_id"`
	Kind             string         `json:"kind"`
	SKU              string         `json:"sku_id"`
	Callback         string         `json:"-"`
	Amount           float64        `json:"amount"`
	Unit             string         `json:"quota_unit"`
	Credit           float64        `json:"credit_usd"`
	Reason           string         `json:"-"`
	Status           string         `json:"status"`
	PurchasePrice    string         `json:"-"`
	PurchaseCurrency string         `json:"-"`
	Result           *opcpay.Result `json:"result,omitempty"`
}

func (s *OPCPayService) enabled(ctx context.Context) bool {
	if s.Config == nil {
		return false
	}
	cfg, err := s.Config.GetPaymentConfig(ctx)
	if err != nil || !cfg.Enabled || cfg.BalanceDisabled {
		return false
	}
	for _, kind := range cfg.EnabledTypes {
		if kind == payment.TypeOPCPay {
			return true
		}
	}
	return false
}
func (s *OPCPayService) Info(ctx context.Context, userID int64) (*OPCPayInfo, error) {
	info := &OPCPayInfo{}
	if s.Config == nil {
		return info, nil
	}
	inst, pay, origin, err := s.Config.opcpayProvider(ctx, 0)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return info, nil
	}
	info.ProviderID = int64(inst.ID)
	info.ProviderName = inst.Name
	info.Configured = pay.Configured()
	info.Enabled = info.Configured && s.enabled(ctx)
	if !info.Enabled {
		return info, nil
	}
	cfg, err := s.Config.decryptConfig(inst.Config)
	if err != nil {
		return nil, errors.New("无法读取一人付配置")
	}
	info.keyHash = opcpayKeyHash(cfg["apiKey"])
	info.origin = origin
	if err = pay.Call(ctx, "/capabilities", nil, &info.Capabilities); err != nil {
		return nil, err
	}
	if err = pay.Call(ctx, "/product", nil, &info.Product); err != nil {
		return nil, err
	}
	if info.Capabilities.Flags.Access && info.Capabilities.ProviderConfigured && info.Product.Status == "ENABLED" {
		info.Access = &opcpay.Access{}
		if err = pay.Call(ctx, "/access", map[string]any{"user_id": strconv.FormatInt(userID, 10), "amount": 1}, info.Access); err != nil {
			return nil, err
		}
	}
	cards := make([]opcpay.Option, 0)
	if info.Product.Mode == "QUANTITY" && info.Product.Unit == "POINT" && info.Product.Status == "ENABLED" && info.Capabilities.ProviderConfigured && info.Capabilities.Flags.Consume && info.Capabilities.Flags.Access {
		info.CreditUSD = 1
		for _, card := range info.Product.Options {
			if card.SKU != "" && card.Price != "" && card.Currency != "" && card.Mode == "QUANTITY" && card.Unit == "POINT" && card.Quantity > 0 && card.Quantity <= 100000000 && math.Trunc(card.Quantity) == card.Quantity {
				cards = append(cards, card)
			}
		}
	}
	info.Product.Options = cards
	pending, err := s.find(ctx, userID, "status NOT IN ('COMPLETED','FAILED','CLOSED')")
	if err != nil {
		return nil, err
	}
	info.Pending = pending
	return info, nil
}
func opcpayKeyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
func (s *OPCPayService) find(ctx context.Context, userID int64, where string, args ...any) (*OPCPayOperation, error) {
	args = append([]any{userID}, args...)
	rows, err := s.DB.QueryContext(ctx, `SELECT business_id,user_id,COALESCE(provider_instance_id,0),provider_key_hash,action_id,kind,sku_id,callback_url,amount,quota_unit,credit_usd,reason,status,COALESCE(purchase_price::text,''),purchase_currency FROM opcpay_operations WHERE user_id=$1 AND `+where+` ORDER BY CASE WHEN kind='consume' THEN 0 ELSE 1 END,created_at DESC LIMIT 1`, args...)
	if err != nil {
		return nil, errors.New("无法读取一人付操作")
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	o := &OPCPayOperation{}
	if err = rows.Scan(&o.ID, &o.UserID, &o.ProviderID, &o.KeyHash, &o.Action, &o.Kind, &o.SKU, &o.Callback, &o.Amount, &o.Unit, &o.Credit, &o.Reason, &o.Status, &o.PurchasePrice, &o.PurchaseCurrency); err != nil {
		return nil, errors.New("无法读取一人付操作")
	}
	return o, nil
}
func (s *OPCPayService) Start(ctx context.Context, userID int64, kind, action, sku string) (*OPCPayOperation, error) {
	if !opcpayAction.MatchString(action) || (kind != "purchase" && kind != "consume") {
		return nil, errors.New("操作编号无效")
	}
	old, err := s.find(ctx, userID, "kind=$2 AND action_id=$3", kind, action)
	if err != nil {
		return nil, err
	}
	if old != nil {
		if old.SKU != sku {
			return nil, errors.New("重试参数不一致")
		}
		return s.Resume(ctx, userID, old.ID)
	}
	info, err := s.Info(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !info.Enabled || !info.Capabilities.ProviderConfigured || info.Product.Status != "ENABLED" {
		return nil, errors.New("一人付尚未开放")
	}
	o := &OPCPayOperation{UserID: userID, ProviderID: info.ProviderID, KeyHash: info.keyHash, Action: action, Kind: kind, SKU: sku, Amount: 1, Unit: "POINT", Credit: 1, Status: "NEW"}
	if kind == "purchase" {
		if !info.Capabilities.Flags.Purchase {
			return nil, errors.New("购买入口尚未开放")
		}
		found := false
		for _, card := range info.Product.Options {
			if card.SKU == sku {
				o.Amount = card.Quantity
				o.Credit = card.Quantity
				price, err := decimal.NewFromString(card.Price)
				currency, currencyErr := payment.NormalizePaymentCurrency(card.Currency)
				if err != nil || currencyErr != nil || strings.TrimSpace(card.Currency) == "" || !price.IsPositive() || !price.Equal(price.Round(2)) || price.GreaterThanOrEqual(decimal.New(1, 18)) {
					return nil, errors.New("充值卡价格无效，请联系管理员")
				}
				o.PurchasePrice = price.StringFixed(2)
				o.PurchaseCurrency = currency
				found = true
			}
		}
		if !found {
			return nil, errors.New("充值卡无效，请刷新充值卡列表")
		}
		if info.origin != "" {
			o.Callback = info.origin + "/purchase"
		}
	} else {
		if info.Pending != nil && info.Pending.Kind == "consume" {
			return nil, errors.New("请先恢复未完成的充值")
		}
		if info.CreditUSD != 1 {
			return nil, errors.New("该项目尚未开放积分充值卡")
		}
		if sku != "" {
			purchase, err := s.find(ctx, userID, "kind='purchase' AND action_id=$2 AND sku_id=$3", action, sku)
			if err != nil {
				return nil, err
			}
			if purchase == nil || purchase.Status != "READY" || purchase.ProviderID != info.ProviderID || purchase.KeyHash != info.keyHash {
				return nil, errors.New("请先选择充值卡并创建购买入口")
			}
			o.Amount = purchase.Amount
			o.Credit = purchase.Credit
			o.PurchasePrice = purchase.PurchasePrice
			o.PurchaseCurrency = purchase.PurchaseCurrency
		}
		_, pay, _, err := s.Config.opcpayProvider(ctx, o.ProviderID)
		if err != nil || pay == nil {
			return nil, errors.New("无法读取原服务商")
		}
		access := &opcpay.Access{}
		if err = pay.Call(ctx, "/access", map[string]any{"user_id": strconv.FormatInt(userID, 10), "amount": o.Amount}, access); err != nil {
			return nil, err
		}
		if (access.Decision != "" && access.Decision != "ALLOW") || !access.Granted || !access.RequiresConsumption || access.Unit != "POINT" || access.Remaining < o.Amount {
			return nil, ErrOPCPayPaymentUnverified
		}
		o.Reason = "sub2api balance credit"
	}
	random := make([]byte, 16)
	if _, err = rand.Read(random); err != nil {
		return nil, errors.New("无法生成业务编号")
	}
	o.ID = "sub2_opc_" + hex.EncodeToString(random)
	_, err = s.DB.ExecContext(ctx, `INSERT INTO opcpay_operations(business_id,user_id,provider_instance_id,provider_key_hash,action_id,kind,sku_id,callback_url,amount,quota_unit,credit_usd,reason,purchase_price,purchase_currency) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,'')::numeric,$14) ON CONFLICT DO NOTHING`, o.ID, o.UserID, o.ProviderID, o.KeyHash, o.Action, o.Kind, o.SKU, o.Callback, o.Amount, o.Unit, o.Credit, o.Reason, o.PurchasePrice, o.PurchaseCurrency)
	if err != nil {
		return nil, errors.New("无法保存一人付操作")
	}
	saved, err := s.find(ctx, userID, "kind=$2 AND action_id=$3", kind, action)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, errors.New("请先恢复未完成的充值")
	}
	if saved.SKU != sku {
		return nil, errors.New("重试参数不一致")
	}
	return s.Resume(ctx, userID, saved.ID)
}
func (s *OPCPayService) Resume(ctx context.Context, userID int64, id string) (*OPCPayOperation, error) {
	o, err := s.find(ctx, userID, "business_id=$2", id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("操作不存在")
	}
	if o.Status == "COMPLETED" {
		s.invalidate(ctx, userID)
		return o, nil
	}
	if o.Status == "FAILED" || o.Status == "CLOSED" {
		return o, nil
	}
	if o.Kind == "consume" && o.Status == "CONFIRMED" {
		if err = s.credit(ctx, o); err != nil {
			return nil, err
		}
		o.Status = "COMPLETED"
		s.invalidate(ctx, userID)
		return o, nil
	}
	if s.Config == nil || o.ProviderID == 0 {
		return nil, errors.New("原操作未绑定服务商，禁止切换编号重试")
	}
	inst, pay, _, err := s.Config.opcpayProvider(ctx, o.ProviderID)
	if err != nil || inst == nil || !inst.Enabled || !s.enabled(ctx) {
		return nil, errors.New("原一人付服务商未启用")
	}
	cfg, err := s.Config.decryptConfig(inst.Config)
	if err != nil || opcpayKeyHash(cfg["apiKey"]) != o.KeyHash {
		return nil, errors.New("原服务商凭证已变更，请联系管理员恢复原配置")
	}
	body := map[string]any{"user_id": strconv.FormatInt(userID, 10), "idempotency_key": o.ID}
	result := &opcpay.Result{}
	if o.Kind == "purchase" {
		body["sku_id"] = o.SKU
		if o.Callback != "" {
			body["callback_url"] = o.Callback
		}
		err = pay.Call(ctx, "/purchase", body, result)
		if err == nil && result.Key == o.ID && (result.Status == "READY" || result.Status == "FAILED") {
			if result.Status == "READY" {
				if !safeOPCPayPurchaseURL(result.URL) {
					return nil, errors.New("购买地址无效")
				}
				o.Result = result
			}
			_, err = s.DB.ExecContext(ctx, `UPDATE opcpay_operations SET status=$2,updated_at=CURRENT_TIMESTAMP WHERE business_id=$1 AND status NOT IN ('CLOSED','FAILED')`, o.ID, result.Status)
			if err != nil {
				return nil, errors.New("无法保存购买入口结果")
			}
			o.Status = result.Status
			return o, nil
		}
	} else {
		err = pay.Call(ctx, "/consumption", body, result)
		if err != nil || result.Status == "UNKNOWN" {
			body["amount"] = o.Amount
			body["reason"] = o.Reason
			result = &opcpay.Result{}
			err = pay.Call(ctx, "/consume", body, result)
		}
		if err == nil && result.Key == o.ID && result.Status == "SUCCEEDED" && result.Confirmed && result.Consumed == o.Amount && result.Unit == o.Unit {
			_, err = s.DB.ExecContext(ctx, `UPDATE opcpay_operations SET status='CONFIRMED',updated_at=CURRENT_TIMESTAMP WHERE business_id=$1 AND status NOT IN ('COMPLETED','FAILED')`, o.ID)
			if err != nil {
				return nil, errors.New("核销结果待保存，请恢复原操作")
			}
			if err = s.credit(ctx, o); err != nil {
				return nil, err
			}
			o.Status = "COMPLETED"
			s.invalidate(ctx, userID)
			return o, nil
		}
		if err == nil && result.Key == o.ID && result.Status == "FAILED" && !result.Confirmed {
			_, err = s.DB.ExecContext(ctx, `UPDATE opcpay_operations SET status='FAILED',updated_at=CURRENT_TIMESTAMP WHERE business_id=$1 AND status!='COMPLETED'`, o.ID)
			if err != nil {
				return nil, errors.New("无法保存核销结果")
			}
			o.Status = "FAILED"
			return o, nil
		}
	}
	o.Status = "UNKNOWN"
	return o, nil
}
func safeOPCPayPurchaseURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.User == nil && u.Host != "" && (u.Scheme == "https" || u.Scheme == "alipays")
}
func (s *OPCPayService) ClosePurchase(ctx context.Context, userID int64, id string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE opcpay_operations SET status='CLOSED',updated_at=CURRENT_TIMESTAMP WHERE business_id=$1 AND user_id=$2 AND kind='purchase' AND status IN ('NEW','READY')`, id, userID)
	if err != nil {
		return errors.New("无法结束购买入口")
	}
	return nil
}
func (s *OPCPayService) credit(ctx context.Context, o *OPCPayOperation) error {
	tx, err := s.DB.Tx(ctx)
	if err != nil {
		return errors.New("核销待交付，请恢复原操作")
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE opcpay_operations SET status='COMPLETED',updated_at=CURRENT_TIMESTAMP WHERE business_id=$1 AND status='CONFIRMED'`, o.ID)
	if err != nil {
		return errors.New("核销待交付，请恢复原操作")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return errors.New("核销待交付，请恢复原操作")
	}
	if n == 1 {
		res, err = tx.ExecContext(ctx, `SELECT wallet_credit_payment($2,$1,$1,opcpay_record_order($3))`, o.Credit, o.UserID, o.ID)
		if err != nil {
			return errors.New("核销待交付，请恢复原操作")
		}
		_, err = tx.ExecContext(ctx, `UPDATE opcpay_operations SET status='CLOSED',updated_at=CURRENT_TIMESTAMP WHERE user_id=$1 AND action_id=$2 AND kind='purchase'`, o.UserID, o.Action)
		if err != nil {
			return errors.New("核销待交付，请恢复原操作")
		}
	}
	if err = tx.Commit(); err != nil {
		return errors.New("核销待交付，请恢复原操作")
	}
	return nil
}
func (s *OPCPayService) invalidate(ctx context.Context, userID int64) {
	if s.redeem != nil {
		s.redeem.invalidateRedeemCaches(ctx, userID, &RedeemCode{Type: RedeemTypeBalance})
	}
}
