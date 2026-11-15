# sub2api 0.2.26

## Changes

- Provision one default unified API key at registration. Use the same key for supported Claude and GPT/Codex models. Change the request model ID to switch models.
- Admit unified-key requests through an eligible subscription first, then through a permitted wallet group. Record the actual billing source of each request. Legacy keys keep their previous group, limit, and billing behavior.
- Rewrite the six primary user pages: Home, API keys, Models, Usage, Billing, and Account. Reduce the ordinary-user sidebar to these six entries. Existing `/subscriptions`, `/redeem`, and `/orders` links remain usable. Administrator navigation is unchanged.
- Add personal gift credit notices to the notification bell. Claiming a gift only acknowledges the notice and never changes the wallet.
- Support Claude Fable and Mythos 5.2 from the upstream merge.
- Publish fork releases from `FORK_VERSION` with `tr-v*` tags. Historical `v*` releases remain available for update and rollback.

## Upgrade

This version adds migrations 256, 257, and 258. Migration 256 adds unified-key metadata; existing keys stay `legacy`. Existing users receive a default key when they open the dashboard. Migration 257 adds billing-source receipts. Migration 258 adds gift notices. Existing balances and historical credits are not backfilled. No wallet balance changes.

Operators must configure real accounts, models, routing groups, prices, subscription products, and signup gifts. The automatic production rollout requires a deployment receiver that accepts `tr-v*` tags.

## Verification

Local real-process acceptance passed: unified server (55 checks), existing payment/wallet/refund regression (55 checks), console browser acceptance (41 checks), and sidebar acceptance (11 checks). Gift notification acceptance scenarios were added; this release records no separate run result for them. Server runs verified 309 migration checksums. Synthetic providers and synthetic payments were used. Real merchant payments and production deployment were not performed for this release.
