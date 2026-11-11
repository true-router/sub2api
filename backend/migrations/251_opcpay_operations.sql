-- Purchase entries are not paid orders. Credits require a confirmed consumption.
CREATE TABLE IF NOT EXISTS opcpay_operations (
    business_id VARCHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    action_id VARCHAR(80) NOT NULL,
    kind VARCHAR(16) NOT NULL CHECK (kind IN ('purchase', 'consume')),
    sku_id TEXT NOT NULL DEFAULT '',
    callback_url TEXT NOT NULL DEFAULT '',
    amount NUMERIC(20,8) NOT NULL DEFAULT 1 CHECK (amount > 0),
    quota_unit VARCHAR(8) NOT NULL DEFAULT '',
    credit_usd NUMERIC(20,8) NOT NULL DEFAULT 0 CHECK (credit_usd >= 0),
    reason TEXT NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'NEW',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, kind, action_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS opcpay_one_pending_consumption
ON opcpay_operations(user_id) WHERE kind = 'consume' AND status NOT IN ('COMPLETED', 'FAILED');
