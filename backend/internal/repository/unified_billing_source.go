package repository

import (
	"context"
	"database/sql"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *usageLogRepository) UnifiedBillingSources(ctx context.Context, userID int64, ids []string) ([]service.UnifiedBillingSource, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT request_id,api_key_id,subscription_id,subscription_name,subscription_cost,bonus_cost,paid_cost,unverified_cost FROM unified_billing_sources WHERE user_id=$1 AND request_id=ANY($2)`, userID, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.UnifiedBillingSource{}
	for rows.Next() {
		var row service.UnifiedBillingSource
		var sub sql.NullInt64
		if err = rows.Scan(&row.RequestID, &row.APIKeyID, &sub, &row.SubscriptionName, &row.SubscriptionCost, &row.BonusCost, &row.PaidCost, &row.UnverifiedCost); err != nil {
			return nil, err
		}
		if sub.Valid {
			row.SubscriptionID = &sub.Int64
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
