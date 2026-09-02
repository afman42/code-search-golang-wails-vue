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
# Backend — same invocation run_tests.sh and CI use
go test -race -covermode=atomic -coverprofile=coverage.out -timeout 600s ./...

cd frontend
npm run test          # Vitest unit/component/composable suite
npm run test:e2e      # Playwright end-to-end suite (system Chrome)
npx vue-tsc --noEmit  # type check, including .vue SFC templates
npm run dev:mock      # VITE_WAILS_MOCK=1 vite — frontend in a browser with a mocked Wails backend (no Go process)
```

- **Full suite**: `bash run_tests.sh` runs the Go tests, the frontend Vitest suite, and the `vue-tsc` type check hermetically (no browser). Set `RUN_E2E=1 bash run_tests.sh` to add the Playwright stage. Each stage runs under `set +e` so a failure is reported at the end rather than aborting the rest.
- **`-race` is not optional**: `run_tests.sh` and CI both run `go test -race -covermode=atomic -coverprofile=coverage.out -timeout 600s ./...`. `-race` is what makes `TestLRUPatternCacheConcurrency` and `TestSymbolIndexCacheConcurrent` mean anything — they were written to catch data races and previously never ran under the detector. `-covermode=atomic` is required alongside `-race`, and the detector is slow enough to need 600s rather than the old 120s.
- **Type check is `vue-tsc`, not `tsc`**: `run_tests.sh` runs `npx vue-tsc --noEmit`. Plain `tsc` skips `.vue` SFCs, so a local `tsc --noEmit` pass could hide exactly the template type errors CI's `vue-tsc --noEmit` step catches. `npm run build` type-checks the same way (`vue-tsc --noEmit && vite build`).
- **Coverage**: the Go gate is 80% statement coverage, read from the `total:` line of `go tool cover -func=coverage.out`; the suite sits at 80.0% and passes. `frontend/vitest.config.ts` also declares thresholds (lines/functions/statements 80, branches 70), but they are **inert** while `npm test` is plain `vitest run` — run `npx vitest run --coverage` to actually enforce them.
- **E2E harness**: `frontend/playwright-tests/` (`flows.spec.ts`, `find-replace.spec.ts`, `filetree-suggestions.spec.ts`, `enhancements.spec.ts`, `search-options.spec.ts`, `advanced-search.spec.ts`, `fuzzy-search.spec.ts`) drives the app against an in-browser mock of the Wails backend (`frontend/src/mocks/wailsMock.ts`), which stubs `window.go.main.App` and `window.runtime`. The mock is installed in `main.ts` only when `VITE_WAILS_MOCK` is set (lazy dynamic import, tree-shaken from production builds). Tests use system Chrome via Playwright's `channel: 'chrome'`; `npm run test:e2e` auto-starts the mock Vite server.
- **E2E flake policy**: `frontend/playwright.config.js` sets `retries: process.env.CI ? 2 : 0` — CI absorbs genuine flake, a local failure fails immediately. `reuseExistingServer: !process.env.CI` likewise reuses an already-running mock server locally but always starts a fresh one in CI, so a stale or foreign server can never serve the tests.

## Lint & vulnerability checks

```bash
gofmt -l .          # must print nothing
go vet ./...
golangci-lint run ./...   # errcheck govet ineffassign staticcheck unused bodyclose misspell (config: .golangci.yml)
staticcheck ./...
govulncheck ./...
```

- **Toolchain**: `golangci-lint` v2 and `staticcheck` require Go ≥ 1.26, so `go install` switches toolchains automatically (`go.mod` pins `go 1.25.0`; CI's `setup-go` uses 1.26). Install the versions CI pins rather than `@latest`, or local results and CI can disagree: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2`, `go install honnef.co/go/tools/cmd/staticcheck@v0.7.0`, `go install golang.org/x/vuln/cmd/govulncheck@v1.1.4`.
- **`.golangci.yml`** enumerates its linter set explicitly instead of riding v2's defaults, so an upgrade cannot silently change what runs: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused` (v2's current defaults) plus `bodyclose` and `misspell`. No style linters — this codebase is written to a consistent hand style. Three noisy rules are narrowed: `errcheck` skips `(*os.File).Close` (deferred close on read-only opens has no actionable error path) and all `_test.go` files (setup failures surface as assertion failures); `staticcheck` drops `QF1003` (quickfix style, not correctness). All five commands above are clean on `main`.
- **CI**: `.github/workflows/build.yml` runs gofmt check → `go vet` → `golangci-lint` → `staticcheck` → `govulncheck` before the Go tests, so a lint regression fails the `test` job and blocks the `build` job. Every installed tool is pinned — `golangci-lint` `v2.13.2` (via `golangci/golangci-lint-action@v9`), `staticcheck@v0.7.0`, `govulncheck@v1.1.4`, and in the build job `garble@v0.14.2` and `wails@v2.14.0` (tracking `github.com/wailsapp/wails/v2` in `go.mod`; a CLI newer than the library is a real breakage source). These were all `@latest`, which made builds non-reproducible: an upstream release could red CI with no repo change, and two runs of the same commit could disagree. After the tests, CI enforces the 80% coverage gate and uploads `coverage.out` as a `go-coverage` artifact.
- **CI matrix — your platform file is no longer unverified**: the `test` job runs `strategy: fail-fast: false` over `matrix.os: [ubuntu-latest, windows-latest, macos-latest]`, and exactly three steps are ungated — Checkout, Setup Go, and `Go Tests`. So the full `-race` Go suite runs on all three OSes while lint, frontend, and E2E stay `if: matrix.os == 'ubuntu-latest'` (OS-independent; running them three times would triple CI minutes for identical results). `fail-fast: false` keeps one platform's failure from hiding the others. The consequence for local work: the build-tagged platform files — `app.go` (`//go:build linux`), `appDarwin.go` (`//go:build darwin`), `appWindows.go` (`//go:build windows`) — are now compiled and tested on their own OS, several of them for the first time. Editing one of them is no longer verified by a Linux-only run, so cross-check with `GOOS=darwin go build ./...` / `GOOS=windows go build ./...` before pushing rather than discovering it in the matrix. Non-Linux legs get a stub embed target (`Stub Frontend Dist (non-Linux)`, `if: matrix.os != 'ubuntu-latest'`) because `main.go` carries `//go:embed all:frontend/dist` and `frontend/dist` is gitignored, so the package will not compile on a fresh checkout without one.
- **Release**: the workflow triggers on `push` to `main`, on tags matching `v*`, and on `pull_request` against `main`. A third `release` job (`needs: build`, `if: startsWith(github.ref, 'refs/tags/v')`, `permissions: contents: write`) downloads every artifact the `build` job uploaded into `dist/<artifact-name>/` and publishes them with `gh release create "$GITHUB_REF_NAME" --generate-notes dist/*/*`. It is tag-only, so a push to `main` never creates a release — cut one by pushing a `v*` tag.

