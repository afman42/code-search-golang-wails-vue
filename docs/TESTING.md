# Testing

## Backend (Go)

36 test files covering search workflows, fuzzy near-miss candidates, edge cases, error recovery, memory/performance, file reading, security, log buffer management, IPC validation, file collection optimizations, find & replace, and .gitignore support:
- `gap_fixes_test.go` — **NEW**: security/perf hardening — `csvSafeCell` leading-space bypass (`" =2+2"`), `MaxResults` hard cap (10000), protected-directory subtree blocking (`/etc` → `/etc/ssh` without blocking `/etc-backup`), symlink file skip and symlink-dir non-traversal in `walkDirectoryTree`.
- `fuzzy_parity_test.go` — frontend/backend fuzzy-threshold parity (tripwire on `SLIDING_WINDOW_SIMILARITY_THRESHOLD` and `MAX_TEXT_LENGTH_FOR_FUZZY_SEARCH`).
- `export_test.go` — CSV export rendering: header structure, field mapping, empty context fields, special characters (commas, quotes), formula-injection guard including space-prefixed triggers.
- `helpers_test.go` — **NEW**: direct unit tests for pure helpers: `parseLogLine`/`parseLogEntryMessage`/`isNoisyMessage` (noise filtering for plain text + JSON), `matchesPattern` (path-component exact/glob/substring), `getFullExtension`/`matchExtension` (compound extensions), `isKnownTextExtension` (case-insensitivity, .wasm exclusion), `containsDotDotComponent` (Unix/Windows separators), `safeContextLinesBytes`/`bytesToStrings`/`searchContextLines` (boundary clamping), `validateAndSetDefaults` (defaults, protected dirs including subtrees, MaxResults cap), `rotateLogFileIfNeeded` (no-op/small/large/overwrite), `ReadFileLog`, `GetDirectoryContents`.
- `file_collection_test.go` — two-phase collection: known-text extension recognition, walk splits text/binary candidates, parallel binary probe filtering, absPath computation (absolute + relative directories), prefix-based traversal check (including sibling-dir edge case), symlink handling (file symlink skip, dir symlink non-traversal, `EvalSymlinks` on base), parallel probe scaling, and `TestGetKnownTextExtensions` which verifies the Wails binding that drives the frontend dropdown (sorted, no leading dot, excludes `.wasm`, round-trips with `isKnownTextExtension`).
- `data_validation_test.go` — input validation including the updated `very_large_values` case (MaxResults 100000 now correctly rejected by the 10000 cap).
- `app_test.go`, `binary_file_test.go`, `editor_detection_test.go`, `error_recovery_test.go`, `extended_app_test.go`, `improved_features_test.go`, `memory_performance_test.go`, `read_file_test.go`, `search_with_progress_test.go`, `security_test.go`.
- `replace_test.go` — **NEW**: `ReplaceInFiles` dry-run writes nothing, apply changes only matched lines atomically and preserves file mode, regex-mode and empty-query rejection, no-op replacement skip, case-sensitivity, multi-file and multi-directory replace, file-vanished-between-preview-and-apply skip, result determinism; `writeFileAtomic` rename-over-directory returns the error and cleans up its temp file.
- `collection_index_test.go` — **NEW**: cache-key canonicalization (sorting, filter sensitivity, slice-collision prevention), `computeCollectionFingerprint` stability + change detection, cache miss/populate/eviction/stale-fingerprint, `collectFilesToProcess` cached-result equality and cache bypass on different filters.
- `gitignore_test.go` — **NEW**: `loadGitignoreMatcher` (root `.gitignore`, `.git/info/exclude`, both, none), negation, `filterByGitignore`, and `collectFilesToProcess` with `RespectGitignore` on/off.
- `search_fuzzy_test.go` — fuzzy near-miss phase: threshold matches the frontend (max(1, floor(len*0.6))), best-window sliding scoring, integration tests for fuzzy off/on over a temp tree, regex-disables-fuzzy gating, exact-match precedence, and maxResults quota enforcement.
- `search_fuzzy_calibration_test.go` — fuzzy calibration: measured false-positive rate on 2000 random text lines (must stay under 5%; currently ~1%), single-edit near-miss sensitivity (substitution/transposition/deletion/insertion), unrelated-word rejection, near-miss discovery at scale (200 files), plus `BenchmarkFuzzyBestWindow` and `BenchmarkSearchFuzzyCandidates` in `search_bench_test.go`.
- `editor_launch_fixes_test.go` — editor-launch hardening: `appendPath` never aliases the shared `editorCatalog` args slices (concurrent-launch safety), `startAndReap` reaps short-lived children (no zombies), and `OpenInDefaultEditor` rejects traversal/nonexistent/empty paths before spawning a process.
- `search_results_sort_test.go` — deterministic result ordering (sorted by file path then line) and the cancelled-search guard (returns empty instead of partial "completed" results).
- `symbol_index_test.go` — persistent symbol index: cache hit on unchanged fingerprint, cache miss + rescan on file change, `ClearSymbolCache` binding, eviction at max capacity, concurrent read/write safety.
- `app_symbols_test.go` — symbol-search bindings (`GetAllSymbols`, `SearchSymbols`) with maxResults truncation.
- `editor_catalog_test.go` — table-driven editor catalog coverage.
- `search_context_test.go` — context-window helpers and the binary-probe buffer pool.
- `search_streaming_test.go` — line-by-line streaming path for large files.
- `multi_dir_test.go` — multi-directory search collects files from all roots; duplicate directories are deduplicated.
- `polling_noise_test.go` — noise filter consistency, log rotation memory leak, shutdown idempotency, shutdown done-channel signaling, re-init cleanup.
- `system_integration_fixes_test.go` — shell-metacharacter filename acceptance, null-byte/traversal rejection, table-driven editor bindings, snapshot-based editor count, `OpenInEditorByName` JetBrains file-extension routing.
- `perf_regression_test.go` — zero-allocation `isBinary`, buffer pool reuse, `bytes.Split` path, literal-mode regex compile, redundant binary check removal.
`symbols_test.go` covers the symbol-extraction engine (`GetAllSymbols`/`SearchSymbols` across Go/TS/JS/Vue, directory skipping, `maxResults` truncation). `ipc_validation_test.go` and `optimization_test.go` cover binding input validation and search-path optimizations. A separate `search_bench_test.go` holds benchmarks for the search pipeline (`go test -bench .`).

