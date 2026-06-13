# Testing

## Backend (Go)

13 test files covering search workflows, edge cases, error recovery, memory/performance, file reading, and security:

- `app_test.go`, `binary_file_test.go`, `data_validation_test.go`, `debug_search_test.go`, `edge_cases_test.go`, `editor_detection_test.go`, `error_recovery_test.go`, `extended_app_test.go`, `improved_features_test.go`, `memory_performance_test.go`, `read_file_test.go`, `search_with_progress_test.go`, `security_test.go`.

Notable coverage:
- Editor detection: `isEditorAvailable` with existing/non-existent commands, `countAvailableEditors` (including Neovim count, JetBrains derived flag), `GetAvailableEditors`, `GetEditorDetectionStatus`, `openInEditor` error handling.
- Path traversal protection: validated across multiple attack vectors.
- Input validation: regex patterns, directory paths, numeric limits, exclude patterns.
- Binary file detection: null bytes, non-printable content.

```bash
go test -v ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Frontend (Vitest)

12 test files with 197+ tests across components, composables, and utilities:

- `unit/components/` — `CodeModal.spec.ts` (24 tests), `CodeModal.syntax.spec.ts`, `LogViewer.spec.ts` (15 tests: collapse/expand, preview logs, placeholder, filtering, log parsing), `ProgressIndicator.spec.ts`, `SearchForm.spec.ts`, `SearchResults.spec.ts`.
- `unit/composables/` — `useSearch.spec.ts`, `useSearch.additional.spec.ts`, `useSearch.comprehensive.spec.ts`, `useToast.spec.ts` (17 tests: add/remove, pause/resume, idempotent operations, concurrent staggered durations, rapid add/remove cycles).
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
