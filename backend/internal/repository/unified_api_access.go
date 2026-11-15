package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// EnsureUnifiedAPIKey serializes provisioning on the account, not the key name.
// Deleted rows intentionally remain eligible for lookup: deletion is a tombstone.
func (r *apiKeyRepository) EnsureUnifiedAPIKey(ctx context.Context, userID int64, credential string, isDefault bool, maxActive int) (int64, error) {
	db, ok := r.sql.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return 0, fmt.Errorf("unified access requires transactional persistence")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var uid, id int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&uid); err != nil {
		return 0, err
	}
	if isDefault {
		err = tx.QueryRowContext(ctx, `SELECT id FROM api_keys WHERE user_id=$1 AND is_default`, uid).Scan(&id)
		if err == nil {
			return id, tx.Commit()
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
	}
	if !isDefault && maxActive > 0 {
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM api_keys WHERE user_id=$1 AND deleted_at IS NULL`, uid).Scan(&count); err != nil {
			return 0, err
		}
		if count >= maxActive {
			return 0, service.ErrAPIKeyCountExceeded
		}
	}
	name := "API key"
	if isDefault {
		name = "Default API key"
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO api_keys(user_id,key,name,group_id,status,access_mode,is_default)
 SELECT $1,$2,$3,group_id,'active','unified',$4 FROM unified_api_group WHERE singleton RETURNING id`, uid, credential, name, isDefault).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