Notable coverage:
- **Editor detection**: `isEditorAvailable` with existing/non-existent commands, `countAvailableEditors` (including Neovim count, JetBrains derived flag), `GetAvailableEditors`, `GetEditorDetectionStatus`, `openInEditor` error handling, `OpenInEditorByName` dispatcher, `editorCatalog` completeness.
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

48 test files with 712 tests across components, composables, services, and utilities:

- `unit/components/` — `CodeModal.spec.ts` (30 tests including language-detection cases for `jsx`/`tsx`/`vue`/`toml`/`txt`, plus a match-counter-clamping test at the last match), `CodeModal.syntax.spec.ts` (33 tests), `LogViewer.spec.ts` (15 tests: collapse/expand, preview logs, placeholder, filtering, log parsing), `ProgressIndicator.spec.ts` (4 tests), `SearchForm.spec.ts` (16 tests), `SearchResults.spec.ts` (13 tests, including a test asserting InlineDiffView renders from raw content on the visible page), `ActionButtons.spec.ts` (8 tests), `DirectoryPicker.spec.ts` (7 tests), `EditorStatusDisplay.spec.ts` (7 tests), `EnhancedTreeItem.spec.ts` (24 tests: tree rendering, expansion, filtering, edge cases), `InlineDiffView.spec.ts` (27 tests), `PatternSelector.spec.ts` (8 tests), `QueryInput.spec.ts` (8 tests), `SearchHistorySidebar.spec.ts` (26 tests), `SearchOptions.spec.ts` (10 tests), `CodeSearch.integration.spec.ts` (14 tests).
- `unit/components/` — `TreeViewPanel.spec.ts` (7 tests: tree building from paths, folder/file ordering, counts, current-file highlight, file-click emission), `SearchSuggestions.spec.ts` (10 tests: rendering from localStorage, select/remove, outside-click and Escape close, listener cleanup on unmount), `EditorSelect.spec.ts` (9 tests: conditional rendering, emit), `SizeLimitOptions.spec.ts` (19 tests: defaults, clamping, update emit, disabled), `SymbolSearch.spec.ts` (14 tests: handleSymbolSearch/fetchAllSymbols/selectSymbol), `ToastNotification.spec.ts` (6 tests: renders, type classes, pause/resume, close).
- `unit/composables/` — `useLogStreaming.spec.ts` (19 tests: `parseLogEntry` variations — structured JSON, noise filtering for both `Skipping` and `Sending file`, plain text, missing content, level field name variants, timestamp formatting; Wails binding mock resolution and cursor behavior), `useSearch.spec.ts` (19 tests), `useSearch.additional.spec.ts` (12 tests), `useSearch.comprehensive.spec.ts` (25 tests), `useSearch.fixes.spec.ts` (11 tests: truncation check respects maxResults, non-array results coerced to [], immediate editor-detection fetch, listener cleanup on completed/error/unmount, cancel-during-flight does not repopulate results), `useToast.spec.ts` (17 tests: add/remove, pause/resume, idempotent operations, concurrent staggered durations), `useFilePreview.spec.ts` (11 tests: openFile, closePreview, same-file content retention, different-file reload, options passthrough), `useSelectionManager.spec.ts` (12 tests: reactive selectedCount/allVisibleSelected, toggleSelected, toggleSelectAll, clearSelection, copy/export selected subset + all-fallback), `useTheme.spec.ts` (28 tests: readInitialTheme/applyTheme/setTheme/toggleTheme/isDark), `useEditorDetection.spec.ts` (22 tests: default factories, event subscription, pull-based status), `useKeyboardShortcuts.spec.ts` (26 tests: Ctrl+K/Cmd+K, Ctrl+Enter, ESC, field suppression, mount/unmount), `useMatchNavigation.spec.ts` (22 tests: totalMatches, next/prev wrap, watch reset), `useCodeHighlighting.spec.ts` (20 tests: renderPlainText, detectLanguage, loadAndHighlight), `useReplace.spec.ts` (7 tests: preview calls the binding with apply=false, apply calls it with apply=true then re-runs search, regex-mode guard, apply-without-preview no-op, zero-change preview, missing-query guard), `useSymbolSearch.spec.ts` (1 test), `searchProgress.spec.ts` (9 tests).
- `unit/utils/` — `searchUiUtils.spec.ts` (14 tests), `fuzzyMatch.spec.ts` (6 tests: findFuzzyMatches similarity thresholds, case-insensitivity, whole-text scan, perf bail-out), `localStorageUtils.spec.ts` (12 tests), `diffUtils.spec.ts` (29 tests: match-range finding, diff-segment building, long-line truncation, HTML rendering, XSS sanitization, case-sensitivity, greedy matching edge cases), `errorUtils.spec.ts` (17 tests: toErrorMessage/asRecord), `fileUtils.spec.ts` (14 tests: formatFilePath/truncatePath), `toastUtils.spec.ts` (9 tests: clipboard/open-file-location/open-in-editor), `htmlUtils.spec.ts` (5 tests), `regexUtils.spec.ts` (5 tests).
- `unit/services/` — `syntaxHighlightingService.spec.ts` (25 tests: detectLanguage/highlightCode/loadHighlightJs).

