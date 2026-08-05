# Testing

## Backend (Go)

23 test files covering search workflows, edge cases, error recovery, memory/performance, file reading, security, log buffer management, IPC validation, and file collection optimizations:- `app_test.go`, `binary_file_test.go`, `data_validation_test.go`, `edge_cases_test.go`, `editor_detection_test.go`, `error_recovery_test.go`, `extended_app_test.go`, `improved_features_test.go`, `memory_performance_test.go`, `read_file_test.go`, `search_with_progress_test.go`, `security_test.go`.
- `search_results_sort_test.go` — deterministic result ordering (sorted by file path then line) and the cancelled-search guard (returns empty instead of partial "completed" results).
- `symbol_index_test.go` — persistent symbol index: cache hit on unchanged fingerprint, cache miss + rescan on file change, `ClearSymbolCache` binding, eviction at max capacity, concurrent read/write safety.
- `multi_dir_test.go` — multi-directory search collects files from all roots; duplicate directories are deduplicated.
- `export_test.go` — CSV export rendering: header structure, field mapping, empty context fields, special characters (commas, quotes).
- `polling_noise_test.go` — noise filter consistency, log rotation memory leak, shutdown idempotency, shutdown done-channel signaling, re-init cleanup.
- `system_integration_fixes_test.go` — shell-metacharacter filename acceptance, null-byte/traversal rejection, table-driven editor bindings, snapshot-based editor count.
- `perf_regression_test.go` — zero-allocation `isBinary`, buffer pool reuse, `bytes.Split` path, literal-mode regex compile, redundant binary check removal.
- `file_collection_test.go` — two-phase collection: known-text extension recognition, walk splits text/binary candidates, parallel binary probe filtering, absPath computation (absolute + relative directories), prefix-based traversal check (including sibling-dir edge case), parallel probe scaling, and `TestGetKnownTextExtensions` which verifies the Wails binding that drives the frontend dropdown (sorted, no leading dot, excludes `.wasm`, round-trips with `isKnownTextExtension`).

`symbols_test.go` covers the symbol-extraction engine (`GetAllSymbols`/`SearchSymbols` across Go/TS/JS/Vue, directory skipping, `maxResults` truncation). `ipc_validation_test.go` and `optimization_test.go` cover binding input validation and search-path optimizations. A separate `search_bench_test.go` holds benchmarks for the search pipeline (`go test -bench .`).

Notable coverage:
- **Editor detection**: `isEditorAvailable` with existing/non-existent commands, `countAvailableEditors` (including Neovim count, JetBrains derived flag), `GetAvailableEditors`, `GetEditorDetectionStatus`, `openInEditor` error handling, `OpenInEditorByName` dispatcher, `editorBindings` map completeness.
- **Path traversal protection**: validated across multiple attack vectors, including sibling-directory prefix edge cases.
- **Input validation**: regex patterns, directory paths, numeric limits, exclude patterns, literal-mode acceptance of invalid-regex strings.
- **Binary file detection**: null bytes, non-printable content, known-text extension shortcut, parallel probe filtering.
- **Log buffer**: noise filter consistency between initial-load and live tail paths, log rotation bounded and GC-safe, shutdown idempotent, done-channel signaling, re-init cleans up previous manager.
- **Wails log bindings**: `TestGetInitialLogs` (nil manager, active manager with entries), `TestGetNewLogs` (nil manager, cursor advance, stale-entry cleanup), `TestGetInitialLogsWithActiveManager` (entries from previous activity).
- **Extension system**: `TestIsKnownTextExtension` covers recognized text extensions, case-insensitivity, and the safe-default behavior for unknown/binary extensions; `TestGetKnownTextExtensions` covers the binding's sort order, leading-dot stripping, `.wasm` exclusion, and round-trip with `isKnownTextExtension`.

```bash
go test -v ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
go test -bench . -benchmem    # run search benchmarks
```

> **Note**: The HTTP polling server (`polling_server_test.go`) has been removed — the frontend now consumes log entries via Wails IPC bindings (`GetInitialLogs`, `GetNewLogs`), not HTTP polling. The `polling_noise_test.go` file covers the buffer/tail core with updated tests that don't depend on an HTTP server.

## Frontend (Vitest)

30 test files with 456 tests across components, composables, and utilities:

