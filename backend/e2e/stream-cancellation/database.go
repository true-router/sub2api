// 真实 PostgreSQL、正式迁移、用量仓储与账本的端到端验收。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "github.com/lib/pq"
)

func checkDatabase() error {
	db, err := sql.Open("postgres", os.Getenv("E2E_DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	var name string
	if err = db.QueryRowContext(ctx, "SELECT current_database()").Scan(&name); err != nil {
		return err
	}
	if name != "stream_cancellation_e2e" {
		return fmt.Errorf("refuse non-fixture database")
	}
	if err = repository.ApplyMigrations(ctx, db); err != nil {
		return err
	}
	var user, key, account int64
	if err = db.QueryRowContext(ctx, `INSERT INTO users(email,password_hash,balance) VALUES('cancel@fixture.invalid','synthetic',10) RETURNING id`).Scan(&user); err != nil {
		return err
	}
	if err = db.QueryRowContext(ctx, `INSERT INTO accounts(name,platform,type) VALUES('cancel-fixture','anthropic','apikey') RETURNING id`).Scan(&account); err != nil {
		return err
	}
	if err = db.QueryRowContext(ctx, `INSERT INTO api_keys(user_id,key,name) VALUES($1,'sk-cancel-fixture','cancel-fixture') RETURNING id`, user).Scan(&key); err != nil {
		return err
	}
	repo := repository.NewUsageLogRepository(nil, db)
	billing := repository.NewUsageBillingRepository(nil, db)
	incomplete := true
	log := &service.UsageLog{UserID: user, APIKeyID: key, AccountID: account, RequestID: "cancel-partial",
		Model: "claude-sonnet-4-6", InputTokens: 10, OutputTokens: 5, Stream: true, RequestType: service.RequestTypeStream,
		UsageIncomplete: &incomplete, RateMultiplier: 1, CreatedAt: time.Now()}
	created, err := repo.Create(ctx, log)
	if err != nil {
		return err
	}
	if !created {
		return fmt.Errorf("first usage missing")
	}
	stored, err := repo.GetByID(ctx, log.ID)
	if err != nil {
		return err
	}
	if stored.UsageIncomplete == nil || !*stored.UsageIncomplete || stored.OutputTokens != 5 {
		return fmt.Errorf("partial usage lost")
	}
	created, err = repo.Create(ctx, log)
	if err != nil {
		return err
	}
	if created {
		return fmt.Errorf("duplicate usage inserted")
	}
	// 合成定价：仅对已观测到的 5 个输出 token 结算 0.05，不推算未观测用量。
	cmd := &service.UsageBillingCommand{RequestID: log.RequestID, APIKeyID: key, UserID: user, AccountID: account,
		Model: log.Model, AccountType: service.AccountTypeAPIKey, InputTokens: 10, OutputTokens: 5, BalanceCost: .05}
	first, err := billing.Apply(ctx, cmd)
	if err != nil {
		return err
	}
	second, err := billing.Apply(ctx, cmd)
	if err != nil {
		return err
	}
	if !first.Applied || second.Applied {
		return fmt.Errorf("billing deduplication failed")
	}
	var balance string
	if err = db.QueryRowContext(ctx, "SELECT balance::text FROM users WHERE id=$1", user).Scan(&balance); err != nil {
		return err
	}
	if balance != "9.95000000" {
		return fmt.Errorf("unexpected balance: %s", balance)
	}
	complete := false
	log.ID = 0
	log.RequestID = "cancel-complete"
	log.UsageIncomplete = &complete
	if _, err = repo.Create(ctx, log); err != nil {
		return err
	}
	stored, err = repo.GetByID(ctx, log.ID)
	if err != nil {
		return err
	}
	if stored.UsageIncomplete == nil || *stored.UsageIncomplete {
		return fmt.Errorf("complete usage lost")
	}
	log.ID = 0
	log.RequestID = "legacy-unknown"
	log.UsageIncomplete = nil
	if _, err = repo.Create(ctx, log); err != nil {
		return err
	}
	stored, err = repo.GetByID(ctx, log.ID)
	if err != nil {
		return err
	}
	if stored.UsageIncomplete != nil {
		return fmt.Errorf("legacy unknown became complete")
	}
	// 验证无 request_id 的单条插入分支，批量分支已由以上真实 Create 路径覆盖。
	log.ID = 0
	log.RequestID = ""
	log.UsageIncomplete = &incomplete
	if _, err = repo.Create(ctx, log); err != nil {
		return err
	}
	// 验证异步最佳努力写入与去重分支使用相同的完整性字段。
	log.RequestID = "best-effort-incomplete"
	if err = repo.(interface {
		CreateBestEffort(context.Context, *service.UsageLog) error
	}).CreateBestEffort(ctx, log); err != nil {
		return err
	}
	var storedFlag sql.NullBool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		err = db.QueryRowContext(ctx, "SELECT usage_incomplete FROM usage_logs WHERE request_id=$1 AND api_key_id=$2", log.RequestID, key).Scan(&storedFlag)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	if !storedFlag.Valid || !storedFlag.Bool {
		return fmt.Errorf("best effort completeness lost")
	}
	if err = repo.(interface {
		CreateBestEffort(context.Context, *service.UsageLog) error
	}).CreateBestEffort(ctx, log); err != nil {
		return err
	}
	// 旧版本只写已存在的列，新增 nullable 列必须允许其继续写入。
	if _, err = db.ExecContext(ctx, `INSERT INTO usage_logs(user_id,api_key_id,account_id,model,input_tokens,output_tokens,total_cost,actual_cost,rate_multiplier)
        VALUES($1,$2,$3,'legacy-fixture',0,0,0,0,1)`, user, key, account); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"result": "PASS", "migration": 260, "partial_usage_persisted": true,
		"complete_usage_persisted": true, "unknown_usage_remains_null": true, "usage_deduplicated": true,
		"billing_applied_once": true, "synthetic_balance": balance, "legacy_insert_compatible": true})
}