**Test infrastructure** (`frontend/tests/`):
- `setup.ts` — preloads highlight.js, mocks `IntersectionObserver`, `scrollIntoView`, clipboard fallback.
- `__mocks__/wailsjs/` — fake Wails binding modules so component tests run without a real bridge. Includes `GetKnownTextExtensions` returning a representative subset of the known-text extension list, plus `GetInitialLogs` and `GetNewLogs` for the log-streaming composable.
- `fixtures/` — shared test data (e.g. `editorAvailability.ts` with all 22 editor fields).

```bash
cd frontend
npm test               # run once
npx vitest             # watch mode
```

## End-to-end (Playwright)

`frontend/playwright-tests/` drives the real UX flows in a browser against a mocked Wails backend (`src/mocks/wailsMock.ts`, installed by `main.ts` when `VITE_WAILS_MOCK` is set). It uses the system Chrome (`channel: 'chrome'`) and auto-starts vite with the mock, so no Go process is needed.

41 flow tests across seven specs:
- `flows.spec.ts` (7) — startup renders the UI (guards the "black screen" regression), Search Code populates results, an empty query keeps the button disabled, the file-preview modal opens with content, symbol search returns matches for a directory (and prompts to select one when absent), and the case-sensitive option is honored.
- `find-replace.spec.ts` (2) — the replace row is hidden under regex mode, and a preview-then-apply flow shows old→new diffs, applies the change, and re-searches to confirm the matches are gone.
- `filetree-suggestions.spec.ts` (2) — the File Explorer tree in the preview modal lists all result files and opening one loads it (title + content + toggle state), and the recent-search suggestions dropdown appears on focus, selects a query, and closes on outside-click and Escape.
- `enhancements.spec.ts` (12) — symbol click opens code preview modal, symbol click shows file content, diff markers (+/-) render on search results, batch export buttons (CSV/JSON) are present, multi-select checkboxes toggle and show count, select-all checkbox selects all visible results, extra directories textarea is present and editable, log viewer search input and auto-scroll toggle are present, "Load All Symbols" shows progress and results, and three symbol-navigation line-jump flows (first click flashes the target line, re-navigation to a second symbol in the SAME file re-jumps, and navigation to a symbol in a DIFFERENT file loads it and jumps).
- `search-options.spec.ts` (6) — regex search matches by pattern (not substring), an invalid-substring regex yields no matches, maxResults caps results and flags truncation, the theme toggle flips `data-theme` and persists to localStorage across reload, copy-line writes the match content to the clipboard, and the preview-modal footer exposes Jump to Line / Show in Folder / Copy actions.
- `advanced-search.spec.ts` (5) — search is scoped to the selected directory (other roots don't leak), pagination splits results into pages of 10 with working Next, the preview modal mounts match-navigation controls for large (>50-line) files and jump-to-line flashes the target, multi-directory search merges results from an extra root, and an exclude pattern drops matching files. Relies on the extended mock FS (`/mock/big/huge.go`, `/mock/lib/extra.go`).
- `fuzzy-search.spec.ts` (7) — the Fuzzy Search checkbox renders and is unchecked by default; fuzzy off returns only the 4 exact `hello` matches with no badges; fuzzy on appends near-miss lines (hxllo/hxllx in `util.go`) beyond the exact hits; near-misses render the `.fuzzy-badge`; regex mode ignores the fuzzy flag; the typo query `helo` returns zero results with fuzzy off but six near-misses with fuzzy on; and long queries raise the threshold so garbage candidates are rejected.

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
