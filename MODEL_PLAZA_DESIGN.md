# Model Plaza Price Comparison

## Contract

- Keep Vue and reuse the existing BoardUI Vue components and semantic tokens.
- Group the catalog by platform and model, with group offers inside each model.
- Default to CNY. Convert both sides when switching to USD.
- Keep balance credits distinct from fiat currency.
- Configure CNY per balance credit independently from the public USD/CNY reference rate.
- Require a positive reference rate and its date before showing savings.
- Calculate savings from unrounded comparable prices: `1 - credits * CNY_per_credit / (official_USD * USD_CNY)`.
- Calculate savings separately for input, output, and cache prices. Do not claim an aggregate workload discount.
- Use the standard recharge basis. Disclose that bonuses, fees, subscriptions, and historical purchases can change personal costs.
- Preserve tiered pricing, media billing, user rates, time windows, and peak-rate disclosures.
- Do not change billing, recharge, model discovery, or access-control behavior.

## Delivery

1. Add persisted comparison settings and expose them through the model-plaza API.
2. Add an administrator control for the recharge basis. Retrieve the reference rate and its date from Frankfurter v2, with caching and bounded stale fallback.
3. Replace the group-first page with BoardUI model cards and comparable price rows.
4. Verify type checking, production compilation, and browser flows with deterministic API fixtures.

## End-to-end acceptance

Use the real application route with fixture API responses. Save the runnable scenario and a JSON receipt locally.

- CNY: 6.4 credits at CNY 1/credit versus USD 8 at CNY 7/USD displays CNY 6.40 versus CNY 56.00 and 88.6% savings.
- USD: display approximately USD 0.91 versus USD 8.00, retaining 88.6% savings.
- Check output and cache prices independently; do not derive them from input savings.
- Apply a user rate in place of the group rate, without applying either twice.
- Hide comparisons when the rate/date or official price is missing, or units cannot be compared.
- Do not show negative savings as a discount, or round a nonzero charge to free.
- Exercise search, platform/group filters, model expansion, tiers, time pricing, mobile width, and dark mode.
- Distinguish fixture-backed UI acceptance from live backend and production acceptance.

## Status

Implemented on `main`, based on commit `6ed4594a28ee09cd16f2b01173f91a92bb2770c5`.

- Backend handler/service compilation passed.
- Frontend type checking and production compilation passed.
- All 22 fixture-backed browser assertions passed, including card layout and chart interactions.
- The real Frankfurter request and subsequent cached read passed.
- Verification commands and coverage boundaries are in `tools/model-plaza/README.md`.
- No production deployment was performed.


## Rectangular card and chart revision

- Use a two-column model-card grid on wide screens and one column on phones.
- Keep each model in one rectangular card. Select its group offer inside the card.
- Add a horizontal cost chart using the existing Chart.js/Vue integration and BoardUI theme tokens.
- Switch input/output comparisons independently. Use a zero baseline and the same currency and unit for both bars.
- Highlight the selected metric's savings percentage and absolute saving. Do not claim one workload-wide discount.
- Preserve the complete pricing table and expandable cache/tier/time details.
- Verify grid geometry, group switching, metric switching, graph values, currency invariance, fallback states, and phone overflow with the real built route.
- Save desktop and phone previews with the repeatable browser receipt. Keep the demo server available to the user.
