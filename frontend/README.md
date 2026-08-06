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
│   ├── utils/                       # Pure, side-effect-free helpers
│   ├── services/                    # App-level singletons / startup logic
│   ├── constants/                   # appConstants.ts and similar
│   ├── types/                       # One .ts file per domain
│   └── mocks/                       # wailsMock.ts for dev/test
├── tests/
│   ├── unit/                        # *.spec.ts, mirrors src/ tree
│   ├── e2e/
│   ├── __mocks__/                   # wailsjs mock bindings
│   └── fixtures/
├── playwright-tests/                # Playwright e2e specs
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

## Testing

### Unit tests (Vitest)

```bash
npm test           # run once
npx vitest         # watch mode
```

- Location: `tests/unit/`, mirroring the `src/` tree
- Naming: `<SourceName>.spec.ts`
- Setup: `tests/setup.ts`; Wails mocks under `tests/__mocks__/wailsjs/`

### E2E tests (Playwright)

```bash
npm run test:e2e
```

- Location: `playwright-tests/*.spec.ts`
- Config: `playwright.config.js`

## Architecture & Conventions

See [`AGENTS.md`](./AGENTS.md) for the full architecture and development
guidelines, including:

- **Import rules** — aliases, barrel imports, `import type` for type-only imports
- **Component conventions** — SFC structure, props/events
- **Separation of concerns** — what belongs in `components/` vs `composables/`
  vs `utils/` vs `services/`
- **Styling** — scoped CSS, design tokens via CSS custom properties
- **Error handling** — narrowing `unknown` Wails payloads with shared helpers

## Recommended IDE Setup

- [VS Code](https://code.visualstudio.com/) + [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar)
  (disable the built-in TypeScript extension to enable Take Over mode for
  `.vue` type support)
