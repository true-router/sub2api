# sub2api 0.2.25

## Changes

- Show OPCPay purchase-time monetary amounts as paid amounts, using the agreed channel accounting convention. Keep credited USD separate from the payment currency and amount.
- Remove the catalog-price label from admin and user order views.
- Show a visible empty state for unavailable revenue and average amounts. Preserve legitimate zero amounts.
- Document historical payment reconciliation and verify that corrected payment amounts restore revenue and averages without crediting the wallet again.

## Upgrade

This version adds no database migration and does not change wallet balances. Existing OPCPay purchases continue to use their saved purchase-time price and currency. Historical payment corrections require an original bill or owner confirmation and an audit record.

## Verification

129 real PostgreSQL/HTTP checks and 8 Chinese/English Vue statistics rendering checks passed. Frontend type checking, scoped ESLint and the frontend build passed. Rendering checks do not verify browser layout or interaction. Real merchant payments and production deployment were not performed for this release.
