# Testing Gaps & Coverage Report

## Current Test Status

### Frontend Tests (679 passing across 45 spec files)
| Component/Test File | Tests | Coverage | Status |
|---|---|---|---|
| InlineDiffView | 18 | Full component logic | ✅ Complete |
| SearchHistorySidebar | 26 | All interactions | ✅ Complete |
| SearchResults | 10 | Pagination, rendering | ✅ Complete |
| useSearch composable | 66 | Core search, fuzzy mode | ✅ Complete |
| CodeSearch integration | 13 | UI flow, sidebar | ✅ Complete |
| searchUiUtils | 42 | highlightMatch, memoization edge cases | ✅ Complete |
| fuzzyMatch | 6 | findFuzzyMatches: similarity thresholds, case-insensitivity, whole-text scan, perf bail-out | ✅ Complete |
| localStorageUtils | 11 | Save/load round-trip, quota/disabled storage, remove, key stability | ✅ Complete |
| TreeViewPanel | 7 | Tree building, ordering, file-click | ✅ Complete |
| SearchSuggestions | 10 | Rendering, select/remove, close-on-outside-click | ✅ Complete |
| errorUtils | 17 | toErrorMessage (Error/string/object/null/fallback), asRecord (object/array/null/primitives) | ✅ Complete |
| fileUtils | 9 | formatFilePath (empty/short/long/multi-part), truncatePath (default/custom maxLength) | ✅ Complete |
| toastUtils | 12 | copyToClipboardWithToast (empty/clipboard API/fallback), openFileLocationWithToast (invalid/success/backslash/error), openInEditorWithToast | ✅ Complete |
| useTheme | 25 | readInitialTheme (localStorage/OS fallback/throw), applyTheme, setTheme (persist/failure), toggleTheme, isDark | ✅ Complete |
| useEditorDetection | 22 | makeDefault factories, subscribeToEditorDetectionEvents (start/progress/complete/dedup), startEditorDetection (pull-based race fix) | ✅ Complete |
| useKeyboardShortcuts | 26 | Ctrl+K/Cmd+K, Ctrl+Enter, ESC (typing vs not), field suppression, handler optional, mount/unmount | ✅ Complete |
| useMatchNavigation | 21 | totalMatches (empty/regex-escape/invalid), goToNextMatch (wrap), goToPreviousMatch (wrap), watch reset | ✅ Complete |
| useCodeHighlighting | 18 | renderPlainText (empty/escape/mark/line-numbers/truncate), detectLanguage, loadAndHighlight (ready/fallback/reject) | ✅ Complete |
| EditorSelect | 9 | Renders available editors, placeholder + default always present, emits editorSelect on change | ✅ Complete |
| SizeLimitOptions | 19 | Default init, prop propagation, contextLines clamping, update emit payload, disabled state | ✅ Complete |
| SymbolSearch | 12 | handleSymbolSearch (empty/no-dir/success/error), fetchAllSymbols (cache/dir/progress), selectSymbol, recentlySeenSymbols | ✅ Complete |
| ToastNotification | 6 | Renders toasts, type classes, pause/resume on hover, close button, progress bar | ✅ Complete |
| syntaxHighlightingService | 27 | isHighlightJsLoaded, loadHighlightJs (idempotent), detectLanguage (extensions/case/unknown), highlightCode (empty/large/truncate/mark/fallback), getHighlightJs | ✅ Complete |
| appInitializationService | 3 | initializeAppServices (idempotent loadHighlightJs, no throw) | ✅ Complete |
| useSelectionManager | 11 | reactive selectedCount/allVisibleSelected, toggleSelected, toggleSelectAll, clearSelection, copy/export selected subset + all-fallback | ✅ Complete |

