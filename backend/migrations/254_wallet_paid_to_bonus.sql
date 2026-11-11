-- Reclassify available paid funds without changing total or frozen balances.
-- The existing accounting trigger refreshes source summaries; this function
-- records the zero-total source transfer explicitly in the journal.
CREATE FUNCTION wallet_convert_paid_to_bonus(uid BIGINT, amount NUMERIC, reason TEXT)
RETURNS TABLE(old_balance NUMERIC, new_balance NUMERIC) LANGUAGE plpgsql AS $$
DECLARE
    wallet_user users%ROWTYPE;
    grant_id BIGINT;
BEGIN
    IF amount IS NULL OR amount::text IN ('NaN','Infinity','-Infinity')
       OR amount <= 0 OR amount <> ROUND(amount,8)
       OR length(btrim(COALESCE(reason,'')))=0 THEN
        RAISE EXCEPTION 'conversion requires a positive amount with at most eight decimals and a reason';
    END IF;
    SELECT * INTO STRICT wallet_user FROM users WHERE id=uid AND deleted_at IS NULL FOR UPDATE;
    IF amount > GREATEST(wallet_user.balance-wallet_user.bonus_balance-wallet_user.unverified_balance,0) THEN
        RAISE EXCEPTION 'insufficient available paid balance';
    END IF;
    INSERT INTO wallet_grants(user_id,source,original_amount,remaining)
    VALUES (uid,'bonus',amount,amount) RETURNING id INTO grant_id;
    PERFORM wallet_prepare(uid,jsonb_build_object('operation','convert_paid_to_bonus','reference',btrim(reason)));
    UPDATE users SET balance=balance,updated_at=NOW() WHERE id=uid;
    INSERT INTO wallet_journal(user_id,operation,delta,paid_balance,bonus_balance,unverified_balance,allocations,reference)
    SELECT id,'convert_paid_to_bonus',0,balance-bonus_balance-unverified_balance,bonus_balance,unverified_balance,
           jsonb_build_array(jsonb_build_object('grant_id',grant_id,'amount',amount)),btrim(reason)
    FROM users WHERE id=uid;
    RETURN QUERY SELECT wallet_user.balance,wallet_user.balance;
END $$;
