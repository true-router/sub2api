# Account model provider acceptance

Run from the repository root:

```sh
node frontend/e2e/model-plaza-providers.mjs
```

Requires installed frontend dependencies and an available `ego-browser` connection.
Ports 13037 and 18087 must be free. The runner stops its Vite and fixture servers when it exits.

The check opens `/account/model-plaza` in the actual Vue frontend with a deterministic API fixture. It verifies company icons, model counts, manufacturer classification across compatible transports, combined search, keyboard selection, reset, wallet pricing, unknown models, 390px and 320px layouts, and error recovery.

Artifacts are written to `output/model-plaza-providers/`: `checks.json`, API requests, Vite logs, and desktop, mobile, and dark-mode screenshots. These artifacts verify frontend behavior with fixture data. They do not verify the production backend or deployment.

Manufacturer classification uses model family names first. Provider-specific platforms supply a fallback. A generic OpenAI-compatible transport does not identify the manufacturer. Unknown models appear under Other providers. The company marks are copied from the installed `@lobehub/icons` package with its MIT license preserved in the asset source.