- `unit/components/` — `CodeModal.spec.ts` (25 tests including language-detection cases for `jsx`/`tsx`/`vue`/`toml`/`txt`, plus a match-counter-clamping test at the last match), `CodeModal.syntax.spec.ts` (33 tests), `LogViewer.spec.ts` (15 tests: collapse/expand, preview logs, placeholder, filtering, log parsing), `ProgressIndicator.spec.ts` (4 tests), `SearchForm.spec.ts` (4 tests), `SearchResults.spec.ts` (6 tests, including a test asserting highlighting runs only for the visible page).
- `unit/components/` (new) — `TreeViewPanel.spec.ts` (7 tests: tree building from paths, folder/file ordering, counts, current-file highlight, file-click emission) and `SearchSuggestions.spec.ts` (10 tests: rendering from localStorage, select/remove, outside-click and Escape close, listener cleanup on unmount).
- `unit/composables/` — `useLogStreaming.spec.ts` (12 tests: `parseLogEntry` variations — structured JSON, noise filtering for both `Skipping` and `Sending file`, plain text, missing content, level field name variants, timestamp formatting; Wails binding mock resolution and cursor behavior), `useSearch.spec.ts` (10 tests), `useSearch.additional.spec.ts` (14 tests), `useSearch.comprehensive.spec.ts` (25 tests), `useSearch.fixes.spec.ts` (11 tests: truncation check respects maxResults, non-array results coerced to [], immediate editor-detection fetch, listener cleanup on completed/error/unmount, cancel-during-flight does not repopulate results), `useToast.spec.ts` (17 tests: add/remove, pause/resume, idempotent operations, concurrent staggered durations, rapid add/remove cycles).
- `unit/utils/` — `searchUiUtils.spec.ts` (33 tests), `searchUiUtils.memo.spec.ts` (9 tests), `fuzzyMatch.spec.ts` (20 tests), `localStorageUtils.spec.ts` (11 tests), and `diffUtils.spec.ts` (27 tests: match-range finding, diff-segment building, long-line truncation, HTML rendering, XSS sanitization, case-sensitivity, greedy matching edge cases).
- `unit/composables/` — `useFilePreview.spec.ts` (7 tests: openFile, closePreview, same-file content retention, different-file reload, options passthrough).
- `EnhancedTreeItem.spec.ts` (23 tests) — tree rendering, expansion, filtering, edge cases.

**Test infrastructure** (`frontend/tests/`):
- `setup.ts` — preloads highlight.js, mocks `IntersectionObserver`, `scrollIntoView`, clipboard fallback.
- `__mocks__/wailsjs/` — fake Wails binding modules so component tests run without a real bridge. Includes `GetKnownTextExtensions` returning a representative subset of the known-text extension list, plus `GetInitialLogs` and `GetNewLogs` for the log-streaming composable.
- `fixtures/` — shared test data (e.g. `editorAvailability.ts` with all 22 editor fields).

```bash
cd frontend
npm test               # run once
npm run test:watch     # watch mode
```

## End-to-end (Playwright)

`frontend/playwright-tests/` drives the real UX flows in a browser against a mocked Wails backend (`src/mocks/wailsMock.ts`, installed by `main.ts` when `VITE_WAILS_MOCK` is set). It uses the system Chrome (`channel: 'chrome'`) and auto-starts vite with the mock, so no Go process is needed.

9 flow tests across two specs:
- `flows.spec.ts` (7) — startup renders the UI (guards the "black screen" regression), Search Code populates results, an empty query keeps the button disabled, the file-preview modal opens with content, symbol search returns matches for a directory (and prompts to select one when absent), and the case-sensitive option is honored.
- `filetree-suggestions.spec.ts` (2) — the File Explorer tree in the preview modal lists all result files and opening one loads it (title + content + toggle state), and the recent-search suggestions dropdown appears on focus, selects a query, and closes on outside-click and Escape.

```bash
cd frontend
npm run test:e2e       # Playwright flows against the mock backend
npm run dev:mock       # serve the mocked frontend in a browser for manual testing
```

## Full validation

```bash
bash run_tests.sh              # Go tests + Vitest + TypeScript check (hermetic)
RUN_E2E=1 bash run_tests.sh    # also runs the Playwright E2E flows
```
