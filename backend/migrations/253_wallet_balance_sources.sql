-- Wallet accounting is serialized by the users row lock. Existing writers keep
-- using balance; ordinary credits are promotional and ordinary debits spend
-- promotional grants first. Explicit functions describe payments and holds.
ALTER TABLE users ADD COLUMN IF NOT EXISTS bonus_balance NUMERIC(20,8) NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS unverified_balance NUMERIC(20,8) NOT NULL DEFAULT 0;
UPDATE users SET unverified_balance = GREATEST(balance, 0);

CREATE TABLE wallet_grants (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    order_id BIGINT,
    source TEXT NOT NULL CHECK (source IN ('bonus','unverified')),
    original_amount NUMERIC(20,8) NOT NULL CHECK (original_amount >= 0),
    remaining NUMERIC(20,8) NOT NULL CHECK (remaining >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX wallet_grants_available ON wallet_grants(user_id, source, id) WHERE remaining > 0;
CREATE INDEX wallet_grants_order ON wallet_grants(user_id, order_id) WHERE order_id IS NOT NULL;
INSERT INTO wallet_grants(user_id, source, original_amount, remaining)
SELECT id, 'unverified', unverified_balance, unverified_balance FROM users WHERE unverified_balance > 0;

CREATE TABLE wallet_holds (
    user_id BIGINT NOT NULL REFERENCES users(id),
    hold_key TEXT NOT NULL,
    amount NUMERIC(20,8) NOT NULL CHECK (amount >= 0),
    paid NUMERIC(20,8) NOT NULL CHECK (paid >= 0),
    allocations JSONB NOT NULL DEFAULT '[]',
    state TEXT NOT NULL CHECK (state IN ('held','released','captured')),
    PRIMARY KEY(user_id, hold_key)
);
-- Reconstruct only hold ownership, never historical funding provenance.
DO $$
DECLARE job RECORD; grant_id BIGINT;
BEGIN
    FOR job IN
        WITH claims AS (
            SELECT request_id,api_key_id FROM usage_billing_dedup
            UNION SELECT request_id,api_key_id FROM usage_billing_dedup_archive
        )
        SELECT j.user_id,j.batch_id,ROUND(COALESCE(j.hold_amount,j.estimated_cost)::numeric,8) AS amount
        FROM batch_image_jobs j
        WHERE COALESCE(j.hold_amount,j.estimated_cost)>0
          AND EXISTS (SELECT 1 FROM claims c WHERE c.api_key_id=j.api_key_id AND c.request_id='batch_image_hold:'||j.batch_id)
          AND NOT EXISTS (SELECT 1 FROM claims c WHERE c.api_key_id=j.api_key_id AND c.request_id IN ('batch_image_capture:'||j.batch_id,'batch_image_release:'||j.batch_id))
    LOOP
        INSERT INTO wallet_grants(user_id,source,original_amount,remaining) VALUES (job.user_id,'unverified',job.amount,0) RETURNING id INTO grant_id;
        INSERT INTO wallet_holds(user_id,hold_key,amount,paid,allocations,state)
        VALUES (job.user_id,'batch:'||job.batch_id,job.amount,0,jsonb_build_array(jsonb_build_object('grant_id',grant_id,'amount',job.amount)),'held');
    END LOOP;
    IF EXISTS (SELECT 1 FROM users u WHERE u.frozen_balance<>(SELECT COALESCE(SUM(h.amount),0) FROM wallet_holds h WHERE h.user_id=u.id AND h.state='held')) THEN
        RAISE EXCEPTION 'historical frozen balance does not match outstanding batch claims; reconcile before migrating';
    END IF;
END $$;

CREATE TABLE wallet_journal (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    operation TEXT NOT NULL,
    order_id BIGINT,
    hold_key TEXT,
    delta NUMERIC(20,8) NOT NULL,
    paid_balance NUMERIC(20,8) NOT NULL,
    bonus_balance NUMERIC(20,8) NOT NULL,
    unverified_balance NUMERIC(20,8) NOT NULL,
    allocations JSONB NOT NULL DEFAULT '[]',
    reference TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX wallet_journal_user ON wallet_journal(user_id, id);

-- Command state is transaction-local and consumed by exactly one user write.
CREATE FUNCTION wallet_prepare(uid BIGINT, command JSONB) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM set_config('sub2api.wallet_command', (command || jsonb_build_object('user_id', uid))::text, true);
END $$;

CREATE FUNCTION wallet_account_balance() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    cmd JSONB := COALESCE(NULLIF(current_setting('sub2api.wallet_command', true), ''), '{}')::jsonb;
    op TEXT;
    movement NUMERIC(20,8);
    needed NUMERIC(20,8);
    taken NUMERIC(20,8);
    paid NUMERIC(20,8) := 0;
    gift NUMERIC(20,8);
    allocations JSONB := '[]';
    allocation JSONB;
    grant_row RECORD;
    held wallet_holds%ROWTYPE;
    v_hold_key TEXT;
    v_order_id BIGINT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW.bonus_balance := GREATEST(NEW.balance, 0);
        NEW.unverified_balance := 0;
        RETURN NEW;
    END IF;
    IF cmd ? 'user_id' THEN
        IF (cmd->>'user_id')::BIGINT <> NEW.id THEN
            RAISE EXCEPTION 'wallet command user mismatch';
        END IF;
        PERFORM set_config('sub2api.wallet_command', '', true);
    END IF;
    op := COALESCE(cmd->>'operation', 'adjust');
    v_hold_key := cmd->>'hold_key';
    v_order_id := (cmd->>'order_id')::BIGINT;
    movement := NEW.balance - OLD.balance;

    IF NEW.frozen_balance <> OLD.frozen_balance AND op NOT IN ('reserve_batch','settle_batch') THEN RAISE EXCEPTION 'frozen balance requires wallet hold'; END IF;
    IF op='reserve_batch' AND NEW.frozen_balance-OLD.frozen_balance <> -movement THEN RAISE EXCEPTION 'invalid wallet hold'; END IF;
    IF op='verify_legacy' THEN
        needed := (cmd->>'paid')::NUMERIC+(cmd->>'bonus')::NUMERIC;
        IF needed > OLD.unverified_balance OR needed <= 0 THEN RAISE EXCEPTION 'invalid legacy verification'; END IF;
        FOR grant_row IN SELECT g.id,g.remaining FROM wallet_grants g WHERE g.user_id=NEW.id AND g.source='unverified' AND g.remaining>0 ORDER BY g.id FOR UPDATE LOOP
            EXIT WHEN needed <= 0;
            taken := LEAST(needed,grant_row.remaining);
            UPDATE wallet_grants SET remaining=remaining-taken WHERE id=grant_row.id;
            needed := needed-taken;
        END LOOP;
        gift := (cmd->>'bonus')::NUMERIC;
        IF gift>0 THEN
            INSERT INTO wallet_grants(user_id,order_id,source,original_amount,remaining) VALUES (NEW.id,v_order_id,'bonus',gift,gift);
        END IF;
    ELSIF op = 'release_refund' OR op = 'settle_batch' THEN
        SELECT * INTO held FROM wallet_holds h WHERE h.user_id=NEW.id AND h.hold_key=v_hold_key FOR UPDATE;
        IF NOT FOUND OR held.state <> 'held' THEN
            RAISE EXCEPTION 'wallet hold missing or already settled';
        END IF;
        needed := CASE WHEN op='settle_batch' THEN (cmd->>'actual')::NUMERIC ELSE 0 END;
        IF needed < 0 OR needed > held.amount OR movement <> held.amount-needed THEN RAISE EXCEPTION 'invalid wallet settlement'; END IF;
        IF op='settle_batch' AND OLD.frozen_balance-NEW.frozen_balance <> held.amount THEN RAISE EXCEPTION 'wallet frozen amount mismatch'; END IF;
        -- Held funds use the same gift-first order as ordinary consumption.
        FOR allocation IN SELECT value FROM jsonb_array_elements(held.allocations) LOOP
            taken := LEAST(needed, (allocation->>'amount')::NUMERIC);
            needed := needed - taken;
            UPDATE wallet_grants SET remaining=remaining+(allocation->>'amount')::NUMERIC-taken WHERE id=(allocation->>'grant_id')::BIGINT;
        END LOOP;
        UPDATE wallet_holds h SET state=CASE WHEN op='release_refund' THEN 'released' ELSE 'captured' END
          WHERE h.user_id=NEW.id AND h.hold_key=v_hold_key;
        allocations := held.allocations;
    ELSIF movement < 0 THEN
        needed := -movement;
        IF op='reserve_refund' THEN
            paid := (cmd->>'paid')::NUMERIC;
            IF paid > GREATEST(OLD.balance-OLD.bonus_balance-OLD.unverified_balance, 0) THEN
                RAISE EXCEPTION 'insufficient refundable paid balance';
            END IF;
            needed := needed-paid;
        END IF;
        FOR grant_row IN
            SELECT g.id, g.remaining FROM wallet_grants g
            WHERE g.user_id=NEW.id AND g.remaining>0
              AND (op<>'reserve_refund' OR (g.source='bonus' AND g.order_id=v_order_id))
            ORDER BY CASE WHEN g.source='bonus' THEN 0 ELSE 1 END, g.id
            FOR UPDATE
        LOOP
            EXIT WHEN needed <= 0;
            taken := LEAST(needed, grant_row.remaining);
            UPDATE wallet_grants SET remaining=remaining-taken WHERE id=grant_row.id;
            allocations := allocations || jsonb_build_array(jsonb_build_object('grant_id',grant_row.id,'amount',taken));
            needed := needed-taken;
        END LOOP;
        IF op='reserve_refund' AND needed <> 0 THEN RAISE EXCEPTION 'order gift changed during refund'; END IF;
        IF op='reserve_batch' THEN paid := needed; END IF;
        IF op IN ('reserve_refund','reserve_batch') THEN
            INSERT INTO wallet_holds(user_id,hold_key,amount,paid,allocations,state)
            VALUES (NEW.id,v_hold_key,-movement,paid,allocations,'held')
            ON CONFLICT (user_id,hold_key) DO UPDATE SET amount=EXCLUDED.amount,paid=EXCLUDED.paid,allocations=EXCLUDED.allocations,state='held'
            WHERE wallet_holds.state='released';
            IF NOT FOUND THEN RAISE EXCEPTION 'wallet hold already exists'; END IF;
        END IF;
    ELSIF movement > 0 THEN
        gift := movement;
        IF op='payment' THEN
            paid := (cmd->>'paid')::NUMERIC;
            IF paid < 0 OR paid > movement THEN RAISE EXCEPTION 'invalid paid credit'; END IF;
            gift := movement-paid;
        END IF;
        IF gift > 0 THEN
            INSERT INTO wallet_grants(user_id,order_id,source,original_amount,remaining)
            VALUES (NEW.id,v_order_id,CASE WHEN op='restore_unverified' THEN 'unverified' ELSE 'bonus' END,gift,gift);
        END IF;
    END IF;
    SELECT COALESCE(SUM(remaining) FILTER (WHERE source='bonus'),0),
           COALESCE(SUM(remaining) FILTER (WHERE source='unverified'),0)
      INTO NEW.bonus_balance, NEW.unverified_balance FROM wallet_grants WHERE user_id=NEW.id AND remaining>0;
    IF movement <> 0 OR NEW.frozen_balance <> OLD.frozen_balance OR op='verify_legacy' THEN
        INSERT INTO wallet_journal(user_id,operation,order_id,hold_key,delta,paid_balance,bonus_balance,unverified_balance,allocations,reference)
        VALUES (NEW.id,op,v_order_id,v_hold_key,movement,NEW.balance-NEW.bonus_balance-NEW.unverified_balance,NEW.bonus_balance,NEW.unverified_balance,allocations,cmd->>'reference');
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER wallet_account_balance BEFORE INSERT OR UPDATE OF balance, frozen_balance ON users
FOR EACH ROW EXECUTE FUNCTION wallet_account_balance();

CREATE FUNCTION wallet_initial_grant() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.bonus_balance > 0 THEN
        INSERT INTO wallet_grants(user_id,source,original_amount,remaining) VALUES (NEW.id,'bonus',NEW.bonus_balance,NEW.bonus_balance);
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER wallet_initial_grant AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION wallet_initial_grant();

CREATE FUNCTION wallet_credit_payment(uid BIGINT, amount NUMERIC, paid NUMERIC, oid BIGINT DEFAULT NULL, evidence TEXT DEFAULT NULL) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF amount < 0 OR paid < 0 OR paid > amount THEN RAISE EXCEPTION 'invalid wallet payment'; END IF;
    PERFORM wallet_prepare(uid,jsonb_build_object('operation','payment','paid',paid,'order_id',oid,'reference',evidence));
    UPDATE users SET balance=balance+amount, total_recharged=total_recharged+paid, updated_at=NOW() WHERE id=uid AND deleted_at IS NULL;
    IF NOT FOUND THEN RAISE EXCEPTION 'wallet user not found'; END IF;
END $$;

CREATE FUNCTION wallet_reserve_refund(uid BIGINT, oid BIGINT, amount NUMERIC) RETURNS NUMERIC LANGUAGE plpgsql AS $$
DECLARE u users%ROWTYPE; gift NUMERIC; held wallet_holds%ROWTYPE; key TEXT := 'refund:'||oid;
BEGIN
    SELECT * INTO STRICT u FROM users WHERE id=uid AND deleted_at IS NULL FOR UPDATE;
    SELECT * INTO held FROM wallet_holds WHERE user_id=uid AND hold_key=key;
    IF FOUND AND held.state='held' THEN
        IF held.paid <> amount THEN RAISE EXCEPTION 'refund hold amount mismatch'; END IF;
        RETURN held.amount;
    END IF;
    IF amount <= 0 OR amount > GREATEST(u.balance-u.bonus_balance-u.unverified_balance,0) THEN
        RAISE EXCEPTION 'insufficient refundable paid balance';
    END IF;
    -- Gifts frozen in an active batch cannot be reclaimed before settlement.
    IF EXISTS (
        SELECT 1 FROM wallet_holds h CROSS JOIN LATERAL jsonb_array_elements(h.allocations) a
        JOIN wallet_grants g ON g.id=(a->>'grant_id')::BIGINT
        WHERE h.user_id=uid AND h.state='held' AND h.hold_key LIKE 'batch:%' AND g.order_id=oid
    ) THEN RAISE EXCEPTION 'order gift is reserved by an active batch'; END IF;
    SELECT COALESCE(SUM(remaining),0) INTO gift FROM wallet_grants WHERE user_id=uid AND order_id=oid AND source='bonus';
    PERFORM wallet_prepare(uid,jsonb_build_object('operation','reserve_refund','paid',amount,'order_id',oid,'hold_key',key));
    UPDATE users SET balance=balance-amount-gift,updated_at=NOW() WHERE id=uid;
    RETURN amount+gift;
END $$;

CREATE FUNCTION wallet_release_refund(uid BIGINT, oid BIGINT, legacy_amount NUMERIC DEFAULT 0) RETURNS void LANGUAGE plpgsql AS $$
DECLARE held wallet_holds%ROWTYPE; key TEXT := 'refund:'||oid;
BEGIN
    PERFORM 1 FROM users WHERE id=uid FOR UPDATE;
    SELECT * INTO held FROM wallet_holds WHERE user_id=uid AND hold_key=key FOR UPDATE;
    IF NOT FOUND THEN
        -- A pre-migration pending refund has no source record. Preserve its
        -- spending value without manufacturing verified refundable funds.
        IF legacy_amount <= 0 THEN RAISE EXCEPTION 'wallet refund hold missing'; END IF;
        PERFORM wallet_prepare(uid,jsonb_build_object('operation','restore_unverified','order_id',oid));
        UPDATE users SET balance=balance+legacy_amount,updated_at=NOW() WHERE id=uid;
        RETURN;
    END IF;
    IF held.state='released' THEN RETURN; END IF;
    IF held.state<>'held' THEN RAISE EXCEPTION 'wallet refund already captured'; END IF;
    PERFORM wallet_prepare(uid,jsonb_build_object('operation','release_refund','order_id',oid,'hold_key',key));
    UPDATE users SET balance=balance+held.amount,updated_at=NOW() WHERE id=uid;
END $$;

CREATE FUNCTION wallet_capture_refund(uid BIGINT, oid BIGINT) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM users WHERE id=uid FOR UPDATE;
    UPDATE wallet_holds SET state='captured' WHERE user_id=uid AND hold_key='refund:'||oid AND state='held';
END $$;

-- Operator-only reconciliation after reviewing the original funding evidence.
-- There is intentionally no public HTTP endpoint for reclassifying balances.
CREATE FUNCTION wallet_verify_legacy(uid BIGINT, paid NUMERIC, bonus NUMERIC, evidence TEXT, oid BIGINT DEFAULT NULL) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF paid<0 OR bonus<0 OR paid+bonus<=0 OR length(trim(COALESCE(evidence,'')))=0 THEN RAISE EXCEPTION 'verification requires amounts and funding evidence'; END IF;
    IF oid IS NOT NULL AND NOT EXISTS (SELECT 1 FROM payment_orders WHERE id=oid AND user_id=uid) THEN RAISE EXCEPTION 'verification order owner mismatch'; END IF;
    PERFORM 1 FROM users WHERE id=uid AND deleted_at IS NULL FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION 'wallet user not found'; END IF;
    PERFORM wallet_prepare(uid,jsonb_build_object('operation','verify_legacy','paid',paid,'bonus',bonus,'reference',evidence,'order_id',oid));
    UPDATE users SET balance=balance,updated_at=NOW() WHERE id=uid;
END $$;
