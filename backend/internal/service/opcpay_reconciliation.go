package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/opcpay"
)

// Reconcile runs without a browser or authenticated session. The existing payment
// worker elects one leader; persistent due times also fence concurrent sweeps.
// Ten candidates use at most forty provider calls per one-minute sweep.
func (s *OPCPayService) Reconcile(ctx context.Context) (int, error) {
	if s == nil || s.DB == nil {
		return 0, nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT o.business_id,o.user_id FROM opcpay_operations o
 WHERE o.next_reconcile_at<=CURRENT_TIMESTAMP AND
 ((o.kind='consume' AND o.status NOT IN ('COMPLETED','FAILED')) OR
 (o.kind='purchase' AND o.status IN ('NEW','READY','CLOSED') AND NOT EXISTS
  (SELECT 1 FROM opcpay_operations c WHERE c.user_id=o.user_id AND c.action_id=o.action_id AND c.kind='consume')))
 ORDER BY CASE WHEN o.kind='consume' THEN 0 ELSE 1 END,o.next_reconcile_at,o.business_id LIMIT 10`)
	if err != nil {
		return 0, errors.New("无法读取待核销充值")
	}
	type candidate struct {
		id   string
		user int64
	}
	var batch []candidate
	for rows.Next() {
		var c candidate
		if err = rows.Scan(&c.id, &c.user); err != nil {
			_ = rows.Close()
			return 0, err
		}
		batch = append(batch, c)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return 0, err
	}
	completed := 0
	var failures []error
	for _, c := range batch {
		if ctx.Err() != nil {
			return completed, ctx.Err()
		}
		// Claim before the network call. Crashes defer this item instead of losing it.
		result, claimErr := s.DB.ExecContext(ctx, `UPDATE opcpay_operations SET next_reconcile_at=CURRENT_TIMESTAMP+INTERVAL '1 minute'
   WHERE business_id=$1 AND next_reconcile_at<=CURRENT_TIMESTAMP`, c.id)
		if claimErr != nil {
			return completed, claimErr
		}
		claimed, claimErr := result.RowsAffected()
		if claimErr != nil {
			return completed, claimErr
		}
		if claimed == 0 {
			continue
		}
		itemCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		operation, reconcileErr := s.reconcileOperation(itemCtx, c.user, c.id)
		cancel()
		if reconcileErr != nil {
			// Provider bodies and credentials never enter logs.
			slog.Warn("opcpay.reconciliation_deferred", "business_id", c.id, "user_id", c.user)
			failures = append(failures, reconcileErr)
		} else if operation != nil && operation.Status == "COMPLETED" {
			completed++
		}
	}
	return completed, errors.Join(failures...)
}

func (s *OPCPayService) reconcileOperation(ctx context.Context, userID int64, id string) (*OPCPayOperation, error) {
	o, err := s.find(ctx, userID, "business_id=$2", id)
	if err != nil || o == nil {
		return nil, err
	}
	if o.Kind == "consume" {
		return s.Resume(ctx, userID, o.ID)
	}
	if !s.enabled(ctx) {
		return nil, nil
	}
	if o.ProviderID <= 0 || o.Unit != "POINT" || o.Amount <= 0 || o.Amount > 99999999 || math.Trunc(o.Amount) != o.Amount || o.Credit != o.Amount {
		return nil, errors.New("原购买记录不满足积分核销约束")
	}
	inst, pay, _, err := s.Config.opcpayProvider(ctx, o.ProviderID)
	if err != nil || inst == nil || !inst.Enabled {
		return nil, errors.New("原一人付服务商不可用")
	}
	cfg, err := s.Config.decryptConfig(inst.Config)
	if err != nil || opcpayKeyHash(cfg["apiKey"]) != o.KeyHash {
		return nil, errors.New("原一人付服务商凭证已变更")
	}
	tx := &opcpay.Transaction{}
	if err = pay.Call(ctx, "/transactions/query", map[string]any{"user_id": strconv.FormatInt(userID, 10), "idempotency_key": o.ID}, tx); err != nil {
		return nil, err
	}
	checkedAt, timeErr := time.Parse(time.RFC3339, tx.CheckedAt)
	if tx.Status != "PAID" {
		return nil, nil
	}
	if tx.UserID != strconv.FormatInt(userID, 10) || tx.Key != o.ID || strings.TrimSpace(tx.TradeNo) == "" || tx.Source != "ALIPAY_PURCHASE_QUERY_V3" || timeErr != nil || checkedAt.After(time.Now().Add(time.Minute)) {
		return nil, errors.New("付款证明与原购买记录不匹配")
	}
	access := &opcpay.Access{}
	if err = pay.Call(ctx, "/access", map[string]any{"user_id": strconv.FormatInt(userID, 10), "amount": o.Amount}, access); err != nil {
		return nil, err
	}
	if (access.Decision != "" && access.Decision != "ALLOW") || !access.Granted || !access.RequiresConsumption || access.Unit != o.Unit || access.Remaining < o.Amount {
		return nil, nil
	}
	random := make([]byte, 16)
	if _, err = rand.Read(random); err != nil {
		return nil, err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO opcpay_operations(business_id,user_id,provider_instance_id,provider_key_hash,action_id,kind,
  sku_id,amount,quota_unit,credit_usd,reason,purchase_price,purchase_currency,payment_trade_no)
 SELECT $2,user_id,provider_instance_id,provider_key_hash,action_id,'consume',sku_id,amount,quota_unit,credit_usd,
  'sub2api balance credit',purchase_price,purchase_currency,$3 FROM opcpay_operations p WHERE p.business_id=$1
  AND NOT EXISTS(SELECT 1 FROM opcpay_operations c WHERE c.user_id=p.user_id AND c.action_id=p.action_id AND c.kind='consume')
 ON CONFLICT DO NOTHING`, o.ID, "sub2_opc_"+hex.EncodeToString(random), tx.TradeNo)
	if err != nil {
		return nil, errors.New("无法保存自动核销操作")
	}
	consume, err := s.find(ctx, userID, "kind='consume' AND action_id=$2", o.Action)
	if err != nil {
		return nil, err
	}
	if consume == nil {
		return nil, errors.New("已有核销操作或支付流水冲突，等待对账")
	}

	return s.Resume(ctx, userID, consume.ID)
}

// Status only reads our durable record. Polling it never calls the payment provider.
func (s *OPCPayService) Status(ctx context.Context, userID int64, id string) (*OPCPayOperation, error) {
	o, err := s.find(ctx, userID, "business_id=$2", id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("操作不存在")
	}
	if o.Kind == "purchase" {
		consume, err := s.find(ctx, userID, "kind='consume' AND action_id=$2", o.Action)
		if err != nil {
			return nil, err
		}
		if consume != nil {
			return consume, nil
		}
	}
	return o, nil
}
