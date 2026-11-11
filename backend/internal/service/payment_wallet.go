package service

import (
	"context"
	"fmt"
	"math"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func refundablePaidBalance(u *User) float64 {
	return math.Max(0, u.Balance-u.BonusBalance-u.UnverifiedBalance)
}

// The database reserves principal and the specific order's remaining grants in
// one user-row transaction. A retry returns the persisted reservation.
func (s *PaymentService) reserveWalletRefund(ctx context.Context, p *RefundPlan) (float64, error) {
	client := s.entClient
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	rows, err := client.QueryContext(ctx, "SELECT wallet_reserve_refund($1,$2,$3)", p.Order.UserID, p.OrderID, p.RefundAmount)
	if err != nil {
		return 0, infraerrors.BadRequest("REFUND_BALANCE_UNAVAILABLE", "paid balance is insufficient or order promotional credit is reserved by an active task").WithCause(err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("wallet reservation returned no result")
	}
	var amount float64
	if err := rows.Scan(&amount); err != nil {
		return 0, err
	}
	return amount, rows.Err()
}

func (s *PaymentService) releaseWalletRefund(ctx context.Context, p *RefundPlan) error {
	client := s.entClient
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	_, err := client.ExecContext(ctx, "SELECT wallet_release_refund($1,$2,$3)", p.Order.UserID, p.OrderID, p.BalanceToDeduct)
	return err
}
