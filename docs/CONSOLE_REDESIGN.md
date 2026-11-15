# User Console Rewrite

## Scope

The six primary user content pages follow the approved immediate-access prototype in `.planning/2026-11-14-console-redesign/prototype.html`. The existing sidebar component and mobile behavior are reused. Ordinary users have six primary entries: Home, API keys, Models, Usage, Billing, and Account. Their sidebar has no search row or workspace subtitle. Administrator navigation is unchanged. Legacy billing links select Billing; optional feature links select Account. This change does not deploy the console.

| Page | Result |
| --- | --- |
| Home | Actual gift balance, default API key, Base URL, compact wallet and recent activity. No chart wall or automatic user onboarding tour. Funding and active-plan notices use account data. |
| API keys | Credential cards, default/status labels, direct optional key creation, copy, rename, enable/disable, and confirmed deletion. Limits and IP restrictions are optional edit details. Existing key metadata stays unchanged. |
| Models | Searchable live catalogue, shared-key explanation, protocol paths, model ID copy, and expandable actual wallet prices. No client selection or CCS import. |
| Usage | Date/model query, three summary values, recent records with actual billing sources, separate empty/error states, and lazy detailed analysis/errors/export. |
| Billing | Wallet source summary and four tabs: funding, plans, redemption, and orders. Existing provider checkout, discounts, fees, refunds, payment recovery, and fulfillment remain connected. |
| Account | Editable profile, focused password/two-factor/passkey/binding sections, and optional avatar, notifications, referrals, MCP, status, and custom-page links. Existing security components and APIs are reused. |

Existing `/subscriptions`, `/redeem`, and `/orders` links remain usable. Billing also embeds these functions without duplicating the layout or sidebar.

## Implementation boundaries

`ConsolePage.vue` composes the existing application layout with one page heading and compact content width. `AppLayout` and `AdminContentHeading` retain their default behavior for other routes. Administrator onboarding remains unchanged. User onboarding no longer automatically interrupts API access.

`BillingCheckout.vue` extracts the existing payment logic from the old purchase page. The checkout stays mounted while users inspect billing tabs. A pending payment locks tab changes and redirects restored order state to the visible checkout panel. Reloading resumes the same order. Returning from the provider UI cannot clear the operation unless the server reports a terminal order status. Unknown status keeps the original operation. Confirmed cancellation restores navigation. Checkout transport failures show a retry action instead of an empty or disabled service state.

The amount cards use the existing quote functions. Each card shows the payment currency and credited USD funds. The final quote includes the existing fees and promotional breakdown. No exchange rate or bonus value is hardcoded into the production UI.

`UsageExplorer.vue` retains the existing detailed analysis, filters, errors, and export as an optional lazy-loaded panel. It does not load on the initial simplified usage page.

The rewrite uses installed Vue BoardUI components and semantic tokens. It adds no dependency and no unit test. The unified-key billing implementation is documented in [UNIFIED_API_ACCESS.md](UNIFIED_API_ACCESS.md).

## Reproducible acceptance

Build the production frontend and start the disposable real-process fixture from the repository root:

```sh
pnpm --dir frontend build
./backend/tools/production-billing-e2e/run.sh \
  --unified-access \
  --browser-ready /tmp/sub2api-console-browser-ready.json \
  --output /tmp/sub2api-console-server-final.json
```

The fixture writes temporary login and redemption data with mode 0600. It waits up to 15 minutes for `/tmp/sub2api-console-browser-ready.json.done`. It runs the actual server with embedded frontend, PostgreSQL, Redis, synthetic model providers, and a synthetic payment provider. No production service or real payment is used.

Run the browser scripts sequentially in another terminal. Keep the same browser TaskSpace for the whole acceptance:

```sh
ego-browser nodejs < frontend/e2e/console-redesign.browser.mjs
ego-browser nodejs < frontend/e2e/console-redesign-visual.browser.mjs
ego-browser nodejs < frontend/e2e/console-redesign-payment.browser.mjs
ego-browser nodejs < frontend/e2e/console-redesign-recovery.browser.mjs
```

The interaction script exercises real data persistence. The visual script records each primary page at desktop and 390px mobile widths in light and dark themes. It waits for theme transitions before saving screenshots. The payment script checks pending-order isolation, recovery, and confirmed cancellation. The recovery script checks checkout retries, settled gift exhaustion, funding guidance, and durable default-key deletion. Rapid admissions can be denied while previous requests retain in-flight reservations. The script waits for settlement and retries only explicit uncharged admission denials; it never replays a successful request. Inspect the masked screenshots before the recovery script releases the fixture through the `.done` file. Then finish the same TaskSpace once.

The browser receipt and screenshots are written under `/tmp/sub2api-console-browser/`. Server receipts record individual assertions, migration verification, binary identity, and an unchanged source manifest. The browser receipt must be correlated with the matching server receipt and compiled frontend identity when archived.

Run the existing payment/wallet/refund regression independently:

```sh
./backend/tools/production-billing-e2e/run.sh --output /tmp/sub2api-console-legacy-final.json
```

## Coverage limits

Synthetic external providers verify integration behavior, not live payment-provider, OAuth, WebAuthn device, or upstream-model availability. Security UI checks do not enroll real two-factor credentials or passkeys. Production deployment and conversion-rate measurement remain separate operations.

## Verified local result (2026-11-15)

- Browser: 41 checks passed.
- Unified server: 55 checks passed.
- Existing payment/wallet/refund regression: 55 checks passed.
- Both server runs verified 309 migrations against source checksums. Source manifests stayed unchanged during acceptance. The browser fixture includes compiled frontend asset hashes.
- Recorded 24 masked screenshots covering all six pages at desktop/mobile widths and light/dark themes. Inspected layout, contrast, credentials, amounts, and mobile overflow.
- Production build, typecheck, scoped ESLint, and whitespace checks passed. No new unit tests were added.
- Receipts, replay scripts, screenshots, earlier admission-probe evidence, and SHA256 hashes are archived outside tracked source at `.local/console-redesign/2026-11-15/`.
- Temporary services and private fixture data were removed. Browser TaskSpace 76 was finished. No commit, push, release, or deployment was performed.

## Sidebar simplification acceptance

The ordinary-user sidebar was reduced to six entries. Subscription, orders, and redemption remain inside Billing. Status, available channels, permitted batch images, referrals, MCP, and custom pages remain available in Account optional features. Billing balance, subscriptions, and redemption remain reachable when online payments are disabled; checkout and order tabs are not mounted in that state.

Real local-server browser acceptance passed 11 checks, including desktop/mobile navigation, legacy-link highlighting, optional features, and disabled-payment behavior. The original local payment setting was restored after acceptance. Receipt and screenshots: `.local/sidebar-simplification/`. No production setting changed.
