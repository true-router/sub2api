# Balance Sources

## Accounting contract

Available `balance` remains the value used for API access checks. It equals signed paid balance plus promotional balance plus unverified balance. Paid balance can be negative under the existing post-billing overdraft policy. Only its positive, available part can fund refunds. Frozen batch funds are separate from available balance.

- `paid_balance`: verified payment principal, returned by user APIs.
- `bonus_balance`: promotional credit, including campaign gifts, signup gifts, referral balance transfers, ordinary vouchers and manual promotional balance additions.
- `unverified_balance`: pre-migration credit whose funding provenance has not been reviewed.

Consumption spends promotional grants in insertion order, then unverified credit, then paid credit. Consumption, adjustments and refund reservations serialize on the user row. Grants retain their originating payment order where applicable.

The user-default setting `default_balance` and per-source signup balance settings grant promotional credit in USD. They do not create paid credit. Zero disables the signup gift. Changes apply to future registrations only. The user repository reloads the inserted row and copies the trigger-maintained balance fields into the created user, so the first registration response reports the same sources as subsequent user reads.

Payment fulfillment records principal as `amount - bonus_amount` and promotional credit as `bonus_amount`. This also covers discounted payments, whose unpaid credit portion is already stored in `bonus_amount`. OPCPay credits verified redeemed payment value as paid credit; its transaction still claims each operation once.

Refund inputs use paid credit units (USD), not total credited units or gateway currency. The maximum is the original order's paid credit. Gateway money is calculated using the original paid credit as denominator and the original payment currency precision. Balance refunds always deduct paid credit. Administrative `force` and disabling deductions cannot make promotional or unverified credit refundable.

Every refund, including a partial refund, reclaims all remaining promotional credit originating from that order. Already consumed promotional credit and unrelated grants are not reclaimed. If that order's gifts are reserved by an active batch, the refund waits until batch settlement or release. Reclamation does not create promotional debt.

A refund reserves its principal and promotional reclamation before calling the provider. Pending refunds retain the reservation. A definitive failure restores the exact original sources. Success captures the reservation with the order status and audit log in one transaction. Replays cannot debit or restore the same reservation twice. Existing restrictions on repeated partial refunds and provider-specific support remain unchanged.

## Persistence boundary

Migration `253_wallet_balance_sources.sql` owns the PostgreSQL accounting functions, balance trigger, grant records, source reservations and journal. The trigger covers existing direct SQL and ORM writers so source accounting cannot be bypassed by an ordinary balance update. Ordinary positive adjustments are gifts; payment fulfillment must call `wallet_credit_payment` explicitly. Administrators select promotional or paid credit when adding funds. Promotional additions use the existing `add` operation. Verified manual payments use `add_paid`, require payment evidence in notes, and record that evidence in the journal. Only verified manual payments can trigger the optional admin-recharge affiliate rebate. The transaction-local command is consumed by the next matching user balance mutation. Batch commands are prepared in the existing billing transaction immediately before its balance mutation.

`wallet_grants` tracks spendable promotional and unverified credit. `wallet_holds` persists source allocations for batch and refund reservations. `wallet_journal` records balance mutations and manual source reconciliation. User-row locking precedes grant and reservation locks.

The admin balance history combines redeem-code records, affiliate transfers and completed OPCPay consumption records. OPCPay entries use the `opcpay_balance` type and the business ID as their code. Their numeric redeem-code ID is zero because they are read-only history projections; clients identify history rows by type and code together. `used_at` is the atomic delivery timestamp, while `created_at` remains the operation creation time. The cumulative recharge summary adds completed OPCPay credits to the existing positive balance/admin-balance redeem-code sum, regardless of pagination or type filter. Purchase entries and undelivered consumption records are excluded. Existing completed operations appear without backfilling records or changing balances. Run `backend/tools/opcpay-e2e/run.sh` to verify this behavior through real HTTP handlers and disposable PostgreSQL; the receipt at `output/opcpay-e2e/checks.json` includes response snapshots and source hashes.

Do not update source columns or grant records directly. Do not remove this accounting trigger while application balance writers remain active.

## Convert paid credit to promotional credit

In **Admin > Users**, choose **Convert to promotional balance** in the balance column. Enter the amount and a required reason. The dialog previews both resulting balances and explains that converted credit cannot be refunded. Confirm to apply the conversion.

The existing admin balance endpoint accepts `operation: "convert_paid_to_bonus"`, `balance` in USD and a nonempty `notes` reason. The UI retains an `Idempotency-Key` for retries of an unchanged submission. Changed submissions receive a new key. The existing endpoint coordinator handles replay and payload conflicts.

Migration `254_wallet_paid_to_bonus.sql` adds the atomic conversion function. It locks the user row, checks available verified paid credit, creates an independent promotional grant, refreshes source summaries through the existing accounting trigger, and writes a `convert_paid_to_bonus` journal record with amount and reason. The conversion leaves total balance, frozen balance, cumulative recharge, existing order amounts and existing order-linked gifts unchanged. It cannot use promotional, unverified or frozen credit. It does not create an affiliate rebate or call a payment provider.

Converted credit is spent under the normal promotional-first rule. It cannot fund a refund. An original order refund can still reclaim that order's remaining original promotional gift; the independent conversion grant is not that original gift.

## Historical balances

The migration preserves historical available balances as unverified credit. It reconstructs existing batch reservations from job records and billing idempotency claims, and treats their sources as unverified. It fails transactionally if historical frozen totals do not match outstanding batch claims. Reconcile that mismatch before deployment; do not skip the check.

A pre-migration pending refund has no grant reservation. If it fails, its returned credit remains unverified. Existing pending refund audit records preserve their original total-credit-to-cash conversion convention. New records identify source-aware accounting with `walletAccounting`.

Review original payment, grant, consumption and refund evidence before converting unverified credit. A cumulative recharge total or a current balance alone is insufficient evidence. After review, use the operator-only database function, with an audit reference and the originating order where applicable:

```sql
BEGIN;
-- Verify 30 USD as principal and 10 USD as remaining promotional credit.
-- Replace the example identifiers and evidence reference with reviewed values.
SELECT wallet_verify_legacy(123, 30, 10, 'review-ticket-456', 789);
COMMIT;
```

The function preserves total spending value, rejects reclassification beyond remaining unverified credit, validates the order owner, and records the evidence reference in `wallet_journal`. Supply a null order only for evidence unrelated to a payment order. There is no public HTTP endpoint for source reclassification.

## Acceptance

Run from the repository root:

```sh
backend/tools/wallet-e2e/run.sh
```

The script creates a disposable PostgreSQL 18 container, builds the Ent schema, applies the accounting migration and runs real fulfillment/refund handlers, repositories and provider adapters against local HTTP payment fixtures. It exercises migration of historical frozen credit, gift-first billing, order-specific reclamation, failed refunds, pending refund recovery through a fresh service instance, replay handling, concurrent consumption/refunds, vouchers and explicit historical reconciliation. It removes the database container on exit.

The receipt is written to `.planning/2026-11-11-balance-sources/artifacts/wallet-e2e.json`. It contains scenario balances and SHA-256 hashes of the accounting sources. It does not contain payment credentials or real user data. The fixture uses synthetic identity and local payment responses. It does not establish live-provider, production-migration or browser acceptance.

Signup acceptance calls the real registration HTTP handler with a default gift of 5 USD and with gifts disabled. It asserts the first response reports zero paid credit, the expected promotional credit, and no unverified credit. It checks subsequent persisted balances and consumes part of the signup gift through the billing repository. No external identity provider or refresh-token cache is used in this fixture.
