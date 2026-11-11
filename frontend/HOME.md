# Public home

The home uses the existing Vue adaptation of [BoardUI](https://www.boardui.com/components/color). It reuses Button, Badge, PlatformIcon, LocaleSwitcher, and existing semantic tokens. It does not add React, a component library, or a paid template.

## Behavior

Custom HTML or URL content keeps precedence. Compact mode shows the navigation, hero, routing illustration, and footer. Full mode adds three capability cards. Site name, logo, subtitle, and documentation come from public settings.

Anonymous actions open login. Member and administrator actions open their respective dashboards. Model plaza visibility retains the existing feature flag and sign-in policy. Theme selection persists. Landing copy uses the existing Chinese and English catalogs.

The routing illustration is descriptive, not live service telemetry. The home has no quick-start navigation, SDK examples, or endpoint-copy panel.

## Reproduce acceptance

From `frontend`:

```sh
pnpm exec vue-tsc --noEmit
pnpm exec eslint src/views/HomeView.vue src/i18n/locales/zh/landing.ts src/i18n/locales/en/landing.ts
pnpm exec vite build --outDir /tmp/truerouter-home-build
node e2e/home.mjs
```

Browser acceptance requires `ego-browser`. The runner owns an isolated fixture on `127.0.0.1:18087` and Vite on `127.0.0.1:13037`. It does not call the production backend. It stops both servers afterward. Fixture settings are handled only by the local E2E server.

Artifacts are saved to `output/home-boardui/`: screenshots, `checks.json`, fixture API requests, and the Vite log. The checks cover routing destinations, model-plaza permissions, compact/full behavior, custom-content precedence, absence of quick-start links and SDK panels, locale switching, persistent dark mode, desktop layout, and 390px/320px mobile layout including long settings.

These checks establish frontend behavior with fixture responses. They do not establish production deployment or live model inference.
