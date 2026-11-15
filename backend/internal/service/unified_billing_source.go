package service

import (
	"context"
	"fmt"
)

type UnifiedBillingSource struct {
	RequestID        string  `json:"request_id"`
	APIKeyID         int64   `json:"api_key_id"`
	SubscriptionID   *int64  `json:"subscription_id,omitempty"`
	SubscriptionName string  `json:"subscription_name,omitempty"`
	SubscriptionCost float64 `json:"subscription_cost"`
	BonusCost        float64 `json:"bonus_cost"`
	PaidCost         float64 `json:"paid_cost"`
	UnverifiedCost   float64 `json:"unverified_cost"`
}

type unifiedBillingSourceReader interface {
	UnifiedBillingSources(context.Context, int64, []string) ([]UnifiedBillingSource, error)
}

func (s *UsageService) UnifiedBillingSources(ctx context.Context, userID int64, records []UsageLog) (map[string]*UnifiedBillingSource, error) {
	out := map[string]*UnifiedBillingSource{}
	repo, ok := s.usageRepo.(unifiedBillingSourceReader)
	if !ok || len(records) == 0 {
		return out, nil
	}
	ids := make([]string, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.RequestID)
	}
	sources, err := repo.UnifiedBillingSources(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	for i := range sources {
		source := &sources[i]
		out[UnifiedBillingSourceKey(source.RequestID, source.APIKeyID)] = source
	}
	return out, nil
}
func UnifiedBillingSourceKey(requestID string, keyID int64) string {
	return requestID + ":" + fmt.Sprint(keyID)
}
