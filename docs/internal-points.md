# Internal points

The internal accounting unit is **points** (Chinese: **积分**). Its compact symbol is **P**. It is not an ISO currency and must not use the dollar symbol.

Balances, promotional gifts, credited recharge amounts, customer charges, API key budgets, subscription allowances, group pricing, and user platform limits use points. Numeric values and billing arithmetic are unchanged. Existing field names such as `credit_usd` and `daily_limit_usd` remain for compatibility. The display-only `unit` metadata in `/v1/usage` is `P`. CC Switch balance queries preserve this unit. Balance alert emails use P.

Payment amounts retain their real payment currency. Subscription selling prices retain their configured payment currency. Actual upstream account costs and official provider reference prices remain in USD when specified as USD. A report that contains both customer charges and upstream costs must label the units separately.

A model price in points is the base charge multiplied by the applicable group multiplier. Its CNY reference cost is the points charge multiplied by the configured CNY recharge cost per point. Do not apply the group multiplier twice. Do not assume one point equals one actual USD. This change does not alter recharge conversion or add a new exchange rate.

## Verification

Run from `frontend`:

```sh
pnpm run build
POINTS_E2E_OUTPUT=/tmp/sub2api-points-e2e node e2e/points-unit.mjs
```

The browser E2E opens the actual Vue application against deterministic local API fixtures. It checks balances, discounted and base model prices, usage charges, CNY recharge payments versus credited points, official currency references, and English labels. It saves a JSON receipt, API requests, build-server logs, and source hashes. It does not verify production persistence or a real payment-provider transaction.

Run the real backend acceptance from `backend`:

```sh
go run ./tools/production-billing-e2e --unified-access --output /tmp/sub2api-points-backend-e2e/summary.json
```

This uses disposable PostgreSQL and Redis, the real application and HTTP handlers, and a local mock model provider. It also verifies points unit metadata against the persisted balance. It removes fixture services after acceptance.

## Account model price comparison

`/account/model-plaza` shows official catalog USD references next to site CNY costs, equivalent USD costs, points charges, and savings for each comparable input/output/cache price. Official references come from the ungrouped model catalog for the selected billing model. A custom group base price is not an official reference.

Site points = base charge × negotiated effective rate (including the applicable current time factor). Site CNY cost = points × the configured standard CNY cost per point. Site USD equivalent = CNY cost / USD-to-CNY reference rate. Savings = 1 − site CNY cost / official CNY equivalent. Calculations use unrounded values. Bonuses, fees and subscriptions can change personal costs.

The existing `/user/api-access/models` array remains compatible and each offer gains an optional `official_pricing`. JWT-protected `/user/api-access/pricing-context` returns the existing plaza recharge-reference setting and cached reference FX, without exposing group configuration or depending on the public showcase switch. Missing conversion context leaves points visible. Missing or expired FX suppresses USD equivalents and savings. Missing official references never fall back to custom group prices. Per-request and per-token units are not compared. Context bands reuse the existing price alignment logic.

Run from `frontend`:

```sh
node e2e/model-price-comparison.mjs
```

The browser receipt is saved to `/tmp/sub2api-account-comparison-e2e/summary.json`. It covers the real Vue route with deterministic local catalog/FX fixtures, including independent output savings, recharge-cost changes, missing data, expired/cached FX, zero charges, request/token unit mismatch, tier alignment, duplicate endpoint offers, mobile overflow and dark mode. The real backend unified-access acceptance also checks authenticated pricing context and independent catalog references.
