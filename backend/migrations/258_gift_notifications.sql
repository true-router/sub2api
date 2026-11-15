-- Personal gift acknowledgements share the wallet credit transaction.
-- Existing historical balances are deliberately not replayed.
CREATE TABLE gift_notifications (
 id bigserial PRIMARY KEY,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 event_key text NOT NULL UNIQUE,
 source text NOT NULL CHECK (source IN ('initial','credit','payment_bonus')),
 amount numeric(20,8) NOT NULL CHECK (amount>0),
 read_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX gift_notifications_user ON gift_notifications(user_id,id DESC);

CREATE FUNCTION notify_initial_gift() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.bonus_balance>0 THEN
  INSERT INTO gift_notifications(user_id,event_key,source,amount)
  VALUES(NEW.id,'initial:'||NEW.id,'initial',NEW.bonus_balance);
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER notify_initial_gift AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION notify_initial_gift();

CREATE FUNCTION notify_wallet_gift() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE gift numeric(20,8):=0; kind text;
BEGIN
 IF NEW.operation='adjust' AND NEW.delta>0 THEN
  gift:=NEW.delta; kind:='credit';
 ELSIF NEW.operation='payment' AND NEW.delta>0 AND NEW.order_id IS NOT NULL THEN
  SELECT COALESCE(SUM(original_amount),0) INTO gift FROM wallet_grants
  WHERE user_id=NEW.user_id AND order_id=NEW.order_id AND source='bonus';
  kind:='payment_bonus';
 END IF;
 IF gift>0 THEN
  INSERT INTO gift_notifications(user_id,event_key,source,amount)
  VALUES(NEW.user_id,'journal:'||NEW.id,kind,gift);
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER notify_wallet_gift AFTER INSERT ON wallet_journal FOR EACH ROW EXECUTE FUNCTION notify_wallet_gift();
