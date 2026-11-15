-- Immutable, request-correlated source receipts for unified credentials.
CREATE TABLE unified_billing_sources (
 request_id text NOT NULL,
 api_key_id bigint NOT NULL,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 subscription_id bigint,
 subscription_name text NOT NULL DEFAULT '',
 subscription_cost numeric(20,8) NOT NULL DEFAULT 0,
 bonus_cost numeric(20,8) NOT NULL DEFAULT 0,
 paid_cost numeric(20,8) NOT NULL DEFAULT 0,
 unverified_cost numeric(20,8) NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(request_id,api_key_id)
);
CREATE INDEX unified_billing_sources_user ON unified_billing_sources(user_id,created_at);
