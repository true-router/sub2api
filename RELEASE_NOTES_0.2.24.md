# sub2api 0.2.24

## Changes

- Record completed OPCPay balance deliveries in order management and user order history.
- Create the order and wallet credit atomically. Retried and concurrent recovery cannot duplicate either record.
- Preserve the purchase catalog price and currency separately from credited USD. Display missing historical payment amounts as unknown and exclude them from monetary aggregates.
- Add the OPCPay order filter and keep unsupported refunds disabled.

## Upgrade

Migration `255_opcpay_payment_orders.sql` adds purchase quote columns and backfills completed historical deliveries. It does not change wallet balances, wallet journal entries or original operation timestamps.

During the first blue/green upgrade, reconcile completed deliveries missing an order after the previous instance has drained. See `docs/PAYMENT.md`, section `OPCPay Orders`. Never replay wallet credits. Preserve the additive migration and orders if rolling back the application.

## Verification

123 real PostgreSQL/HTTP end-to-end checks passed with a simulated merchant. Backend build and static analysis, frontend type checking and build, and scoped ESLint passed. Real merchant payment acceptance was not performed.
