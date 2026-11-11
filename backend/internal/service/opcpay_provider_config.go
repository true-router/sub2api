package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentproviderinstance"
	"github.com/Wei-Shaw/sub2api/internal/opcpay"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const opcpayConfigPrefix = "opcpay:v1:"

func validateOPCPayConfig(cfg map[string]string) error {
	key := strings.TrimSpace(cfg["apiKey"])
	if key == "" || len(key) > 4096 || strings.ContainsAny(key, "\r\n") {
		return infraerrors.BadRequest("OPCPAY_API_KEY_REQUIRED", "请填写一人付 API Key")
	}
	if origin := strings.TrimSpace(cfg["publicOrigin"]); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return infraerrors.BadRequest("OPCPAY_ORIGIN_INVALID", "回跳网站地址必须是 HTTPS 来源，且不包含路径")
		}
	}
	return nil
}
func (s *PaymentConfigService) encodeProviderConfig(providerKey string, cfg map[string]string) (string, error) {
	if providerKey != payment.TypeOPCPay {
		return s.encryptConfig(cfg)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", errors.New("无法保存一人付配置")
	}
	encrypted, err := payment.Encrypt(string(raw), s.encryptionKey)
	if err != nil {
		return "", infraerrors.ServiceUnavailable("OPCPAY_ENCRYPTION_UNAVAILABLE", "服务端加密配置不可用，无法安全保存 API Key")
	}
	return opcpayConfigPrefix + encrypted, nil
}
func (s *PaymentConfigService) opcpayProvider(ctx context.Context, id int64) (*dbent.PaymentProviderInstance, *opcpay.Client, string, error) {
	if s == nil || s.entClient == nil {
		return nil, nil, "", errors.New("一人付尚未配置")
	}
	q := s.entClient.PaymentProviderInstance.Query().Where(paymentproviderinstance.ProviderKeyEQ(payment.TypeOPCPay))
	if id > 0 {
		q = q.Where(paymentproviderinstance.IDEQ(id))
	} else {
		q = q.Where(paymentproviderinstance.EnabledEQ(true))
	}
	inst, err := q.Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil, "", nil
	}
	if err != nil {
		return nil, nil, "", errors.New("无法读取一人付服务商")
	}
	cfg, err := s.decryptConfig(inst.Config)
	if err != nil {
		return nil, nil, "", errors.New("无法解密一人付配置")
	}
	return inst, opcpay.NewClient(strings.TrimSpace(cfg["apiKey"])), strings.TrimRight(strings.TrimSpace(cfg["publicOrigin"]), "/"), nil
}
func (s *PaymentConfigService) validateOPCPayInstance(ctx context.Context, id int64, types string, enabled, refund, userRefund bool) error {
	if types != "opcpay" {
		return infraerrors.BadRequest("OPCPAY_TYPES_INVALID", "一人付只支持积分充值卡")
	}
	if refund || userRefund {
		return infraerrors.BadRequest("OPCPAY_REFUND_UNSUPPORTED", "一人付未接入退款能力")
	}
	if enabled {
		count, err := s.entClient.PaymentProviderInstance.Query().Where(paymentproviderinstance.ProviderKeyEQ(payment.TypeOPCPay), paymentproviderinstance.EnabledEQ(true), paymentproviderinstance.IDNEQ(id)).Count(ctx)
		if err != nil {
			return errors.New("无法读取一人付服务商")
		}
		if count > 0 {
			return infraerrors.Conflict("OPCPAY_ALREADY_ENABLED", "只能启用一个一人付服务商")
		}
	}
	return nil
}
