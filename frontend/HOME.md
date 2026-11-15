# Public home

The home is a one-to-one port of the approved landing prototype. It is a single Vue component, `src/views/HomeView.vue`, with scoped CSS and no new dependency. Brand marks come from `@lobehub/icons` and are inlined in `src/assets/lobe-icons.ts`.

## Behavior

Custom HTML or URL content in public settings keeps precedence over the landing page.

Site name, documentation URL, and registration availability come from public settings. The model plaza link keeps its feature flag and sign-in policy. Missing links are hidden.

Anonymous visitors open registration (or login when registration is disabled). Members and administrators open their dashboards, and the gift section shows the claimed state.

The page is light-only and Chinese-only, as in the prototype. Price rows, the savings calculator, and the three test-agency scores are example content. They are not live data.

## Reproduce acceptance

From `frontend`:

```sh
pnpm exec vue-tsc --noEmit
pnpm exec eslint src/views/HomeView.vue
node e2e/home.mjs
```

Browser acceptance requires `ego-browser`. The runner owns a fixture API on `127.0.0.1:18087` and Vite on `127.0.0.1:13037`. It does not call the production API.

Artifacts are saved to `output/home-boardui/`: screenshots, `checks.json`, fixture API requests, and the Vite log. The checks cover routing destinations, price tabs, the gauge, the calculator, the gift flow, model-plaza visibility, custom-content precedence, and 390px/320px overflow.
