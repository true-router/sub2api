# Usage session acceptance

The admin usage table displays the explicit client session ID. Clicking it scopes
the detail list to the selected user and exact session ID. The current date range
and other filters remain active. Missing session IDs display `Not provided`.

Session filtering applies to usage details and their Excel export. Summary cards,
charts, error records, and user rankings retain their existing filter semantics.
Cleanup is disabled while a session filter is active because the cleanup API does
not support session filtering. No schema migration is required.

## HTTP and PostgreSQL acceptance

From the repository root:

```sh
sh backend/tools/usage-session-e2e/run.sh
```

The script creates disposable PostgreSQL, applies the production migrations, and
uses the real admin HTTP handler, service, repository, association hydration, and
DTO serialization. It checks exact matching, user scope, pagination, whitespace,
SQL parameter binding, missing IDs, filter reset, and date bounds. It removes the
fixture container and volume on exit. The metadata receipt defaults to
`/tmp/usage-session-e2e.json`; override it with `USAGE_SESSION_E2E_RECEIPT`.

## Browser acceptance

With Docker, frontend dependencies, and ego-browser available:

```sh
node frontend/e2e/usage-session.mjs
```

This command starts isolated loopback servers on available ports and uses real
usage HTTP queries backed by disposable PostgreSQL. Unrelated admin/auth/chart
endpoints use existing preview fixtures. It checks the default column, copy,
user-scoped drilldown, manual filtering, reset, sorting, cleanup protection,
filtered export, workbook contents, and narrow screens. Clipboard writes are
instrumented; the system clipboard is unchanged. These checks do not exercise
production authentication or call a model provider.

Receipts, request metadata, accessibility snapshots, server output, and the
synthetic Excel export are saved under `output/usage-session/`. All servers and
the PostgreSQL fixture are stopped after acceptance. Production is unchanged.
