package service

import (
	"context"
	"time"
)

// Only committed deliveries count as top-ups. Purchase entries and confirmed
// consumptions awaiting delivery must not appear as credited funds.
func (s *adminServiceImpl) opcpayBalanceHistoryTotals(ctx context.Context, userID int64) (int64, float64, error) {
	if s.entClient == nil {
		return 0, 0, nil
	}
	rows, err := s.entClient.QueryContext(ctx, `
SELECT COUNT(*), COALESCE(SUM(credit_usd), 0)::double precision
FROM opcpay_operations
WHERE user_id = $1 AND kind = 'consume' AND status = 'COMPLETED'`, userID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	var count int64
	var amount float64
	if rows.Next() {
		if err := rows.Scan(&count, &amount); err != nil {
			return 0, 0, err
		}
	}
	return count, amount, rows.Err()
}

func (s *adminServiceImpl) sumUserBalanceRecharges(ctx context.Context, userID int64) (float64, error) {
	amount, err := s.redeemCodeRepo.SumPositiveBalanceByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	_, opcpayAmount, err := s.opcpayBalanceHistoryTotals(ctx, userID)
	if err != nil {
		return 0, err
	}
	return amount + opcpayAmount, nil
}

func (s *adminServiceImpl) listOPCPayBalanceHistory(ctx context.Context, userID int64, offset, limit int) ([]RedeemCode, int64, error) {
	total, _, err := s.opcpayBalanceHistoryTotals(ctx, userID)
	if err != nil || total == 0 {
		return []RedeemCode{}, total, err
	}
	rows, err := s.entClient.QueryContext(ctx, `
SELECT business_id, credit_usd::double precision, created_at, updated_at
FROM opcpay_operations
WHERE user_id = $1 AND kind = 'consume' AND status = 'COMPLETED'
ORDER BY updated_at DESC, business_id DESC
OFFSET $2 LIMIT $3`, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	codes := make([]RedeemCode, 0)
	for rows.Next() {
		var code string
		var amount float64
		var createdAt, completedAt time.Time
		if err := rows.Scan(&code, &amount, &createdAt, &completedAt); err != nil {
			return nil, 0, err
		}
		// This is a read-only history projection, not a redeem-code entity.
		// Its identity is the record type plus the unique business ID in Code.
		codes = append(codes, RedeemCode{
			Code: code, Type: RedeemTypeOPCPayBalance, Value: amount,
			Status: StatusUsed, UsedBy: &userID, UsedAt: &completedAt, CreatedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return codes, total, nil
}
