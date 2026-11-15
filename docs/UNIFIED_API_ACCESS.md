# Unified API Access

## User flow

Registration provisions one default API key. The dashboard shows the actual promotional balance, a masked default API key, and the API origin. Copy the key and Base URL into an API integration. Use the same credential for supported Claude and GPT/Codex models. Change the request model ID to switch models.

There is no client selection, CCS import, mandatory setup wizard, or mandatory test call. The keys page can create additional unified keys directly. Legacy key editing remains available. The account model catalogue lists permitted models and wallet prices. Usage records show the actual subscription, promotional, purchased, and unverified balance portions. The purchase page discloses automatic wallet continuation.

## Identity and provisioning

Migration 256 adds explicit `access_mode` and `is_default` metadata. Existing keys default to `legacy`. Provisioning locks the user row and a unique partial index guarantees one default identity per user. The index includes deleted rows. Deleting a default key creates a durable tombstone. Reconciliation never recreates that key. Users can create additional keys after deletion.

All registration bootstrap paths attempt provisioning. Provisioning errors are logged without invalidating a completed registration. The authenticated dashboard reconciliation endpoint retries provisioning. Existing users receive a default key through this endpoint. A default can exceed the legacy active-key count by one; additional keys obey creation limits.

The managed Composite group is an identity container. Each request gets its own key copy with a permitted existing routing group. The cached key is never mutated. Group rebinding through the legacy key update endpoint is rejected for unified keys.

## Admission and billing

1. Find an active subscription that supports the model and protocol. Choose earliest expiry, then lowest subscription ID.
2. Check current subscription windows. Exact exhaustion prevents a new admission.
3. If no subscription qualifies, choose a permitted wallet group. Explicit user group discounts take priority, followed by stable group ID order.
4. Pin the source and routing group for the admitted request. Preserve the existing scheduler, provider selection, model restrictions, group prices, and user discount rules.
5. Settle the whole request to the pinned source. An admitted request may exceed a subscription limit. Subsequent admissions use another eligible subscription or wallet.
6. Consume promotional funds before purchased funds through the existing wallet ledger. Preserve historical unverified-balance semantics.

Provider rejection does not cause a paid wallet retry. Unified keys share the account subscription and wallet budgets. Per-key restrictions still apply. Legacy keys retain their previous group, limits, and billing behavior.

Native Responses WebSocket requests resolve admission for each turn. A connection is bound to an upstream account and routing group. If the next turn requires another group or subscription source, the server closes with code 1013 before forwarding that turn. Reconnect with the same credential. No new key or client configuration is required.

Migration 257 adds a billing-source receipt inside the existing idempotent settlement transaction. Wallet portions are computed with PostgreSQL numeric arithmetic. Receipts correlate by request ID and API key ID and are exposed only through the owning user's usage API.

## Model catalogue and prices

The authenticated catalogue derives names from permitted groups, configured account mappings, Composite exact aliases, and permitted channels. Private channel names are excluded. Model discovery does not activate subscriptions or modify billing windows. Discovery remains available to a subscriber with no wallet funds.

Wallet offers reuse the billing pricing resolver, negotiated rates, group peak multiplier, channel time multiplier, and existing long-context schedule. Composite route aliases use their configured concrete upstream model for displayed settlement prices, matching the existing route body rewrite. Different supported protocols have separate offers.

Operators must configure real accounts, models, routing groups, prices, subscription products, and signup gifts. Provisioning a credential does not invent provider capacity or grant private groups. Unknown models fail explicitly.

## End-to-end acceptance

Run from the repository root with Docker, Go, and the frontend dependencies available:

```sh
./backend/tools/production-billing-e2e/run.sh --unified-access --output /tmp/sub2api-unified-final.json
./backend/tools/production-billing-e2e/run.sh --output /tmp/sub2api-legacy-billing-final.json
```

Both commands build and run the actual server against disposable PostgreSQL and Redis. Synthetic external providers implement HTTP and native Responses WebSocket traffic. Payment verification uses the existing synthetic payment adapter. No production system or real payment is used. Containers, credentials, and temporary databases are removed.

Receipts contain individual assertions, binary identity, migration verification, and a source SHA256 manifest. Acceptance fails if relevant source files change during the run.

For browser acceptance, first build the frontend. Then run:

```sh
cd frontend && pnpm build
cd ..
./backend/tools/production-billing-e2e/run.sh --unified-access --browser-ready /tmp/sub2api-unified-browser-ready.json --output /tmp/sub2api-unified-final.json
```

The browser fixture embeds the compiled frontend and writes temporary login data with mode 0600. It waits up to fifteen minutes for `/tmp/sub2api-unified-browser-ready.json.done`. Use the fixture to verify the dashboard, copy actions, direct additional-key creation, model prices, usage sources, and mobile layout. Do not include credentials or revealed API keys in artifacts. The fixture removes both coordination files on completion.

## Delivery boundary

The immediate-access backend and six prototype user content pages are implemented. The existing sidebar is reused. See CONSOLE_REDESIGN.md for the full page rewrite and its acceptance. No production rollout, push, release, or data backfill is included. Production provider behavior and conversion-rate improvement require deployment and subsequent observation.

## Local acceptance result (2026-11-15)

- Unified server: 55 checks passed.
- Existing payment, wallet, and refund flow: 55 checks passed.
- Both server runs verified 309 migration checksums and unchanged source manifests.
- Browser against the embedded production build: 13 checks passed. Inspected masked desktop and 390px mobile screenshots.
- Frontend production build, typecheck, scoped ESLint, Go package build, and whitespace checks passed.
- Receipts, source identities, browser replay script, screenshots, and artifact hashes are stored in `.local/unified-console-access/2026-11-15/` outside tracked source.
- Temporary PostgreSQL, Redis, server processes, private browser credentials, and browser TaskSpace were cleaned up.
