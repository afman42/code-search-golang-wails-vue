# Development

## Setup

```bash
go mod tidy
cd frontend && npm install && cd ..
```

## Run

```bash
wails dev     # hot-reload development server (Vite + Go)
wails build   # production binary in build/bin/
```

## Testing

```bash
cd frontend
npm run test          # Vitest unit/component/composable suite
npm run test:e2e      # Playwright end-to-end suite (system Chrome)
npm run dev:mock      # VITE_WAILS_MOCK=1 vite — frontend in a browser with a mocked Wails backend (no Go process)
```

- **Full suite**: `bash run_tests.sh` runs Go tests + the frontend Vitest suite hermetically (no browser). Set `RUN_E2E=1 bash run_tests.sh` to add the Playwright stage with the backend included.
- **E2E harness**: `frontend/playwright-tests/` (`flows.spec.ts`, `filetree-suggestions.spec.ts`, `enhancements.spec.ts`) drives the app against an in-browser mock of the Wails backend (`frontend/src/mocks/wailsMock.ts`), which stubs `window.go.main.App` and `window.runtime`. The mock is installed in `main.ts` only when `VITE_WAILS_MOCK` is set (lazy dynamic import, tree-shaken from production builds). Tests use system Chrome via Playwright's `channel: 'chrome'`; `npm run test:e2e` auto-starts the mock Vite server.

## Conventions

- **Go**: format with `go fmt ./...`; use context for cancellation; add godoc on exported symbols.
- **Vue**: Composition API with `<script setup>`; TypeScript throughout. Large components are split into focused child components (e.g. `SearchForm` is composed of `ActionButtons`, `DirectoryPicker`, `QueryInput`, `SearchOptions`, `SizeLimitOptions`, `PatternSelector`, and `EditorStatusDisplay`).
- **Design system**: Consume tokens from `frontend/src/style.css` (`--color-*`, `--space-*`, `--radius-*`, `--shadow-*`, `--font-size-*`) in component styles — do not hard-code colors, radii, or spacing. Use the dark-surface tokens (`--color-surface-dark*`, `--color-border-dark`, `--color-text-dark*`) for deliberately dark panels (log viewer, history sidebar, suggestions dropdown), and keep content-semantic colors (log levels, diff colors) as-is.
- **TypeScript**: `strict` is off but `tsconfig.json` enables `noUnusedLocals` and `noUnusedParameters`, so dead locals/params fail the type check. Prefer `unknown` plus narrowing over `any` — the shared boundary helpers `toErrorMessage` and `asRecord` in `frontend/src/utils/errorUtils.ts` narrow untyped values (e.g. caught errors, event payloads); do not reintroduce `: any`/`as any`.
- **Composables**: Extract reusable state and logic into composables under `frontend/src/composables/`. Composables should encapsulate a single responsibility (e.g., `useSearch` for search state, `useLogStreaming` for log streaming, `useEditorDetection` for editor detection, `useFilePreview` for the singleton file-preview modal state, `useTheme` for dark/light theme) and expose a clean return interface. Lifecycle hooks (`onMounted`, `onUnmounted`) belong in composables, not components, so the logic is testable independently of the template.
- **Utils**: Pure utility functions live in `frontend/src/utils/` and should have no Vue imports (so they are unit-testable in isolation). Examples: `diffUtils.ts` (match-range diff + truncation), `fuzzyMatch.ts`, `localStorageUtils.ts`, `searchUiUtils.ts`, `errorUtils.ts`.
- **Symbol index**: The persistent symbol cache (`symbol_index.go`) is keyed by a directory fingerprint (file path + size + mtime hash). The cache is in-memory and per-process; call `ClearSymbolCache()` to force a re-index. The `globalSymbolIndex` variable is set by the App binding methods so the standalone extraction function can access it.
- **Export**: `ExportSearchResults` in `export.go` handles CSV/JSON export via Wails `SaveFileDialog`. The pure rendering function `renderResultsCSV` is testable without a Wails context.
- **Multi-directory search**: `SearchRequest.Directories` holds additional search roots. `SearchWithProgress` deduplicates and walks each root. The primary `Directory` field is always searched first.
- **Tests**: Go tests for backend functions; Vitest specs for components and composables. Composables should have their own test file (e.g., `useLogStreaming.spec.ts`) covering exported functions and state transitions; component tests should focus on template rendering and user interaction.
- **Security**: Keep input validation and path-safety checks intact when changing backend code.
- **File extensions**: The backend's `knownTextExtensions` map in `text_extensions.go` is the single source of truth for which file types are text. The frontend loads it via the `GetKnownTextExtensions()` Wails binding — do not hand-maintain a parallel extension list in the UI. The language-detection map in `syntaxHighlightingService.ts` is separate (it maps extensions to highlight.js languages, not to text/binary). See [`EXTENSIONS.md`](EXTENSIONS.md) for the full system and the steps to add a new extension.
- **Log streaming**: Log entries are delivered to the frontend via Wails IPC bindings (`GetInitialLogs`, `GetNewLogs`), not HTTP polling. The `useLogStreaming` composable encapsulates all streaming logic. Do not reintroduce HTTP polling — Wails bindings avoid CORS/mixed-content issues in production builds and are always available (same process).
- **Docs**: Update this file, the README, and the relevant `docs/` page when behavior changes.
