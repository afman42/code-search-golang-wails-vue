# Frontend Architecture & Development Guidelines

A Vue 3 + TypeScript frontend for a Wails-based code-search tool. This document
describes the conventions the existing codebase actually follows.

Tech stack: Vue 3 (`<script setup lang="ts">`, Composition API), TypeScript
(`strict: false`, `noUnusedLocals`/`noUnusedParameters` on), Vite 8, Vitest for
unit tests, Playwright for e2e. Wails generates the Go↔JS bindings under
`wailsjs/`.

---

## 1. Directory Structure

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

---

## 2. Import Rules

### 2.1 Use `@/` for `src/` and `@wails/` for `wailsjs/` alias
For example, if your `tsconfig.json` looks like this:
```typescript
"compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "@wails/*": ["wailsjs/*"]
    }
  }
```
For example, if your `vite.config.ts` looks like this:
```js
export default defineConfig({
    plugins: [vue()],
    resolve: {
      tsconfigPaths: true,
    }
});
```
```typescript
// ✅ Correct
import type { SearchState } from '@/types';

// ❌ Incorrect
import type { SearchState } from '../../types/search';
import EditorSelect from './EditorSelect.vue';
```

### 2.2 Barrel imports (`types/`, `utils/`, `composables/`, `services/`)

There is a barrel `index.ts` in each of these directories.

```typescript
// ✅ Correct
import type { SearchRequest } from '@/types';

// ❌ Don't
import type { SearchRequest, SearchResult } from '@/types/search';
import type { RecentSearch } from '@/types/recentSearch';
```

Use `import type` for interfaces/types; plain `import` is used when a type file
also exports runtime values (e.g. `search.ts` exports interfaces only, so
`import type` is fine there).

#### `@/utils`

```typescript
// ✅ Correct
import { findMatchRanges, buildDiffSegments, formatFilePath } from '@/utils';

// ❌ Don't
import { formatFilePath } from '@/utils/fileUtils';
import { findMatchRanges, buildDiffSegments } from '@/utils/diffUtils';
```

#### `@/composables`

```typescript
// ✅ Correct
import { 
    useCodeHighlighting,
    makeDefaultEditorAvailability,
    makeDefaultEditorDetectionStatus,
    subscribeToEditorDetectionEvents,
    startEditorDetection,
} from "@/composables";
  
// ❌ Don't
import { useCodeHighlighting } from "@/composables/useCodeHighlighting";
import {
  makeDefaultEditorAvailability,
  makeDefaultEditorDetectionStatus,
  subscribeToEditorDetectionEvents,
  startEditorDetection,
} from "@/composables/useEditorDetection";
```

#### `@/services`

```typescript
// ✅ Correct
import { 
  initializeAppServices,
  isHighlightJsLoaded,
  loadHighlightJs,
  detectLanguage,
  highlightCode,
  isHighlightingReady,
  getHighlightJs,
} from "@/services";

// ❌ Don't
import { initializeAppServices } from "@/services/appInitializationService";
import {
  isHighlightJsLoaded,
  loadHighlightJs,
  detectLanguage,
  highlightCode,
  isHighlightingReady,
  getHighlightJs,
} from "@/services/syntaxHighlightingService";
```

### 2.3 Components: one file barrel import

Inside `@/components/ui`:

```typescript
// ✅ Correct
import { EditorSelect, CodeModal } from '@/components/ui';

// ❌ Don't
import EditorSelect from '@/components/ui/EditorSelect.vue';
import CodeModal from '@/components/ui/CodeModal.vue';
```

Outside `@/components`:

```typescript
// ✅ Correct
import { EditorSelect, CodeModal } from '@/components';

// ❌ Don't
import EditorSelect from '@/components/ui/EditorSelect.vue';
import CodeModal from '@/components/ui/CodeModal.vue';
import { EditorSelect, CodeModal } from '@/components/ui';
```

### 2.4 Wails bindings

Generated bindings live under `wailsjs/`. Import with the correct relative
depth from `src/` (typically three levels up):

```typescript
import { SearchWithProgress, CancelSearch } from '@wails/go/main/App';
import { EventsOn } from '@wails/runtime';
```

Never hand-edit `wailsjs/` — it is regenerated by `wails dev`/`wails build`.

---

## 3. Component Conventions

### 3.1 SFC structure

All components use `<script setup lang="ts">`:

```vue
<template>
  <!-- markup -->
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { SearchState } from '@/types';
import { EditorSelect, CodeModal } from '@/components';

interface Props {
  directory: string;
  disabled?: boolean;
}
const props = withDefaults(defineProps<Props>(), { disabled: false });
const emit = defineEmits<{ select: []; update: [value: string] }>();
</script>

<style scoped>
/* component-scoped styles */
</style>
```

Order: `<template>` → `<script setup lang="ts">` → `<style scoped>`. Styling is
scoped per component; shared design tokens live as CSS custom properties in
`src/style.css` (e.g. `--color-accent`, `--radius-sm`, `--font-mono`).

### 3.2 Props and events

- Define props with `defineProps<T>()` (TS interface), almost always paired
  with `withDefaults` for optional props.
- Emit typed events with `defineEmits`. The codebase uses **both** valid
  syntaxes — pick either, but stay consistent within a file:
  ```typescript
  // Tuple short-form (preferred for new code; majority usage)
  const emit = defineEmits<{
    select: [];
    update: [value: string];
  }>();

  // Call-signature form (also present in older components)
  const emit = defineEmits<{
    (e: 'fileClick', path: string): void;
  }>();
  ```
