-- Preserve the purchase quote. Historical prices remain unknown.
ALTER TABLE opcpay_operations ADD COLUMN IF NOT EXISTS purchase_price NUMERIC(20,2);
ALTER TABLE opcpay_operations ADD COLUMN IF NOT EXISTS purchase_currency VARCHAR(3) NOT NULL DEFAULT '';

-- Called inside the wallet delivery transaction, and by the historical backfill.
-- This function never changes wallet balances or operation timestamps.
CREATE OR REPLACE FUNCTION opcpay_record_order(bid TEXT) RETURNS BIGINT LANGUAGE plpgsql AS $$
DECLARE
    operation opcpay_operations%ROWTYPE;
    oid BIGINT;
BEGIN
    SELECT * INTO STRICT operation FROM opcpay_operations WHERE business_id=bid FOR UPDATE;
    IF operation.kind <> 'consume' OR operation.status <> 'COMPLETED' THEN
        RAISE EXCEPTION 'OPCPay order requires a delivered consumption';
    END IF;
    INSERT INTO payment_orders (
        user_id,user_email,user_name,user_notes,amount,pay_amount,fee_rate,bonus_amount,
        recharge_code,out_trade_no,payment_type,payment_trade_no,order_type,
        provider_instance_id,provider_key,provider_snapshot,status,refund_amount,force_refund,
        expires_at,paid_at,completed_at,client_ip,src_host,created_at,updated_at
    )
    SELECT operation.user_id,u.email,u.username,u.notes,operation.credit_usd,
        COALESCE(operation.purchase_price,0),0,0,'',operation.business_id,'opcpay','', 'balance',
        operation.provider_instance_id::text,'opcpay',
        jsonb_build_object('schema_version',1,'provider_key','opcpay',
            'provider_instance_id',operation.provider_instance_id::text,
            'currency',operation.purchase_currency,
            'pay_amount_known',operation.purchase_price IS NOT NULL AND operation.purchase_currency <> '',
            'payment_amount_source',CASE WHEN operation.purchase_price IS NULL THEN 'unknown' ELSE 'catalog' END,
            'sku_id',operation.sku_id),
        'COMPLETED',0,FALSE,operation.updated_at,operation.updated_at,operation.updated_at,
        '','',operation.created_at,operation.updated_at
    FROM users u WHERE u.id=operation.user_id
    ON CONFLICT (out_trade_no) WHERE out_trade_no <> '' DO NOTHING;

    -- Never reuse an unrelated order if an external ID collides.
    SELECT id INTO STRICT oid FROM payment_orders
    WHERE out_trade_no=operation.business_id AND user_id=operation.user_id
      AND payment_type='opcpay' AND provider_key='opcpay' AND status='COMPLETED'
      AND amount=operation.credit_usd;
    RETURN oid;
END $$;

-- Each operation has one stable external ID. Replaying cannot duplicate orders.
-- No provider calls, wallet writes, or invented historical prices.
SELECT opcpay_record_order(business_id)
FROM opcpay_operations WHERE kind='consume' AND status='COMPLETED'
ORDER BY business_id;
