# Testing

## Backend (Go)

39 test files covering search workflows, streamed result batches, failed-file reporting, fuzzy near-miss candidates, nested `.gitignore` resolution, edge cases, error recovery, memory/performance, file reading, security, log buffer management, IPC validation, file collection optimizations, and find & replace:
- `gap_fixes_test.go` — **NEW**: security/perf hardening — `csvSafeCell` leading-space bypass (`" =2+2"`), `MaxResults` hard cap (10000), protected-directory subtree blocking (`/etc` → `/etc/ssh` without blocking `/etc-backup`), query-length cap (2000 chars), symlink file skip and symlink-dir non-traversal in `walkDirectoryTree`.
- `fuzzy_parity_test.go` — frontend/backend fuzzy-threshold parity (tripwire on `SLIDING_WINDOW_SIMILARITY_THRESHOLD` and `MAX_TEXT_LENGTH_FOR_FUZZY_SEARCH`).
- `export_test.go` — CSV export rendering: header structure, field mapping, empty context fields, special characters (commas, quotes), formula-injection guard including space-prefixed triggers.
- `helpers_test.go` — **NEW**: direct unit tests for pure helpers: `parseLogLine`/`parseLogEntryMessage`/`isNoisyMessage` (noise filtering for plain text + JSON), `matchesPattern` (path-component exact/glob/substring), `getFullExtension`/`matchExtension` (compound extensions), `isKnownTextExtension` (case-insensitivity, .wasm exclusion), `containsDotDotComponent` (Unix/Windows separators), `safeContextLinesBytes`/`bytesToStrings`/`searchContextLines` (boundary clamping), `validateAndSetDefaults` (defaults, protected dirs including subtrees, MaxResults cap), `rotateLogFileIfNeeded` (no-op/small/large/overwrite), `ReadFileLog`, `GetDirectoryContents`.
- `file_collection_test.go` — two-phase collection: known-text extension recognition, walk splits text/binary candidates, parallel binary probe filtering, absPath computation (absolute + relative directories), prefix-based traversal check (including sibling-dir edge case), symlink handling (file symlink skip, dir symlink non-traversal, `EvalSymlinks` on base), parallel probe scaling, and `TestGetKnownTextExtensions` which verifies the Wails binding that drives the frontend dropdown (sorted, no leading dot, excludes `.wasm`, round-trips with `isKnownTextExtension`).
- `data_validation_test.go` — input validation including the updated `very_large_values` case (MaxResults 100000 now correctly rejected by the 10000 cap).
- `app_test.go`, `binary_file_test.go`, `editor_detection_test.go`, `error_recovery_test.go`, `extended_app_test.go`, `improved_features_test.go`, `memory_performance_test.go`, `read_file_test.go`, `search_with_progress_test.go`, `security_test.go`.
- `replace_test.go` — **NEW**: `ReplaceInFiles` dry-run writes nothing, apply changes only matched lines atomically and preserves file mode, regex-mode and empty-query rejection, no-op replacement skip, case-sensitivity, multi-file and multi-directory replace, file-vanished-between-preview-and-apply skip, result determinism; `writeFileAtomic` rename-over-directory returns the error and cleans up its temp file.
- `collection_index_test.go` — **NEW**: cache-key canonicalization (sorting, filter sensitivity, slice-collision prevention), `computeCollectionFingerprint` stability + change detection, cache miss/populate/eviction/stale-fingerprint, `collectFilesToProcess` cached-result equality, cache bypass on different filters, and `get` returning a copy (mutating the result cannot poison the cache).
- `gitignore_test.go` — **NEW**: `loadGitignoreMatcher` (root `.gitignore`, `.git/info/exclude`, both, none), negation, `filterByGitignore`, and `collectFilesToProcess` with `RespectGitignore` on/off.
- `gitignore_nested_test.go` — **NEW**: nested `.gitignore` via `ignoreStack` — a deeper ignore file's rules override a shallower one, patterns resolve relative to their own `.gitignore`'s directory (not the search root), a plain directory rule prunes the whole subtree, pruning is *abandoned* when a contents-rule negation (`build/*` + `!build/keep.txt`) re-includes a file, `RespectGitignore` off reads no ignore files at all (the byte-identical-to-before guarantee), the memo map proves each ignore file is read once per directory rather than once per file, and editing a nested `.gitignore` changes `computeCollectionFingerprint` so a stale collection cannot be served.
- `search_fuzzy_test.go` — fuzzy near-miss phase: threshold matches the frontend (max(1, floor(len*0.6))), best-window sliding scoring, integration tests for fuzzy off/on over a temp tree, regex-disables-fuzzy gating, exact-match precedence, and maxResults quota enforcement.
- `search_fuzzy_calibration_test.go` — fuzzy calibration: measured false-positive rate on 2000 random text lines (must stay under 5%; currently ~1%), single-edit near-miss sensitivity (substitution/transposition/deletion/insertion), unrelated-word rejection, near-miss discovery at scale (200 files), plus `BenchmarkFuzzyBestWindow` and `BenchmarkSearchFuzzyCandidates` in `search_bench_test.go`.
- `editor_launch_fixes_test.go` — editor-launch hardening: `appendPath` never aliases the shared `editorCatalog` args slices (concurrent-launch safety), `startAndReap` reaps short-lived children (no zombies), and `OpenInDefaultEditor` rejects traversal/nonexistent/empty paths before spawning a process.
- `search_results_sort_test.go` — deterministic result ordering (sorted by file path then line) and the cancelled-search guard (returns empty instead of partial "completed" results).
- `search_streaming_batch_test.go` — **NEW**: `resultBatcher` flushes at `resultBatchSize` with a monotonic `seq` starting at 1, an empty terminal flush emits nothing and burns no sequence number (otherwise the frontend would treat `seq` 1 as delivered and drop the next search's first batch), a flushed batch never shares its backing array with the following one, `SearchState.recordFailure` caps the listed sample at `maxFailedPathsReported` while `failedFiles` stays exact and `snapshotFailedPaths` returns a copy, an empty path still counts but is not listed, and the real pipeline run against an unreadable file reports it on the terminal `completed` payload as both a count and a named path.
- `app_shared_test.go` — **NEW**: the path-validation trust boundary every editor launch and folder reveal crosses, previously covered only through build-tagged platform wrappers that each run on one OS. `sanitizePath` rejects traversal in both the original *and* the `filepath.Clean`-ed path (`/tmp/../etc/passwd` cleans to `/etc/passwd`, which would otherwise hide the intent), dots inside filenames are still accepted, `validatePathForEditor` requires the file to exist, `validatePathForShowInFolder` resolves to the parent directory, `lookUpEditor` returns the resolved absolute path (the TOCTOU-closing contract), `appendPath` never mutates the shared `editorCatalog` args slices; plus `isSymbolSupportedExtension` across all ten languages and `symbolCacheKey` normalization.
- `symbol_index_test.go` — persistent symbol index: cache hit on unchanged fingerprint, cache miss + rescan on file change, `ClearSymbolCache` binding, eviction at max capacity, concurrent read/write safety, and `get` returning a copy (mutating the result cannot poison the cache).
- `app_symbols_test.go` — symbol-search bindings (`GetAllSymbols`, `SearchSymbols`) with maxResults truncation.
- `editor_catalog_test.go` — table-driven editor catalog coverage.
- `search_context_test.go` — context-window helpers and the binary-probe buffer pool.
- `search_streaming_test.go` — line-by-line streaming path for large files.
- `multi_dir_test.go` — multi-directory search collects files from all roots; duplicate directories are deduplicated.
- `polling_noise_test.go` — noise filter consistency, log rotation memory leak, shutdown idempotency, shutdown done-channel signaling, re-init cleanup.
- `system_integration_fixes_test.go` — shell-metacharacter filename acceptance, null-byte/traversal rejection, table-driven editor bindings, snapshot-based editor count, `OpenInEditorByName` JetBrains file-extension routing.
- `perf_regression_test.go` — zero-allocation `isBinary`, buffer pool reuse, `bytes.Split` path, literal-mode regex compile, redundant binary check removal.
`symbols_test.go` covers the symbol-extraction engine (`GetAllSymbols`/`SearchSymbols`, directory skipping, `maxResults` truncation) with real fixture files in Go, TypeScript, and Vue. `TestGetPatternsForExtension` (`symbols_test.go:334-355`) now iterates `symbolSupportedExtensions` instead of a hardcoded list, so adding a language cannot leave the slice and the `getPatternsForExtension` switch disagreeing silently; its negative case is `.txt` (text, so it reaches collection, but with no symbol grammar) — it used to be `.py`, which broke the moment Python became supported. `ipc_validation_test.go` and `optimization_test.go` cover binding input validation and search-path optimizations. A separate `search_bench_test.go` holds benchmarks for the search pipeline (`go test -bench .`).

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
# What run_tests.sh:16 runs; CI (.github/workflows/build.yml:125) adds -v.
# -covermode=atomic is required by -race: the default `set` mode is not
# race-safe, so the detector and a non-atomic counter cannot coexist.
# -race is several times slower, hence -timeout 600s instead of the default.
go test -race -covermode=atomic -coverprofile=coverage.out -timeout 600s ./...
go tool cover -func=coverage.out    # total — this is the number the CI gate reads
go tool cover -html=coverage.out    # per-line view
go test -bench . -benchmem          # run search benchmarks
```

> **Note**: The HTTP polling server (`polling_server_test.go`) has been removed — the frontend now consumes log entries via Wails IPC bindings (`GetInitialLogs`, `GetNewLogs`), not HTTP polling. The `polling_noise_test.go` file covers the buffer/tail core with updated tests that don't depend on an HTTP server.

## Frontend (Vitest)

48 test files with 714 tests across components, composables, services, and utilities:

Per-spec test counts used to be listed here and are deliberately gone. They rot on
every commit, and they had: seven of the numbers below disagreed with the ones
`docs/TESTING_GAPS.md` quoted for the same specs (useTheme, useMatchNavigation,
useCodeHighlighting, fileUtils, toastUtils, SymbolSearch, InlineDiffView), and
`useSelectionManager` was recorded as 12 in both files while the spec held 13. The
suite total is cheap to keep right; what each spec defends is the durable part.

- `unit/components/` — `CodeModal.spec.ts` (language detection for `jsx`/`tsx`/`vue`/`toml`/`txt`, match-counter clamping at the last match), `CodeModal.syntax.spec.ts`, `LogViewer.spec.ts` (collapse/expand, preview logs, placeholder, filtering, log parsing), `ProgressIndicator.spec.ts`, `SearchForm.spec.ts`, `SearchResults.spec.ts` (including InlineDiffView rendering from raw content on the visible page), `ActionButtons.spec.ts`, `DirectoryPicker.spec.ts`, `EditorStatusDisplay.spec.ts`, `EnhancedTreeItem.spec.ts` (tree rendering, expansion, filtering, edge cases), `InlineDiffView.spec.ts`, `PatternSelector.spec.ts`, `QueryInput.spec.ts`, `SearchHistorySidebar.spec.ts`, `SearchOptions.spec.ts`, `CodeSearch.integration.spec.ts`.
- `unit/components/` — `TreeViewPanel.spec.ts` (tree building from paths, folder/file ordering, counts, current-file highlight, file-click emission), `SearchSuggestions.spec.ts` (rendering from localStorage, select/remove, outside-click and Escape close, listener cleanup on unmount), `EditorSelect.spec.ts` (conditional rendering, emit), `SizeLimitOptions.spec.ts` (defaults, clamping, update emit, disabled), `SymbolSearch.spec.ts` (handleSymbolSearch/fetchAllSymbols/selectSymbol), `ToastNotification.spec.ts` (renders, type classes, pause/resume, close).
- `unit/composables/` — `useLogStreaming.spec.ts` (`parseLogEntry` variations — structured JSON, noise filtering for both `Skipping` and `Sending file`, plain text, missing content, level field name variants, timestamp formatting; Wails binding mock resolution and cursor behavior), `useSearch.spec.ts` (including the 2000-char query cap), `useSearch.additional.spec.ts`, `useSearch.comprehensive.spec.ts`, `useSearch.fixes.spec.ts` (truncation check respects maxResults, non-array results coerced to `[]`, immediate editor-detection fetch, listener cleanup on completed/error/unmount, cancel-during-flight does not repopulate results), `useToast.spec.ts` (add/remove, pause/resume, idempotent operations, concurrent staggered durations), `useFilePreview.spec.ts` (openFile, closePreview, same-file content retention, different-file reload, options passthrough), `useSelectionManager.spec.ts` (reactive `selectedCount`/`allVisibleSelected`, toggles, `clearSelection`, copy/export of the selected subset with all-results fallback, and the export-format regression — the requested format is threaded through instead of hardcoded `"csv"`), `useTheme.spec.ts`, `useEditorDetection.spec.ts` (default factories, event subscription, pull-based status race fix), `useKeyboardShortcuts.spec.ts`, `useMatchNavigation.spec.ts`, `useCodeHighlighting.spec.ts`, `useReplace.spec.ts`, `useSymbolSearch.spec.ts`, `searchProgress.spec.ts`.
  `useSearch` now awaits `ValidateDirectory` before searching, so `useSearch.spec.ts` and `useSearch.fixes.spec.ts` flush microtasks (`await Promise.resolve()`) before asserting on `SearchWithProgress` or on listener registration — without that, the assertions race the guard they exist to verify.
- `unit/utils/` — `searchUiUtils.spec.ts`, `fuzzyMatch.spec.ts` (findFuzzyMatches similarity thresholds, case-insensitivity, whole-text scan, perf bail-out), `localStorageUtils.spec.ts`, `diffUtils.spec.ts` (match-range finding, diff-segment building, long-line truncation, HTML rendering, escapeHtml-before-DOMPurify escaping, case-sensitivity, greedy matching edge cases), `errorUtils.spec.ts` (toErrorMessage/asRecord), `fileUtils.spec.ts` (formatFilePath/truncatePath), `toastUtils.spec.ts` (clipboard/open-file-location/open-in-editor), `htmlUtils.spec.ts`, `regexUtils.spec.ts`.
- `unit/services/` — `syntaxHighlightingService.spec.ts` (detectLanguage/highlightCode/loadHighlightJs).

**Test infrastructure** (`frontend/tests/`):
- `setup.ts` — preloads highlight.js, mocks `IntersectionObserver`, `scrollIntoView`, clipboard fallback.
- `__mocks__/wailsjs/` — fake Wails binding modules so component tests run without a real bridge. Includes `GetKnownTextExtensions` returning a representative subset of the known-text extension list, `GetInitialLogs`/`GetNewLogs` for the log-streaming composable, and `ValidateDirectory` defaulting to `true` (every search in `useSearch` gates on it, so an unset mock would reject each spec's search before it reached the backend). `GetAllSymbols` and `SearchSymbols` are declared here **once** — every symbol spec used to re-declare them in a local `vi.mock`, so nothing detected the drift. The `SearchCode` export is gone: no such binding exists on the Go side, so the mock was advertising a method the real bridge would never have.
- `fixtures/` — shared test data (e.g. `editorAvailability.ts` with all 22 editor fields).

```bash
cd frontend
npm test               # run once
npx vitest             # watch mode
```

## End-to-end (Playwright)

`frontend/playwright-tests/` drives the real UX flows in a browser against a mocked Wails backend (`src/mocks/wailsMock.ts`, installed by `main.ts` when `VITE_WAILS_MOCK` is set). It uses the system Chrome (`channel: 'chrome'`) and auto-starts vite with the mock, so no Go process is needed. The mock emits `search-results` batches (two batches, monotonic `seq`) and implements `ValidateDirectory`, so the E2E run exercises the streaming append path and the pre-search validation guard rather than only the resolved-value path. `frontend/playwright.config.js` sets `retries: process.env.CI ? 2 : 0` (`:29`) — CI absorbs genuine flake, locally a failure fails now — and `reuseExistingServer: !process.env.CI` (`:36`), so CI always starts a fresh server and a stale or foreign one can never serve the tests.

41 flow tests across seven specs:
- `flows.spec.ts` (7) — startup renders the UI (guards the "black screen" regression), Search Code populates results, an empty query keeps the button disabled, the file-preview modal opens with content, symbol search returns matches for a directory (and prompts to select one when absent), and the case-sensitive option is honored.
- `find-replace.spec.ts` (2) — the replace row is hidden under regex mode, and a preview-then-apply flow shows old→new diffs, applies the change, and re-searches to confirm the matches are gone.
- `filetree-suggestions.spec.ts` (2) — the File Explorer tree in the preview modal lists all result files and opening one loads it (title + content + toggle state), and the recent-search suggestions dropdown appears on focus, selects a query, and closes on outside-click and Escape.
- `enhancements.spec.ts` (12) — symbol click opens code preview modal, symbol click shows file content, diff markers (+/-) render on search results, batch export buttons (CSV/JSON) are present, multi-select checkboxes toggle and show count, select-all checkbox selects all visible results, extra directories textarea is present and editable, log viewer search input and auto-scroll toggle are present, "Load All Symbols" shows progress and results, and three symbol-navigation line-jump flows (first click flashes the target line, re-navigation to a second symbol in the SAME file re-jumps, and navigation to a symbol in a DIFFERENT file loads it and jumps).
- `search-options.spec.ts` (6) — regex search matches by pattern (not substring), an invalid-substring regex yields no matches, maxResults caps results and flags truncation, the theme toggle flips `data-theme` and persists to localStorage across reload, copy-line writes the match content to the clipboard, and the preview-modal footer exposes Jump to Line / Show in Folder / Copy. The footer test targets `/mock/big` (a 60-line file) rather than the 3-file `/mock/project` fixture: `ModalFooter.vue:21` renders "Jump to Line" only under `canJumpToLine`, which `CodeModal.vue:159-160` computes as `totalLines > LINE_JUMP_MIN_LINES` (50) — the same threshold that mounts `MatchNavigationControls` and its `.line-input`. On short files the whole file is already on screen, so the button is correctly hidden rather than offered as a dead control. Clicking it focuses that inline input, which replaced a native `prompt()` (unstyled and untestable in a WebView).
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

## Coverage

Go statement coverage is **80.0%** (`go tool cover -func=coverage.out` total, measured under `-race -covermode=atomic`). CI gates at 80% (`.github/workflows/build.yml:127-134`), so the suite **passes** — with no margin. What breaks that gate in practice is a new file landing untested, not a regression inside an existing one. The profile is uploaded as the `go-coverage` artifact (`:136-141`).

`frontend/vitest.config.ts:43-55` declares coverage thresholds (lines/functions/statements 80, branches 70; branches sits lower because Vue SFC render branches inflate the denominator). `ponytail:` those thresholds are **inert** — `npm test` is `vitest run` with no `--coverage` (`frontend/package.json`), so nothing computes coverage to compare against them, and the frontend number is currently unmeasured rather than merely unenforced. Upgrade: add a `test:coverage` script running `vitest run --coverage` and call it from the CI frontend step (or pass `--coverage` there). Until then the thresholds are documentation, not a gate.

## CI

`.github/workflows/build.yml` has three jobs: `test`, `build` (`needs: test`), and `release` (`needs: build`). Triggers are a push to `main`, a push of a `v*` tag, and a pull request against `main` (`:3-8`).

**3-OS matrix** (`:19-23`) — `[ubuntu-latest, windows-latest, macos-latest]` with `fail-fast: false`, `runs-on: ${{ matrix.os }}`. This is the first time `appDarwin.go` (`//go:build darwin`) and `appWindows.go` (`//go:build windows`) compile in CI at all: a Linux-only build never sees them, so they shipped without ever being compiled, let alone tested. `fail-fast: false` keeps a Windows-only path bug from cancelling the run before the other platforms report.

Exactly three steps are ungated and so run on all three OSes: Checkout, Setup Go, and `go test -v -race -covermode=atomic -coverprofile=coverage.out -timeout 600s ./...` (`:123-125`). Everything else carries `if: matrix.os == 'ubuntu-latest'` — the frontend toolchain, gofmt/vet/lint, coverage gate, unit tests, E2E, and `vue-tsc` are OS-independent, and running them three times would triple CI minutes for identical results.

**Non-Linux `frontend/dist` stub** (`:79-84`, gated `if: matrix.os != 'ubuntu-latest'`) — `main.go` carries `//go:embed all:frontend/dist` and `frontend/dist` is gitignored, so the Go package does not compile on a fresh checkout without an embed target. Windows and macOS only need the Go tests, so they write a one-line stub `index.html` instead of installing Node and building the real frontend.

**Pinned tool versions** — `golangci-lint` v2.13.2 (`:98-102`), `staticcheck@v0.7.0` (`:107-111`), `govulncheck@v1.1.4` (`:113-117`), `garble@v0.14.2` (`:184-185`), `wails@v2.14.0` (`:195-198`, tracking `github.com/wailsapp/wails/v2` in `go.mod`). These were `@latest`, which made builds non-reproducible: a new upstream release could fail CI with no repo change, and two runs of the same commit could disagree.

**Tag-triggered release** (`:236-262`) — `if: startsWith(github.ref, 'refs/tags/v')` with `permissions: contents: write`. It downloads every artifact the `build` job uploaded (no `name:`, so each lands in its own `dist/<artifact-name>/`) and publishes them with `gh release create "${{ github.ref_name }}" --title "${{ github.ref_name }}" --generate-notes dist/*/*`. A push to `main` never creates a release.

`.golangci.yml` enumerates its linters explicitly — `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `bodyclose`, `misspell` — so a golangci-lint upgrade cannot silently change the effective set. `run_tests.sh` mirrors CI: the same `-race -covermode=atomic -coverprofile -timeout 600s` Go invocation, and `npx vue-tsc --noEmit` rather than plain `tsc`, which skips `.vue` SFCs and so let a local pass hide template type errors CI caught.
