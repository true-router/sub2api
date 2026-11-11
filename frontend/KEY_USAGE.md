# Public key usage

`/key-usage` uses the same public header, semantic surfaces, accent buttons and rounded panels as the BoardUI home. It composes the existing Vue ports of Input, Button, Badge, Skeleton, StatCard, SegmentedControl and Table. Chart.js and vue-chartjs are already installed; KeyUsageCharts reuses the shared useBoardUIChartTheme hook. Existing quota SVGs use semantic chart tokens. It does not use paid chart source or add dependencies.

## Query contract

The browser sends the entered key as the Bearer header on a direct `/v1/usage` request. Dates, timezone and the daily-usage period retain their existing request parameters. The key remains in component memory and is not written to browser storage or the URL. The privacy text states that the key is sent to this site for querying.

The default result presents quota rings, three headline figures, and three charts:

- A chronological daily trend switches between cost, requests and input/output tokens. Its 7/30/90-day selector controls the existing daily-record window. It refreshes only daily_usage using the last submitted key and date snapshot. Other charts, headline cards, metric selections and expanded details remain mounted. Loading and retryable errors stay inside the trend card. A new full query or unmount aborts and invalidates pending daily requests.
- A model doughnut switches between cost and requests. Its scope is the selected query date range. Duplicate model names are merged; more than six models become the top five and an Other models slice. The total is preserved.
- Horizontal bars compare input, output, cache creation and cache read counts. The user can switch between today and lifetime. These are category counts, not mutually exclusive shares or a cache-savings estimate.

Daily and model cost use actual_cost when present, including zero; otherwise they use cost. Charts do not invent missing days, comparisons or forecasts. A zero model total displays an empty state instead of a synthetic slice. Zero token data has an explicit empty state. Canvas labels and the expandable details provide access to the underlying data.

The native View usage details disclosure starts closed and renders its content only after opening. It retains all 16 statistics, quota details and both original tables. The page retains wallet balance, subscription limits, key quota status, expiry, reset countdown, and existing cost formatting. Empty keys and incomplete or reversed custom dates show inline errors. Request errors clear old results. While querying, controls are disabled and shared skeletons display loading state.

The page and charts disable animation. Dark mode follows semantic tokens and persists through the existing theme setting. Wide tables scroll within their panels and remain keyboard accessible.

## Reproduce acceptance

From `frontend`:

```sh
pnpm exec vue-tsc --noEmit
pnpm exec eslint src/views/KeyUsageView.vue src/components/key-usage/KeyUsageCharts.vue src/i18n/locales/zh/landing.ts src/i18n/locales/en/landing.ts
pnpm exec vite build --outDir /tmp/truerouter-key-usage-build
node e2e/key-usage.mjs
```

The E2E runner requires `ego-browser`. It starts an isolated Vite server on `127.0.0.1:13038` and a fixture on `127.0.0.1:18088`. It does not call production. Shared fixture responses are in `e2e/fixtures/key-usage.mjs`. Only `sk-preview-*` synthetic keys are recognized by this fixture. The runner checks frontend behavior and verifies that custom dates, daily periods and timezone reach the fixture API.

Artifacts are saved in `output/key-usage-boardui/`: `checks.json`, screenshots, redacted request scenarios/parameters, and the Vite log. Real keys are never written to these artifacts. Browser checks cover initial/error/loading states, reveal, wallet/subscription/quota data, plotted values and chronological ordering, chart metric switches, collapsed details, zero actual cost, grouped model totals, cost fallback, date controls, zero/missing/empty data, exhausted and expired statuses, no key persistence, locale, dark mode, return navigation and 390px/320px layouts.

## Local public preview

The existing Cloudflare tunnel points to a static local build. Its preview server can use `serveKeyUsageFixture` for `/v1/usage`. Use `sk-preview-wallet`, `sk-preview-subscription`, or `sk-preview-quota` to inspect results. These are simulated values, not live balances. The production application continues to use its own `/v1/usage` endpoint.
