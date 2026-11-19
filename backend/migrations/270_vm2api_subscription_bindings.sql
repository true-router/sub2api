ALTER TABLE groups ADD COLUMN subscription_usage_source varchar NOT NULL DEFAULT 'local' CHECK (subscription_usage_source IN ('local','vm2api'));

CREATE TABLE IF NOT EXISTS vm2api_subscription_bindings (
    user_id bigint PRIMARY KEY REFERENCES users(id),
    remote_user_id text NOT NULL UNIQUE,
    remote_username text NOT NULL,
    origin text NOT NULL,
    account_id bigint NOT NULL UNIQUE REFERENCES accounts(id),
    group_id bigint NOT NULL UNIQUE REFERENCES groups(id)
);

-- Protect dedicated groups across admin, redeem and bulk assignment paths.
CREATE OR REPLACE FUNCTION check_vm2api_subscription_owner() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM vm2api_subscription_bindings
               WHERE group_id = NEW.group_id AND user_id <> NEW.user_id) THEN
        RAISE EXCEPTION 'vm2api subscription owner mismatch' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER vm2api_subscription_owner BEFORE INSERT OR UPDATE OF group_id, user_id
ON user_subscriptions FOR EACH ROW EXECUTE FUNCTION check_vm2api_subscription_owner();
