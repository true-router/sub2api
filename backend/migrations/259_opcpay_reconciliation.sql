-- Dialog visibility never controls payment delivery. Keep durable retry scheduling.
ALTER TABLE opcpay_operations
    ADD COLUMN IF NOT EXISTS next_reconcile_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS payment_trade_no TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS opcpay_reconciliation_due
    ON opcpay_operations(next_reconcile_at, business_id);
-- A provider transaction must not fund two different consumption operations.
CREATE UNIQUE INDEX IF NOT EXISTS opcpay_consumption_trade_unique
    ON opcpay_operations(provider_instance_id,provider_key_hash,payment_trade_no)
    WHERE kind='consume' AND payment_trade_no<>'';
