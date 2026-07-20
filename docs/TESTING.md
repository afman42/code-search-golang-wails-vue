# Testing

## Backend (Go)

17 test files covering search workflows, edge cases, error recovery, memory/performance, file reading, security, the log-polling server, and file collection optimizations:

- `app_test.go`, `binary_file_test.go`, `data_validation_test.go`, `debug_search_test.go`, `edge_cases_test.go`, `editor_detection_test.go`, `error_recovery_test.go`, `extended_app_test.go`, `improved_features_test.go`, `memory_performance_test.go`, `polling_server_test.go`, `read_file_test.go`, `search_with_progress_test.go`, `security_test.go`.
- `polling_noise_test.go` — noise filter consistency, log rotation memory leak, shutdown idempotency, CORS origin allowlist, re-init cleanup.
- `system_integration_fixes_test.go` — shell-metacharacter filename acceptance, null-byte/traversal rejection, table-driven editor bindings, snapshot-based editor count.
- `perf_regression_test.go` — zero-allocation `isBinary`, buffer pool reuse, `bytes.Split` path, literal-mode regex compile, redundant binary check removal.
- `file_collection_test.go` — two-phase collection: known-text extension recognition, walk splits text/binary candidates, parallel binary probe filtering, absPath computation (absolute + relative directories), prefix-based traversal check (including sibling-dir edge case), parallel probe scaling.

A separate `search_bench_test.go` holds benchmarks for the search pipeline (`go test -bench .`).

Notable coverage:
- Editor detection: `isEditorAvailable` with existing/non-existent commands, `countAvailableEditors` (including Neovim count, JetBrains derived flag), `GetAvailableEditors`, `GetEditorDetectionStatus`, `openInEditor` error handling, `OpenInEditorByName` dispatcher, `editorBindings` map completeness.
- Path traversal protection: validated across multiple attack vectors, including sibling-directory prefix edge cases.
- Input validation: regex patterns, directory paths, numeric limits, exclude patterns, literal-mode acceptance of invalid-regex strings.
- Binary file detection: null bytes, non-printable content, known-text extension shortcut, parallel probe filtering.
- Log-polling server: binds to loopback (`127.0.0.1`), not reachable on external IP, CORS allowlisted origins only, noise filter consistency between initial-load and live tail, log rotation bounded, shutdown idempotent.

```bash
go test -v ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
go test -bench . -benchmem    # run search benchmarks
```

## Frontend (Vitest)

13 test files with 208+ tests across components, composables, and utilities:

- `unit/components/` — `CodeModal.spec.ts` (24 tests), `CodeModal.syntax.spec.ts`, `LogViewer.spec.ts` (15 tests: collapse/expand, preview logs, placeholder, filtering, log parsing), `ProgressIndicator.spec.ts`, `SearchForm.spec.ts`, `SearchResults.spec.ts` (includes a test asserting highlighting runs only for the visible page).
- `unit/composables/` — `useSearch.spec.ts`, `useSearch.additional.spec.ts`, `useSearch.comprehensive.spec.ts`, `useSearch.fixes.spec.ts` (10 tests: truncation check respects maxResults, non-array results coerced to [], immediate editor-detection fetch, listener cleanup on completed/error/unmount), `useToast.spec.ts` (17 tests: add/remove, pause/resume, idempotent operations, concurrent staggered durations, rapid add/remove cycles).
- `unit/utils/` — `searchUiUtils.spec.ts` (23 tests: literal/regex matching, case sensitivity, ReDoS protection, XSS sanitization, lookahead, word boundaries, null/overflow inputs).
- `EnhancedTreeItem.spec.ts` — tree rendering, expansion, filtering, edge cases.

**Test infrastructure** (`frontend/tests/`):
- `setup.ts` — preloads highlight.js, mocks `IntersectionObserver`, `scrollIntoView`, clipboard fallback.
- `__mocks__/wailsjs/` — fake Wails binding modules so component tests run without a real bridge.
- `fixtures/` — shared test data (e.g. `editorAvailability.ts` with all 22 editor fields).

```bash
cd frontend
npm test               # run once
npm run test:watch     # watch mode
```

## Full validation

```bash
bash run_tests.sh      # Runs Go tests + Vitest + TypeScript check
```
