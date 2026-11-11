# Production billing acceptance

Run from the repository root:

```sh
backend/tools/production-billing-e2e/run.sh
```

Requirements: Go matching `backend/go.mod`, Git, and a running Docker daemon.
Docker must provide `postgres:18-alpine` and `redis:8-alpine`, or be able to pull
them. The wrapper runs from `backend` regardless of the caller's directory.

The default receipt is `.local/production-billing-e2e/receipt.json`. Select another
path with `-output`; relative paths resolve from `backend`:

```sh
backend/tools/production-billing-e2e/run.sh \
  -output ../.local/production-billing-e2e/review.json
```

The command exits nonzero when acceptance, service cleanup, or receipt writing
fails. Each run creates dedicated PostgreSQL and Redis containers, builds
`cmd/server`, and starts an actual server process in `standard` mode. It removes
its server, containers, temporary configuration and credentials after the run.
It does not reuse local development databases or change existing services.

## Failure catalog

These failures define the acceptance contract:

1. The production setup or complete migration chain fails on a fresh database.
2. Anonymous, expired, forged, non-admin or cross-user access bypasses guards.
3. An unpaid order, invalid signature or wrong amount credits funds.
4. A repeated signed payment callback credits funds twice.
5. Gateway pricing charges the wrong amount or spends principal before gifts.
6. An unsuccessful upstream request debits funds or records successful usage.
7. Warmed caches allow spending after a refund empties a wallet.
8. A refund returns gifts as cash or uses converted gifts to replace principal.
9. Conversion changes total funds or repeated requests convert funds twice.
10. A conflicting idempotency payload succeeds.
11. Process restart loses balances, credentials or completed idempotency records.
12. A revoked API key still reaches the model service through a warmed cache.
13. Usage records, conversion journals or administrator audit records are absent.
14. Fixture processes, containers or temporary credentials remain after cleanup.

## Real application path

All business setup and changes use production HTTP routes: password login,
compliance acknowledgement for the synthetic administrator, user/group/account
creation, API key creation, provider configuration, checkout, signed callback,
model gateway consumption, user refund request, administrator refund, conversion
and API key revocation. SQL queries only inspect results. They never create or
change funds, orders, users, or accounting records.

The fixture checks each applied migration checksum against the source SQL. It
compares exact decimal wallet values with both PostgreSQL and the profile API.
It verifies successful usage counts and costs. It stops the actual process with
SIGTERM, starts it again, and reuses earlier JWTs, API keys and idempotency keys.

The JSON receipt includes individual checks, wallet snapshots, external request
paths, migration count, process start count, cleanup results, source commit,
backend diff hash, executable hash, and harness hashes. Archive the receipt with
the matching source revision. A receipt is scenario evidence, not a line or
branch coverage percentage.

## External boundaries and limits

Only external model responses, remote price data, and the EasyPay merchant API
are simulated with a local HTTP server. The application uses its real EasyPay
adapter, signature validation, provider verification, fulfillment and refund
logic. No real money moves.

This suite covers HTTP application paths. It does not operate the browser UI or
validate real merchant credentials, every payment adapter, streaming/token-priced
model variants, historical customer-data upgrades, high concurrency, or forced
crashes between database commits. Restart checks use graceful termination after
completed requests. They do not establish crash-window atomicity. CI enforcement
is a separate task.