### Backend Tests (27 Go test files)
| File | Focus Area | Coverage |
|---|---|---|
| helpers_test.go | parseLogLine/parseLogEntryMessage/isNoisyMessage, matchesPattern, getFullExtension/matchExtension, isKnownTextExtension, containsDotDotComponent, safeContextLinesBytes/bytesToStrings/searchContextLines, validateAndSetDefaults, rotateLogFileIfNeeded, ReadFileLog, GetDirectoryContents | ✅ Complete |
| ipc_validation_test.go | JSON serialization | ✅ Complete |
| app_test.go | App lifecycle, IsAppReady, GetInitialLogs/GetNewLogs | ✅ Complete |
| search_with_progress_test.go | Search engine | ✅ Complete |
| perf_regression_test.go | Performance baselines | ✅ Complete |
| optimization_test.go | Optimization paths | ✅ Complete |
| symbols_test.go | Symbol Search extraction | ✅ Complete |
| export_test.go | CSV rendering, empty results rejection, binding requires context | ✅ Complete |
| system_integration_fixes_test.go | Editor bindings, JetBrains routing, snapshot count, path traversal, null bytes | ✅ Complete |
| search_fuzzy_test.go | Fuzzy near-miss phase: threshold (60% positional), best-window scoring, regex gating, exact-match exclusion, quota enforcement | ✅ Complete |
| search_fuzzy_calibration_test.go | Fuzzy calibration: measured false-positive rate on 2000 random lines (<5% ceiling, ~1% measured), single-edit near-miss sensitivity, unrelated-word rejection, near-miss discovery at scale (200 files) | ✅ Complete |
| editor_launch_fixes_test.go | Editor-launch hardening: appendPath aliasing safety, startAndReap zombie reaping, OpenInDefaultEditor path validation | ✅ Complete |

### End-to-End Tests (Playwright, 39 tests in 6 spec files)
`playwright-tests/` drives the app against an in-browser Wails mock backend
(`src/mocks/wailsMock.ts`, enabled via `VITE_WAILS_MOCK=1`): `flows.spec.ts`,
`filetree-suggestions.spec.ts`, `enhancements.spec.ts`, `search-options.spec.ts`,
`advanced-search.spec.ts`, and `fuzzy-search.spec.ts`.
Run with `npm run test:e2e` (opt-in in `run_tests.sh` via `RUN_E2E=1`).
| Flow | Coverage | Status |
|---|---|---|
| App startup renders | Initial UI mount | ✅ Complete |
| Search → results populate | Query submit + result rendering | ✅ Complete |
| Empty query disables button | Button guard on empty input | ✅ Complete |
| File-preview modal opens with content | CodeModal visibility + content | ✅ Complete |
| File Explorer tree | Result files listed; opening a file loads it | ✅ Complete |
| Suggestions dropdown | Show on focus, select, close on outside-click/Escape | ✅ Complete |
| Symbol search with a directory | Directory-scoped symbol lookup | ✅ Complete |
| Symbol search without a directory | Undirected symbol lookup | ✅ Complete |
| Case-sensitivity | Case-sensitive matching | ✅ Complete |
| Symbol click → line jump | First click flashes target line in preview | ✅ Complete |
| Symbol re-navigation (same file) | Second symbol in same file re-jumps to new line | ✅ Complete |
| Symbol navigation (different file) | Loads a new file via ReadFile and jumps | ✅ Complete |
| Multi-select + select-all | Checkbox toggles reactive count; select-all checks page | ✅ Complete |
| Regex search | Pattern match (not substring); no-match path | ✅ Complete |
| maxResults truncation | Result cap + truncation flag | ✅ Complete |
| Theme toggle | Flips data-theme, persists to localStorage across reload | ✅ Complete |
| Copy line | Writes match content to clipboard | ✅ Complete |
| Modal footer actions | Jump to Line / Show in Folder / Copy present + safe | ✅ Complete |
| Directory scoping | Results limited to selected root | ✅ Complete |
| Pagination | 10/page split, Next advances to remainder | ✅ Complete |
| Match navigation (large file) | Nav controls mount >50 lines; jump-to-line flashes | ✅ Complete |
| Multi-directory search | Extra root merges into results | ✅ Complete |
| Exclude pattern | Drops matching files from results | ✅ Complete |
| Fuzzy off — exact only | Only substring hits returned; no badges | ✅ Complete |
| Fuzzy on — near-misses appended | Near-miss lines beyond exact matches; `.fuzzy-badge` rendered | ✅ Complete |
| Regex disables fuzzy | Regex mode ignores the fuzzy flag | ✅ Complete |
| Typo query recovery | "helo" finds 6 near-misses with fuzzy on, zero with fuzzy off | ✅ Complete |
| Long-query threshold | Raised threshold rejects garbage candidates from long queries | ✅ Complete |

