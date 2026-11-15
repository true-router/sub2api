# Gift Notifications

## User behavior

The notification bell now has two types: system announcements and personal gift credit. The gift dialog follows `.planning/2026-11-14-console-redesign/prototype.html` (`dialog.gift`). The closed state shows a wobbling box, an eyebrow ("Welcome gift" for `initial` gifts), and an **Open gift** action; it does not reveal the amount. Clicking the box or the action shakes the box while the claim request runs (at least 500 ms). On success, the lid flies off, rays spin, 36 confetti pieces burst, and the amount counts up over 900 ms. A failed claim returns to the closed state with an error. **Start using my gift** closes the dialog, navigates to the dashboard, and shows a balance toast. Reduced motion skips the shake, confetti, and count-up. The dialog uses a native modal `<dialog>` and fits narrow screens.

Gift credit is spendable when the credit transaction commits. Claiming is an acknowledgement, not a second funding transaction. The current wallet balance remains the source of truth for funds that are still available. A gift notice remains a historical receipt if its funds have already been spent or reclaimed.

Opening or dismissing a gift leaves it unread. The user must click **Claim gift** and receive a successful acknowledgement before the dialog shows the opened state. A failed request keeps the claim action available. Claim replay preserves the first acknowledgement time and cannot add funds.

**Mark announcements read** acknowledges only system announcements. It never claims gifts. Dismissed gifts stay in the inbox and may appear again in a new session. A claimed gift can be reviewed as claimed without another claim button.

The inbox checks for updates on a 30-second tick while the authenticated page is visible, with a 30-second request throttle. Opening the bell forces a fresh request. The popup queue identifies entries by both type and ID. Existing announcement publication and legacy announcement endpoints remain available.

## Durable event sources

Migration 258 adds `gift_notifications`, separate from public announcements. Each row belongs to one user and contains an immutable credited amount, event identity, source, creation time, and optional acknowledgement time.

- A new user with a positive initial promotional balance creates an initial gift notice.
- A positive ordinary wallet adjustment creates a credit gift notice. This covers administrator promotional credits and existing redemption/reward paths that use that accounting operation.
- A completed payment with an associated promotional grant creates a notice for the gift portion only.
- Purchased principal, negative adjustments, paid-to-promotional reclassification, historical source verification, and hold/refund release do not create new gift notices.
- Existing balances and historical credits are not backfilled.

The user-insert and wallet-journal triggers insert notices in the same database transaction as the credit. Rollback removes both the credit and its notice. The event key is unique. Existing credit/payment idempotency prevents replayed funding from producing another journal event or notice.

## API

`GET /api/v1/notifications` combines visible system announcements and the authenticated user's personal gifts. Each item has an `id` and a `type` (`announcement` or `gift`). Gift items additionally contain `gift_amount` and `gift_source`. IDs are scoped by type; callers must not treat a numeric ID alone as notification identity.

`POST /api/v1/notifications/:type/:id/read` acknowledges the item for the authenticated owner. A gift acknowledgement uses a conditional owner-scoped update and `COALESCE(read_at, NOW())`. It never modifies wallet tables. Cross-user gift access returns not found. Anonymous access requires authentication.

The gift repository returns all unread gifts and the most recent 20 acknowledged gifts. Public announcement targeting remains independent. Frontend list updates do not truncate unread gifts. Account logout invalidates pending inbox reads and clears session popup state.

## Reproducible acceptance

```sh
pnpm --dir frontend build
./backend/tools/production-billing-e2e/run.sh \
  --gift-notifications \
  --browser-ready /tmp/sub2api-gifts-browser-ready.json \
  --output /tmp/sub2api-gifts-server-final.json
```

In another terminal, run:

```sh
ego-browser nodejs < frontend/e2e/gift-notifications.browser.mjs
```

The fixture builds the actual server and embedded frontend, runs real migrations against disposable PostgreSQL and Redis, and uses synthetic external payment services. The browser uses the real login, inbox, claim, and wallet routes. The private browser coordination file contains only disposable fixture credentials and is removed after completion.

The server scenario covers administrator credits, credit replay, owner isolation, typed ID collisions, claim replay, no extra wallet credit, non-gift operations, payment bonus delivery, callback replay, registration gifts, transactional rollback, and real process restart. The rollback probe intentionally executes a failed transaction; it commits no funding changes. Other funding actions use actual authenticated HTTP routes.

The browser scenario covers opening, dismissal, announcement popup acknowledgement and scroll unlock, announcement detail reads, announcement-only bulk acknowledgement, failed claim retry, successful opening animation, unchanged wallet balance, restart/reload read state, mobile layout, and reduced motion. Receipts and screenshots are written under `/tmp/sub2api-gift-notifications-browser/`. Server receipts include source manifests, binary hashes, and migration verification.

Run the existing payment and refund regression independently:

```sh
./backend/tools/production-billing-e2e/run.sh --output /tmp/sub2api-gifts-legacy.json
```

No production rollout or live payment is part of this change.

## Local verification (2026-11-15)

- Gift server scenario: 20 checks passed.
- Existing payment, wallet, and refund regression: 55 checks passed.
- Actual compiled-frontend browser scenario: 16 checks passed.
- Both server runs verified 310 source migration checksums. Source manifests stayed unchanged during acceptance; the browser run includes compiled frontend hashes.
- Inspected desktop unclaimed/claimed and mobile claimed screenshots. Reduced motion follows the existing global 0.01ms transition policy and disables gift-box animation.
- Ordinary announcement popups now use BaseDialog for focus and scroll locking. Admin previews keep their existing props and close behavior.
- Artifacts and SHA256 hashes: `.local/gift-notifications/2026-11-15/`.
- The local development database was backed up before migration. `console-user@local.test` received one idempotent $5 demonstration gift, left unread for manual interaction. No real payment or production change occurred.

## Animation alignment verification (2026-11-15)

- Gift server scenario: 21 checks passed. Actual compiled-frontend browser scenario: 16 checks passed.
- The browser check for announcement detail reads now polls for up to 5 seconds. The bell opens the detail before its read request completes, so an immediate check was timing dependent.
- Screenshots wait for finite animations before capture. Inspected desktop unclaimed/claimed and mobile claimed screenshots against the prototype.
- Artifacts and SHA256 hashes: `.local/gift-notifications/2026-11-15-animation/`.