## Conventions

- **Go**: format with `go fmt ./...`; keep `golangci-lint`, `staticcheck`, and `govulncheck` clean (see above); use context for cancellation; add godoc on exported symbols.
- **Vue**: Composition API with `<script setup>`; TypeScript throughout. Large components are split into focused child components (e.g. `SearchForm` is composed of `ActionButtons`, `DirectoryPicker`, `QueryInput`, `SearchOptions`, `SizeLimitOptions`, `PatternSelector`, and `EditorStatusDisplay`).
- **Design system**: Consume tokens from `frontend/src/style.css` (`--color-*`, `--space-*`, `--radius-*`, `--shadow-*`, `--font-size-*`) in component styles — do not hard-code colors, radii, or spacing. Use the dark-surface tokens (`--color-surface-dark*`, `--color-border-dark`, `--color-text-dark*`) for deliberately dark panels (log viewer, history sidebar, suggestions dropdown), and keep content-semantic colors (log levels, diff colors) as-is.
- **TypeScript**: `strict` is off but `tsconfig.json` enables `noUnusedLocals` and `noUnusedParameters`, so dead locals/params fail the type check. Prefer `unknown` plus narrowing over `any` — the shared boundary helpers `toErrorMessage` and `asRecord` in `frontend/src/utils/errorUtils.ts` narrow untyped values (e.g. caught errors, event payloads); do not reintroduce `: any`/`as any`.
- **Types**: all backend types live in `models.go` — do not declare a request/result/progress struct beside the code that uses it. All shared frontend types live in `frontend/src/types/` (one file per domain) and **must** be re-exported from the barrel `frontend/src/types/index.ts`; a type that exists in `search.ts` but is missing from the barrel fails `vue-tsc`, which is how the `SearchResultBatch`/`ReplaceProgress` additions first broke the type check. Type-only exports belong in the `export type { … }` block; runtime values (the `isSearchStatus` guard) need a separate plain `export`.
- **Events**: new backend→frontend events go out through `a.safeEmitEvent` (`logger_utils.go`), never `wailsRuntime.EventsEmit` directly — it returns early on a nil or already-cancelled `a.ctx` and scopes a `recover()` to the `EventsEmit` call alone, so the "runtime not ready" panic Wails raises in tests and dev mocks is swallowed while a real panic elsewhere still surfaces. Two older emitters (`symbol-progress` in `app_symbols.go`, `app-ready` in `logger_utils.go`) still call `EventsEmit` directly behind a bare `a.ctx != nil` check; do not copy that shape. On the frontend every payload crosses the bridge as `unknown` and must be coerced defensively rather than cast — see `coerceProgress` and `coerceResultBatch` in `frontend/src/composables/searchProgress.ts`, which degrade a missing or renamed field to a default instead of throwing inside an event handler. Current event inventory: `search-progress`, `search-results`, `replace-progress`, `symbol-progress`, `editor-detection-start`/`-progress`/`-complete`, and the one-shot `app-ready`.
- **Composables**: Extract reusable state and logic into composables under `frontend/src/composables/`. Composables should encapsulate a single responsibility (e.g., `useSearch` for search state, `useLogStreaming` for log streaming, `useEditorDetection` for editor detection, `useFilePreview` for the singleton file-preview modal state, `useTheme` for dark/light theme) and expose a clean return interface. Lifecycle hooks (`onMounted`, `onUnmounted`) belong in composables, not components, so the logic is testable independently of the template.
- **Utils**: Pure utility functions live in `frontend/src/utils/` and should have no Vue imports (so they are unit-testable in isolation). Examples: `diffUtils.ts` (match-range diff + truncation), `fuzzyMatch.ts`, `localStorageUtils.ts`, `searchUiUtils.ts`, `errorUtils.ts`.
- **Symbol index**: The persistent symbol cache (`symbol_index.go`) is keyed by a directory fingerprint (file path + size + mtime hash). Every `get`/`set` normalizes the directory through `symbolCacheKey()` (`filepath.Abs` + `filepath.Clean`, mirroring `collectionCacheKey`), so `/a/b` and `./b` from `/a` share one slot instead of burning two on the same tree — normalize in the cache, not at the call sites. The cache is in-memory and per-process; call `ClearSymbolCache()` to force a re-index (the UI's Re-index button). The `globalSymbolIndex` variable is set by the App binding methods so the standalone extraction function can access it.
- **Export**: `ExportSearchResults` in `export.go` handles CSV/JSON export via Wails `SaveFileDialog`. The pure rendering function `renderResultsCSV` is testable without a Wails context.
- **Multi-directory search**: `SearchRequest.Directories` holds additional search roots. `SearchWithProgress` deduplicates and walks each root. The primary `Directory` field is always searched first.
- **Tests**: Go tests for backend functions; Vitest specs for components and composables. Composables should have their own test file (e.g., `useLogStreaming.spec.ts`) covering exported functions and state transitions; component tests should focus on template rendering and user interaction.
- **Security**: Keep input validation and path-safety checks intact when changing backend code.
- **File extensions**: The backend's `knownTextExtensions` map in `text_extensions.go` is the single source of truth for which file types are text. The frontend loads it via the `GetKnownTextExtensions()` Wails binding — do not hand-maintain a parallel extension list in the UI. The language-detection map in `syntaxHighlightingService.ts` is separate (it maps extensions to highlight.js languages, not to text/binary). See [`EXTENSIONS.md`](EXTENSIONS.md) for the full system and the steps to add a new extension.
- **Log streaming**: Log entries are delivered to the frontend via Wails IPC bindings (`GetInitialLogs`, `GetNewLogs`), not HTTP polling. The `useLogStreaming` composable encapsulates all streaming logic. Do not reintroduce HTTP polling — Wails bindings avoid CORS/mixed-content issues in production builds and are always available (same process).
- **Docs**: Update this file, the README, and the relevant `docs/` page when behavior changes.
