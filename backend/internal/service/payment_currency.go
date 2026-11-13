package service

import (
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func paymentProviderConfigCurrency(providerKey string, cfg map[string]string) string {
	switch strings.TrimSpace(providerKey) {
	case payment.TypeStripe, payment.TypeAirwallex:
		currency, err := payment.NormalizePaymentCurrency(cfg["currency"])
		if err == nil {
			return currency
		}
	}
	return payment.DefaultPaymentCurrency
}

func PaymentOrderCurrency(order *dbent.PaymentOrder) string {
	if order != nil && order.PaymentType == payment.TypeOPCPay && !PaymentOrderPayAmountKnown(order) {
		return ""
	}
	if snapshot := psOrderProviderSnapshot(order); snapshot != nil {
		if currency, err := payment.NormalizePaymentCurrency(snapshot.Currency); err == nil {
			return currency
		}
	}
	return payment.DefaultPaymentCurrency
}

// OPCPay historical deliveries have no payment quote. Zero is a storage
// placeholder, never a claim that the purchase was free.
func PaymentOrderPayAmountKnown(order *dbent.PaymentOrder) bool {
	if order == nil {
		return false
	}
	if order.PaymentType != payment.TypeOPCPay {
		return true
	}
	known, _ := order.ProviderSnapshot["pay_amount_known"].(bool)
	return known
}
