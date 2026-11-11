# BoardUI-inspired Vue workspace

This project uses a Vue adaptation of BoardUI's design language. It does **not** use the React component runtime or claim compatibility with the BoardUI React registry. The existing Vue 3, Tailwind CSS 3, Vue Router, Pinia, Chart.js, and VueUse stack remains in place.

## Design sources

- [BoardUI semantic colors and typography](https://www.boardui.com/components/color)
- [BoardUI sidebar](https://www.boardui.com/components/sidebar)
- [BoardUI component registry](https://www.boardui.com/r/sidebar.json)

The implementation adapts the floating navigation, neutral surfaces, blue selected state, 24px panel radii, compact controls, composite typography, and restrained motion. Existing Vue components keep their business events, slots, keyboard handling, and API contracts.

## Files and ownership

- `src/styles/boardui-tokens.css`: light/dark semantic roles. RGB channels support Tailwind CSS 3 opacity modifiers.
- `tailwind.config.js`: semantic color utilities and composite type utilities. Do not install the Tailwind CSS 4 registry stylesheet into this build.
- `src/styles/boardui-admin.css`: workspace component skin, activated by `data-admin-ui="boardui"` on the document root.
- `src/App.vue`: activates the skin for admin routes and account routes marked with `workspace: 'account'`. This appearance metadata does not affect authorization. The document boundary includes teleported dialogs and dropdowns. Leaving the workspace removes the attribute.
- `src/components/layout`: existing Vue shell, grouped navigation, navigation search, mobile drawer, and collapse behavior.
- `src/components/admin/AdminMetricCard.vue`: statistics card with a footer slot for existing supporting metrics.
- `src/views/admin` and `src/components/admin`: existing business pages using semantic colors and composite type utilities.

The adopted admin layout follows the user-provided medical-profile reference: workspace and quick search at the top of a floating sidebar; navigation in the middle; theme, settings, and account controls at the bottom. There is no global top header in the workspace. Breadcrumb, page heading, and page-local actions belong to the right content region.

The My Account routes share this workspace and use the Vue ports for actions, fields, role and status badges, segmented controls, and pagination. The command palette lists only entries available to the signed-in role. Public pages, authentication pages, the standalone user dashboard, and payment redirects do not use the workspace shell.

## Maintenance

Use role names such as `text-text-primary`, `bg-background-primary-default`, and `border-border-button-default`. Add a semantic role when the existing roles cannot express a real interaction. Change token definitions to rebrand the interface.

Use composite typography such as `text-title-1-medium`, `text-body-regular`, and `text-caption-1-medium`. Keep spacing on the existing 4px scale. Use the installed Vue icons and Vue components. Do not add React to display an icon or control.

Preserve the existing `DataTable` virtualization and sticky-column geometry. Preserve dialog focus management, Escape dismissal, and scroll locking. Dark mode must follow the root `.dark` class. Honor reduced motion. Do not shrink buttons on press.

## Reproducible acceptance

From `frontend`:

```sh
pnpm install --frozen-lockfile
pnpm exec vue-tsc --noEmit
pnpm exec vite build --outDir /tmp/sub2api-boardui-build
node e2e/preview-admin.mjs
# Run in a second terminal.
node e2e/check-preview.mjs layout
node e2e/check-preview.mjs routes
```

The E2E runner requires the installed `ego-browser` CLI. The preview starts a local fixture API on `127.0.0.1:18081` and Vite on `127.0.0.1:13031`. It creates synthetic users only in memory and does not connect to a production API.

The browser checks write screenshots and assertion results under `output/`. These artifacts establish frontend behavior with fixture responses. They do not establish production API integration or every provider's account-management workflow.

## Component-level Vue ports

The first visual-only pass was insufficient. The following ports now provide real Vue components and replace live callers. These are local adaptations of the free registry source, not the original React packages. React Aria runtime code is not copied into the application. Vue components preserve existing native semantics, Vue models, and the project's dialog focus handling.

| BoardUI registry component | Vue implementation | Live adoption and behavior |
| --- | --- | --- |
| [Button](https://www.boardui.com/r/button.json) / icon button | `components/boardui/Button.vue` | 379 management actions across 79 modules; variants, sizes, loading, disabled, form types, icons, merged caller classes |
| [Input](https://www.boardui.com/r/input.json) | `components/boardui/Input.vue` | Shared Input adapter and user creation; unique labels, hints, error associations, password reveal, prefix/suffix |
| [Search field](https://www.boardui.com/r/search-field.json) | `components/boardui/SearchField.vue` | User search and shared SearchInput; clear action, Escape, submit, loading |
| [Number field](https://www.boardui.com/r/number-field.json) | `components/boardui/NumberField.vue` | User balance, concurrency, and RPM fields; steppers, bounds, native keyboard input |
| [Table](https://www.boardui.com/r/table.json) | `components/boardui/Table.vue` | Shared DataTable and standalone tables; native table semantics, compact and regular sizes, semantic headers and row dividers |
| Checkbox | `components/boardui/Checkbox.vue` | Shared DataTable selection; native input, indeterminate state, focus ring, animated check |
| [Switch](https://www.boardui.com/r/switch.json) | `components/boardui/Switch.vue` | Shared Toggle adapter; 3 sizes, 2 shapes, disabled state, stable light thumb, native button semantics |
| [Badge](https://www.boardui.com/r/badge.json) | `components/boardui/Badge.vue` | Shared StatusBadge; status tone, dot, optional removable tag |
| [Modal](https://www.boardui.com/r/modal.json) | `components/common/BaseDialog.vue` | Existing management dialogs; description, header actions, focus trap/restoration, Escape, reduced motion |
| [Drawer](https://www.boardui.com/r/drawer.json) | `components/boardui/Drawer.vue` | User filter settings; side panel reuses dialog focus and scroll ownership |
| [Breadcrumb](https://www.boardui.com/r/breadcrumb.json) | `components/boardui/Breadcrumb.vue` | Management header; linked ancestors and current-page semantics |
| [Command palette](https://www.boardui.com/r/command-palette.json) | `components/boardui/CommandPalette.vue` | Global management navigation; permission-filtered items, search, arrows, Enter, Escape, Cmd/Ctrl+K |
| [PillTab](https://www.boardui.com/zh-hans/components/tabs#pill-tabs) | `components/boardui/PillTabs.vue` | System settings navigation; gray selection thumb, keyboard navigation, horizontal overflow, and a labelled tab panel |
| [Segmented control](https://www.boardui.com/r/segmented-control.json) | `components/boardui/SegmentedControl.vue` | Dashboard distribution/ranking selector; moving indicator, resize measurement, radio semantics, arrow/Home/End keys |
| [Pagination](https://www.boardui.com/r/pagination.json) | `components/boardui/Pagination.vue` | Shared desktop pagination; existing page-size/page contracts, page cells, disabled boundaries, keyboard navigation |
| [Skeleton](https://www.boardui.com/r/skeleton.json) | `components/boardui/Skeleton.vue` | Shared Skeleton adapter; dimensions and variants, sheen, reduced motion |
| [Text area](https://www.boardui.com/r/textarea.json) | `components/boardui/TextArea.vue` | Shared TextArea adapter; auto height, limits/count, labels, hint/error associations |
| [Toast](https://www.boardui.com/r/toast.json) | `components/boardui/Toast.vue` | Existing app toast store; stacked views, status semantics, close action, duration progress |
| [Stat cards](https://www.boardui.com/r/stat-cards.json) | `components/admin/AdminMetricCard.vue` | Dashboard; icon tile, label/value hierarchy, inner footer tile, actual API metrics |

DataTable renders the Vue port of the BoardUI Table primitive. The adapter keeps Vue slots, sticky columns, mobile cards, TanStack virtualization, and the existing sorting and selection contracts. Sort buttons accept Enter and Space. Standalone tables use the same primitive with their existing scroll containers. The searchable Select and the React Aria grid implementation remain separate. Paid chart/template source is not used.

Start a manual preview with `node e2e/preview-admin.mjs`. This uses the same `e2e/fixtures/admin-api.mjs` contracts as browser acceptance, preventing divergence between the demonstration and the test server. Preview credentials are `admin@example.test` / `BoardUI-Preview-2026` and have no production meaning.

Component acceptance: `node e2e/check-preview.mjs components`.
Route/console regression: `node e2e/check-preview.mjs routes`.
Both run against the already-running local preview. They write results and screenshots under `output/`.

Layout and sidebar-state acceptance: `node e2e/check-preview.mjs layout`. The selected row follows the registry sidebar's 36px height, 10px radius, blue-500-to-blue-600 vertical gradient, 1px accent outline, and 25.1% inner top highlight. The account card shows the available balance before its menu is opened.

Account workspace acceptance:

```sh
node e2e/preview-admin.mjs --account
# Run in a second terminal.
node e2e/check-preview.mjs account
```

The account preview runs on port 13032 and uses an isolated in-memory fixture. Sign in with `admin` / `admin`. The short credentials are enabled only in this local preview. The acceptance script checks account routes, role boundaries, and form and navigation interactions. It writes screenshots and `checks.json` to `output/account-boardui/`. It does not submit payments or change account credentials.

Table acceptance: with the account preview running, run `node e2e/check-preview.mjs table`. The script checks native markup, compact cells, server sort requests, selection, pagination, sticky geometry, standalone tables, and mobile cards. Artifacts are in `output/boardui-table/`.
