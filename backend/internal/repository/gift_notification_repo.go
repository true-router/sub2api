package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type giftNotificationRepository struct{ db *sql.DB }

func NewGiftNotificationRepository(db *sql.DB) service.GiftNotificationRepository {
	return &giftNotificationRepository{db: db}
}
func (r *giftNotificationRepository) List(ctx context.Context, userID int64) ([]service.GiftNotification, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,amount,source,created_at,read_at FROM gift_notifications WHERE user_id=$1 AND (read_at IS NULL OR id IN (SELECT id FROM gift_notifications WHERE user_id=$1 AND read_at IS NOT NULL ORDER BY id DESC LIMIT 20)) ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.GiftNotification{}
	for rows.Next() {
		var item service.GiftNotification
		if err = rows.Scan(&item.ID, &item.Amount, &item.Source, &item.CreatedAt, &item.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
func (r *giftNotificationRepository) MarkRead(ctx context.Context, userID, id int64) error {
	var found int64
	err := r.db.QueryRowContext(ctx, `UPDATE gift_notifications SET read_at=COALESCE(read_at,NOW()) WHERE id=$1 AND user_id=$2 RETURNING id`, id, userID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrNotificationNotFound
	}
	return err
}
