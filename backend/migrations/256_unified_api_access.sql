-- Unified credentials opt in explicitly. Existing keys retain legacy behavior.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS access_mode varchar(20) NOT NULL DEFAULT 'legacy';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS is_default boolean NOT NULL DEFAULT false;
ALTER TABLE api_keys ADD CONSTRAINT api_keys_access_mode_check CHECK (access_mode IN ('legacy', 'unified'));
CREATE UNIQUE INDEX IF NOT EXISTS api_keys_one_default_per_user ON api_keys(user_id) WHERE is_default;
-- A soft-deleted default row remains the provisioning tombstone.
CREATE TABLE unified_api_group (
 singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
 group_id bigint NOT NULL UNIQUE REFERENCES groups(id) ON DELETE RESTRICT
);
WITH inserted AS (
 INSERT INTO groups(name,description,platform,subscription_type,rate_multiplier,is_exclusive,status)
 VALUES ('__system_unified_access__','Managed unified API identity; request routes preserve account entitlements.','composite','standard',1,true,'active')
 RETURNING id
)
INSERT INTO unified_api_group(group_id) SELECT id FROM inserted;