---

## Recently Closed Gaps

### ✅ Backend Helper Unit Tests (NEW)
All previously-untested pure helpers now have direct unit coverage in
`helpers_test.go`:
- `parseLogLine` / `parseLogEntryMessage` / `isNoisyMessage` — noise filtering
  for plain text and JSON log lines (string, object, other types, empty)
- `matchesPattern` — path-component exact match, glob match, substring
  non-match, empty pattern edge case
- `getFullExtension` / `matchExtension` — compound extensions (.min.js,
  .tar.gz), case-insensitivity, empty request
- `isKnownTextExtension` — case-insensitivity, unknown extension safe default,
  .wasm explicit exclusion
- `containsDotDotComponent` — Unix/Windows separators, non-component dots
  (foo..bar.txt), empty path
- `safeContextLinesBytes` / `bytesToStrings` / `searchContextLines` — boundary
  clamping, empty ranges, nil input, default and max clamping
- `validateAndSetDefaults` — defaults for zero values, explicit preservation,
  empty/non-existent/protected directory rejection
- `rotateLogFileIfNeeded` — no-op for missing/small file, rotation on
  exceed, overwrite of previous .1 rotation
- `ReadFileLog` — path resolution, different names
- `GetDirectoryContents` — subdirectory listing, hidden dir skipping, file
  exclusion, non-existent path returns empty

### ✅ ExportSearchResults Binding Tests (NEW)
`export_test.go` now covers the binding's pre-dialog contract: empty/nil
results rejection, and the Wails context requirement (skipped —
SaveFileDialog panics with nil ctx, a Wails runtime behavior).

### ✅ Frontend Utils — errorUtils, fileUtils, toastUtils (NEW)
Three previously-untested pure utility modules now have dedicated specs:
- `errorUtils.spec.ts` (17 tests): toErrorMessage for Error/string/object/
  null/undefined/number/non-string-message, custom fallback; asRecord for
  objects/arrays/null/primitives
- `fileUtils.spec.ts` (9 tests): formatFilePath truncation rules, truncatePath
  with default and custom maxLength
- `toastUtils.spec.ts` (12 tests): clipboard copy (empty/clipboard API/fallback),
  open file location (invalid/success/backslash/error re-throw), open in editor
  (invalid/success/error no-throw)

### ✅ Frontend Composables — useTheme, useEditorDetection, useKeyboardShortcuts, useMatchNavigation, useCodeHighlighting (NEW)
Five previously-untested composables now have dedicated specs:
- `useTheme.spec.ts` (25 tests): readInitialTheme (localStorage/OS/throw),
  applyTheme, setTheme (persist/failure), toggleTheme, isDark
- `useEditorDetection.spec.ts` (22 tests): default factories, event
  subscription (start/progress/complete/dedup), pull-based status race fix
- `useKeyboardShortcuts.spec.ts` (26 tests): Ctrl+K/Cmd+K, Ctrl+Enter, ESC
  (typing vs not), field suppression, optional handlers, mount/unmount
- `useMatchNavigation.spec.ts` (21 tests): totalMatches (empty/regex-escape/
  invalid), next/prev wrap, watch reset
- `useCodeHighlighting.spec.ts` (18 tests): renderPlainText (empty/escape/
  mark/line-numbers/truncate), detectLanguage, loadAndHighlight

### ✅ Frontend Components — EditorSelect, SizeLimitOptions, SymbolSearch, ToastNotification (NEW)
Four previously-untested components now have dedicated specs:
- `EditorSelect.spec.ts` (9 tests): conditional option rendering, placeholder
  + default always present, editorSelect emit
- `SizeLimitOptions.spec.ts` (19 tests): default init, prop propagation,
  contextLines clamping, update emit payload, disabled state