- Child→parent communication is via events; parent→child is via props. Do not
  reach into a child's internals with template refs unless necessary.

---

## 4. Separation of Concerns

| Layer | Location | What goes here | What does NOT |
|:---|:---|:---|:---|
| **UI rendering** | `components/**/*.vue` | Template, presentational logic, transient UI state (toggle, hover, modal open) | API calls, business rules |
| **Reactive state / domain logic** | `composables/use*.ts` | `reactive`/`ref` state, Wails calls, event subscriptions, reusable domain behavior | Markup, CSS |
| **Pure helpers** | `utils/*.ts` | Synchronous, side-effect-free functions (formatting, parsing, fuzzy match) | State, network, DOM mutation |
| **App singletons** | `services/*.ts` | Startup orchestration, lazily-loaded shared modules | Per-component state |
| **Constants** | `constants/*.ts` | Plain exported `const` values | Logic, state |

### 4.1 Composable pattern

```typescript
// composables/useSearch.ts
import { reactive } from 'vue';
import { SearchWithProgress } from '@wails/go/main/App';
import type { SearchState } from '@/types';

export function useSearch() {
  const state = reactive<SearchState>({ /* ... */ });
  // domain logic using Wails bindings + utils
  return { state, /* exposed actions */ };
}
```

Composables return reactive state plus actions. For app-wide shared state, a
composable may hold module-level state and export a singleton instance
(`useToast.ts` does this with `export const toastManager = useToast()`).

### 4.2 Error handling for Wails boundaries

Wails binding rejections and event payloads arrive typed as `unknown`. Use the
shared helpers in `utils/errorUtils.ts` rather than ad-hoc casts:

```typescript
import { toErrorMessage, asRecord } from '@/utils';

try {
  await SearchWithProgress(req);
} catch (e) {
  toast.error(toErrorMessage(e));
}

// Narrowing an unknown event payload:
function coerceProgress(payload: unknown): SearchProgress {
  const p = asRecord(payload);
  // read fields with typeof checks — degrade to defaults, never throw
}
```

---

## 5. Styling

- **Scoped styles** per component via `<style scoped>`.
- **Design tokens** are CSS custom properties defined in `src/style.css`. Prefer
  `var(--color-accent)` over hard-coded values.
- **No CSS-in-JS.** No Tailwind. Plain CSS + custom properties.
- Global stylesheets (`src/style.css`, `src/assets/css/highlightjs-agate.css`)
  are imported once in `main.ts` / a service.

---

## 6. Testing

### 6.1 Unit tests (Vitest)

- Location: `tests/unit/`, mirroring the `src/` tree
  (`tests/unit/components/`, `tests/unit/composables/`, `tests/unit/utils/`,
  `tests/unit/services/`).
- Naming: `<SourceName>.spec.ts`. Additional specs for the same source append a
  suffix (`useSearch.fixes.spec.ts`, `useSearch.comprehensive.spec.ts`).
- Setup: `tests/setup.ts`; Wails mocks under `tests/__mocks__/wailsjs/`.
- Run: `npx vitest` (config in `vitest.config.ts`).

### 6.2 E2E tests (Playwright)

- Location: `playwright-tests/*.spec.ts` (also `tests/e2e/`).
- Config: `playwright.config.js`.
- Run: `npx playwright test`.

### 6.3 What to test

- Components: props→render behavior, emitted events, user interactions.
- Composables: state transitions, Wails call sequencing (mocked), event handling.
- Utils: pure-function behavior, edge cases, determinism.

---

## 7. Key Files Reference

| File | Role |
|:---|:---|
| `src/App.vue` | Root: shows `StartupLoader` until backend signals ready, then `CodeSearch` |
| `src/components/CodeSearch.vue` | Main layout, wires composables to child components |
| `src/composables/useSearch.ts` | Search state machine: directory select, search w/ progress, cancel, fuzzy match |
| `src/composables/useToast.ts` | Global toast store (`toastManager` singleton) |
| `src/composables/useEditorDetection.ts` | Detects available code editors |
| `src/services/appInitializationService.ts` | Lazy-loads highlight.js at startup |
| `src/constants/appConstants.ts` | Defaults, storage keys, timing constants |
| `src/utils/errorUtils.ts` | `toErrorMessage`, `asRecord` — shared narrowing helpers |
| `wailsjs/go/main/App.ts` | Generated Go bindings (regenerated by Wails) |

---

## 8. Do / Don't Summary

**Do**
- Use `import type` for type-only imports.
- Put reactive/domain logic in `composables/use*.ts`, pure helpers in `utils/`.
- Wrap Wails calls in try/catch; narrow `unknown` with `toErrorMessage`/`asRecord`.
- Mirror the `src/` tree in `tests/unit/` with `*.spec.ts` files.
- Use CSS custom properties from `src/style.css` for theming.
- Create barrel `index.ts` files in `types/` or `components/ui/` or
  `utils/` or `components/` or `services/` or `composables/`.
- Use the `@/` and `@wails/` aliases.

**Don't**
- Don't write `.tsx`/`.jsx` — this is a Vue SFC project.
- Don't hand-edit `wailsjs/` — it's generated.
- Don't put business logic or API calls inside `.vue` `<script setup>`; extract
  to a composable.
- Don't mutate reactive state outside the owning composable's returned actions.
