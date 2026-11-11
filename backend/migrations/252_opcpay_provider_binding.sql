-- Bind immutable payment operations to their original provider instance.
ALTER TABLE opcpay_operations ADD COLUMN IF NOT EXISTS provider_instance_id BIGINT;
ALTER TABLE opcpay_operations ADD COLUMN IF NOT EXISTS provider_key_hash VARCHAR(64) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS opcpay_pending_provider_operations
ON opcpay_operations(provider_instance_id) WHERE status NOT IN ('COMPLETED','FAILED','CLOSED');
CREATE UNIQUE INDEX IF NOT EXISTS opcpay_one_enabled_provider
ON payment_provider_instances(provider_key) WHERE provider_key='opcpay' AND enabled=true;
