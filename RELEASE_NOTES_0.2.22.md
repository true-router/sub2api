# sub2api 0.2.22

## Changes

- Display client session IDs in the admin usage table. Select a session to filter usage details by the exact session ID and user while preserving the date range and other active filters.
- Apply the session filter to Excel exports. Disable usage cleanup while a session filter is active because the cleanup API does not support that filter.
- Use the shared login layout and BoardUI components on the forgot-password page. Display inline errors and the submitted email address after a successful request.

## Upgrade notes

No database migrations are added in this release. Summary cards, charts, error records, and user rankings retain their existing filter behavior.
