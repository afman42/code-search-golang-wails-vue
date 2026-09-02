# Code Search — Frontend

Vue 3 + TypeScript frontend for a [Wails](https://wails.io)-based code-search
tool. The Go backend exposes search, file-read, symbol-index, and
editor-detection bindings through `wailsjs/`; this frontend consumes them and
renders the full search → results → preview UX.

## Tech Stack

- **Vue 3** — `<script setup lang="ts">`, Composition API
- **TypeScript** — `strict: false`, `noUnusedLocals`/`noUnusedParameters` on
- **Vite 8** — dev server + bundler
- **Vitest** — unit tests (jsdom)
- **Playwright** — end-to-end tests
- **Wails** — generates the Go↔JS bindings under `wailsjs/` (do not hand-edit)

## Getting Started

```bash
# Install dependencies
npm install

# Dev server (Wails backend required for real search)
npm run dev

# Dev server with in-browser mock backend (no Go process needed)
npm run dev:mock

# Production build (type-checks first)
npm run build

# Preview the production build
npm run preview
```

## Project Structure

```
frontend/
├── src/
│   ├── App.vue                      # Root: startup loader → CodeSearch
│   ├── main.ts                      # createApp + service init
│   ├── style.css                    # Global styles + CSS custom properties
│   ├── components/
│   │   ├── CodeSearch.vue           # Top-level layout component
│   │   ├── StartupLoader.vue
│   │   └── ui/                      # Reusable UI components (.vue only)
│   ├── composables/                 # use*.ts — reactive state + domain logic
│   ├── utils/                       # Pure helpers: diffUtils, errorUtils, htmlUtils (escapeHtml), fuzzyMatch (findFuzzyMatches + debounce/DebouncedFn), localStorageUtils, ...
│   ├── services/                    # App-level singletons / startup logic
│   ├── constants/                   # appConstants.ts and similar
│   ├── types/                       # One .ts file per domain
│   └── mocks/                       # wailsMock.ts for dev/test
├── tests/
│   ├── unit/                        # *.spec.ts, mirrors src/ tree
│   ├── __mocks__/                   # wailsjs mock bindings
│   └── fixtures/
├── playwright-tests/                # Playwright e2e specs (the only e2e location)
├── wailsjs/                         # Generated Wails bindings (do not hand-edit)
├── tsconfig.json
├── vite.config.ts
└── vitest.config.ts
```

## Import Aliases

Two path aliases are defined in `tsconfig.json` `paths`:

| Alias | Resolves to | Use for |
|:---|:---|:---|
| `@/` | `src/` | All application source imports |
| `@wails/` | `wailsjs/` | Generated Wails Go↔JS bindings |

`vite.config.ts` resolves them natively via `resolve.tsconfigPaths: true`
(Vite 8+). `vitest.config.ts` mirrors them as explicit `resolve.alias` entries
(since vitest does not yet support `tsconfigPaths`).

```typescript
// ✅ Correct
import type { SearchState } from '@/types';
import { EditorSelect, CodeModal } from '@/components/ui';
import { SearchWithProgress, CancelSearch } from '@wails/go/main/App';
import { EventsOn } from '@wails/runtime';

// ❌ Incorrect — relative paths or direct file imports
import type { SearchState } from '../../types/search';
import EditorSelect from './EditorSelect.vue';
import { SearchWithProgress } from '../../wailsjs/go/main/App';
```

Barrel `index.ts` files exist in `types/`, `components/ui/`, `components/`,
`composables/`, `services/`, and `utils/` — import from the barrel rather than
individual files.

For `types/` the barrel is load-bearing: a type declared in
`src/types/search.ts` but missing from `src/types/index.ts` fails `vue-tsc` as
soon as a consumer imports it from `@/types`. Two exports are deliberately
file-only and must be imported by path: `useToast` (a barrel export would create
a composables → services → composables cycle) and `ExportFormat` from
`composables/useSelectionManager.ts`.

## Testing

### Unit tests (Vitest)

```bash
npm test           # run once
npx vitest         # watch mode
```

- Location: `tests/unit/`, mirroring the `src/` tree
- Naming: `<SourceName>.spec.ts`
- Setup: `tests/setup.ts`; Wails mocks under `tests/__mocks__/wailsjs/`.
  `tests/__mocks__/wailsjs/go/main/App.ts` is the one place bindings are
  stubbed — add a new binding there instead of re-mocking per spec
- `vitest.config.ts` declares coverage thresholds (lines/functions/statements
  80, branches 70), but `npm test` is plain `vitest run`, so they only apply
  under `npx vitest run --coverage`

### E2E tests (Playwright)

```bash
npm run test:e2e
```

- Location: `playwright-tests/*.spec.ts` — the only e2e location (`tests/e2e/`
  was empty and untracked; it has been deleted)
- Config: `playwright.config.js`. `retries` is 2 in CI and 0 locally;
  `reuseExistingServer` reuses a running mock server locally but always starts a
  fresh one in CI

### Type check

```bash
npx vue-tsc --noEmit
```

`vue-tsc`, not `tsc`: plain `tsc` skips `.vue` SFCs, so it can pass locally
while CI's `vue-tsc --noEmit` fails on a template type error. `npm run build`
runs the same check first.

See [`AGENTS.md`](./AGENTS.md) for the full architecture and development
guidelines, including:

- **Import rules** — aliases, barrel imports, `import type` for type-only imports
- **Component conventions** — SFC structure, props/events
- **Separation of concerns** — what belongs in `components/` vs `composables/`
  vs `utils/` vs `services/`
- **State patterns** — factory composables (fresh state per component) are the
  default; module-level singletons are reserved for app-global concerns
  (`toastManager`, `useTheme`); presentational components keep only transient
  UI state
- **Styling** — scoped CSS, design tokens via CSS custom properties
- **Error handling** — narrowing `unknown` Wails payloads with shared helpers
- **Security** — `index.html` ships a `Content-Security-Policy` meta (`default-src 'self'`) to limit script/style/connect sources in the Wails webview; rendered HTML goes through `escapeHtml` (`utils/htmlUtils.ts`) plus DOMPurify, and `useSearch` rejects queries over 2000 chars to match the backend cap

## Recommended IDE Setup

- [VS Code](https://code.visualstudio.com/) + [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar)
  (disable the built-in TypeScript extension to enable Take Over mode for
  `.vue` type support)
