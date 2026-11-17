package service

// 夹具失败方式：把 PostgreSQL 账本函数发给 SQLite；事务中丢失 SQL 边界；
// 多次调用被静默接受；误将 SQL mock 当作真实账本余额证明。
// 本夹具只对 wallet_* 调用使用已有 sqlmock。普通订单与审计仍由 SQLite 执行。
// 它不实现账本规则。真实余额、幂等和退款由 production-billing-e2e 验证。
import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"entgo.io/ent/dialect"
)

type paymentWalletFixtureDriver struct {
	dialect.Driver
	wallet *sql.DB
}

type paymentWalletFixtureSQL interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (d *paymentWalletFixtureDriver) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.HasPrefix(query, "SELECT wallet_") {
		return d.wallet.ExecContext(ctx, query, args...)
	}
	base, ok := d.Driver.(paymentWalletFixtureSQL)
	if !ok {
		return nil, fmt.Errorf("fixture driver does not support context SQL")
	}
	return base.ExecContext(ctx, query, args...)
}
func (d *paymentWalletFixtureDriver) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.HasPrefix(query, "SELECT wallet_") {
		return d.wallet.QueryContext(ctx, query, args...)
	}
	base, ok := d.Driver.(paymentWalletFixtureSQL)
	if !ok {
		return nil, fmt.Errorf("fixture driver does not support context SQL")
	}
	return base.QueryContext(ctx, query, args...)
}
func (d *paymentWalletFixtureDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	tx, err := d.Driver.Tx(ctx)
	if err != nil {
		return nil, err
	}
	return &paymentWalletFixtureTx{Tx: tx, wallet: d.wallet}, nil
}

type paymentWalletFixtureTx struct {
	dialect.Tx
	wallet *sql.DB
}

func (tx *paymentWalletFixtureTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.HasPrefix(query, "SELECT wallet_") {
		return tx.wallet.ExecContext(ctx, query, args...)
	}
	base, ok := tx.Tx.(paymentWalletFixtureSQL)
	if !ok {
		return nil, fmt.Errorf("fixture driver does not support context SQL")
	}
	return base.ExecContext(ctx, query, args...)
}
func (tx *paymentWalletFixtureTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.HasPrefix(query, "SELECT wallet_") {
		return tx.wallet.QueryContext(ctx, query, args...)
	}
	base, ok := tx.Tx.(paymentWalletFixtureSQL)
	if !ok {
		return nil, fmt.Errorf("fixture driver does not support context SQL")
	}
	return base.QueryContext(ctx, query, args...)
}