- `SymbolSearch.spec.ts` (12 tests): handleSymbolSearch (empty/no-dir/success/
  error), fetchAllSymbols (cache/dir/progress), selectSymbol, recentlySeenSymbols
- `ToastNotification.spec.ts` (6 tests): renders toasts, type classes,
  pause/resume on hover, close button, progress bar

### ✅ Frontend Services — syntaxHighlightingService, appInitializationService (NEW)
Both previously-untested services now have dedicated specs:
- `syntaxHighlightingService.spec.ts` (27 tests): isHighlightJsLoaded,
  loadHighlightJs (idempotent), detectLanguage (extensions/case/unknown),
  highlightCode (empty/large/truncate/mark/fallback), getHighlightJs
- `appInitializationService.spec.ts` (3 tests): initializeAppServices
  (idempotent loadHighlightJs, no throw)

### ✅ Previously Closed Gaps (from prior sessions)
- End-to-End UX Flows (Playwright harness): 18 tests covering startup, search,
  preview, symbol search, tree, suggestions, case-sensitivity, and enhancements
- Symbol Search backend coverage (symbols_test.go)
### ✅ Backend + E2E — Fuzzy Search (NEW)
Backend phase-2 near-miss candidates via `searchFuzzyCandidates()` in
`SearchWithProgress`: threshold matches the frontend (`max(1, floor(len*0.6))`),
sliding-window best-window scoring, exact-match exclusion in fuzzy phase, regex
disables-fuzzy gating, quota enforcement capped by maxResults. E2E flow via
`fuzzy-search.spec.ts` (7 tests): checkbox presence, fuzzy off/on result delta
(4 exact vs 4+2), fuzzy badge rendering, regex bypasses fuzzy, typo recovery,
long-query threshold behavior.
- File Preview "Black Screen" bug fix (CodeModal v-if guard)
- "Search Returns Nothing" symbol binding fix
- Deterministic result ordering & cancel semantics
- Memoization edge cases (highlightMatch)
- Dead code removal (globalSearchWorkerPool, cachedCompileRegex)

#### 1. Fuzzy Search Accuracy (Frontend)
**File**: `fuzzyMatch.ts`
**Status**: ✅ Closed — E2E flow covered (`fuzzy-search.spec.ts`, 7 tests) and calibration
measured in `search_fuzzy_calibration_test.go`: false-positive rate on 2000 random
text lines stays under the 5% ceiling (~1% measured), single-edit near-misses
(substitution/transposition/deletion/insertion) are reliably found, unrelated words
are rejected, and `BenchmarkFuzzyBestWindow` / `BenchmarkSearchFuzzyCandidates`
(200-file corpus) pin the per-line and per-corpus costs.
#### 2. InlineDiffView Context Rendering
**File**: `InlineDiffView.spec.ts`
**Status**: Partially closed — empty context arrays and multi-match lines covered
(18 tests). Remaining edge cases:
- Single-line matches (match line with no surrounding lines)
- Multiple distinct matches on one line (>3 occurrences)
- Context rendering when the match is at the very first/last line

#### 3. ExportSearchResults Dialog Integration
**File**: `export.go`
**Status**: Pre-dialog logic tested (empty results rejection). The
SaveFileDialog path (format dispatch, file write, user cancel) requires a Wails
runtime context and is only testable in integration.

## Coverage Targets

| Category | Previous | Current | Target | Gap |
|---|---|---|---|---|
| Frontend Unit | 95% (30 files) | ~100% (45 files) | 100% | InlineDiffView edge cases |
| Backend Critical Paths | 90% (24 files) | ~95% (27 files) | 95% | ExportSearchResults dialog |
| Integration | 70% | 85% | 85% | — |
| Edge Cases | 85% | ~95% | 95% | InlineDiffView boundary lines |
| Performance | 60% | 75% | 80% | Fuzzy-scale benchmarks at 10k+ files |
| E2E UX Flows | 9 Playwright | 39 Playwright | Broader coverage | — |

---

Last Updated: 2026-08-14
