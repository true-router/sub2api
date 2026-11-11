# Model Plaza verification

Run from the repository root. These scenarios use the production frontend with deterministic API fixtures. They do not use production credentials or mutate billing data.

## Build

```sh
cd frontend
pnpm exec vue-tsc --noEmit
pnpm exec vite build
cd ..
```

## Browser acceptance

Start the fixture server in a separate terminal:

```sh
node tools/model-plaza/fixture-server.mjs
```

Run the browser scenario with Ego Lite:

```sh
ego-browser nodejs < tools/model-plaza/browser-e2e.mjs
```

The scenario creates `/tmp/sub2api-model-plaza-e2e/receipt.json`. It contains every assertion and the observed page text. The fixed USD/CNY quote is 7 so the example always gives 88.6% savings.

The 22 checks cover CNY and USD prices, stable savings across currencies, output conversion, cache/peak disclosures, search, user rate replacement, absent FX, absent official prices, premium prices, subscriptions, tiers/time periods, tiny positive prices, cached quotes, mobile page overflow, and dark surfaces. They also verify the chart values, independent output comparison, hidden charts without FX, two-column card layout, group-offer selection, and phone chart sizing.

Full-page visual artifacts are saved next to the receipt: `cards-desktop.png`, `cards-mobile.png`, and `cards-mobile-dark.png`.

Stop the fixture server after verification.

## Live reference rate

```sh
tools/model-plaza/check-public-rate.sh
```

This invokes the production Go exchange-rate service against `https://api.frankfurter.dev/v2/rate/USD/CNY`, verifies a second read uses its cache, and saves `/tmp/sub2api-model-plaza-e2e/public-rate.json`. Its temporary Go entry point is removed on exit.

## Scope

- The public rate probe exercises the real external API and cache.
- Browser acceptance exercises the real compiled Vue route with fixture API data.
- Backend compilation verifies the new handler/settings integration compiles.
- Administrator settings persistence, authenticated live catalog data, platform/group filter clicks, and production deployment are not covered by the fixture browser receipt.
- No unit tests are added by this change.
