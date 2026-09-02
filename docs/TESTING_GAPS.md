# Testing Gaps & Coverage Report

## Current Test Status

### Frontend Tests (714 passing across 48 spec files)

Per-spec test counts are deliberately absent from this document. They used to be
carried in a `Tests` column here, and ten of them had drifted: InlineDiffView,
useTheme, useMatchNavigation, useCodeHighlighting, fileUtils, toastUtils and
SymbolSearch each disagreed with the "Recently Closed Gaps" narrative further
down this same file, the counts quoted there for `replace_test.go` (11) and
`collection_index_test.go` (8) were both short of the real ones (12 each), and
`useSelectionManager` was listed as 12 in both this file and `docs/TESTING.md`
while the spec held 13. A count changes on every commit that adds a case; what a
spec defends does not. Totals are kept because they are cheap to keep right.

| Component/Test File | Coverage | Status |
|---|---|---|
| InlineDiffView | Full component logic | ✅ Complete |
| SearchHistorySidebar | All interactions | ✅ Complete |
| SearchResults | Pagination, rendering | ✅ Complete |
| useSearch composable | Core search, fuzzy mode, streamed-batch append | ✅ Complete |
| CodeSearch integration | UI flow, sidebar | ✅ Complete |
| searchUiUtils | highlightMatch, memoization edge cases | ✅ Complete |
| fuzzyMatch | findFuzzyMatches: similarity thresholds, case-insensitivity, whole-text scan, perf bail-out; debounce helper | ✅ Complete |
| localStorageUtils | Save/load round-trip, quota/disabled storage, remove, key stability | ✅ Complete |
| TreeViewPanel | Tree building, ordering, file-click | ⚠️ Filter path untested |
| SearchSuggestions | Rendering, select/remove, close-on-outside-click | ✅ Complete |
| errorUtils | toErrorMessage (Error/string/object/null/fallback), asRecord (object/array/null/primitives) | ✅ Complete |
| fileUtils | formatFilePath (empty/short/long/multi-part), truncatePath (default/custom maxLength) | ✅ Complete |
| toastUtils | copyToClipboardWithToast (empty/clipboard API/fallback), openFileLocationWithToast (invalid/success/backslash/error) | ✅ Complete |
| useTheme | readInitialTheme (localStorage/OS fallback/throw), applyTheme, setTheme (persist/failure), toggleTheme, isDark | ✅ Complete |
| useEditorDetection | makeDefault factories, subscribeToEditorDetectionEvents (start/progress/complete/dedup), startEditorDetection (pull-based race fix) | ✅ Complete |
| useKeyboardShortcuts | Ctrl+K/Cmd+K, Ctrl+Enter, ESC (typing vs not), field suppression, handler optional, mount/unmount | ✅ Complete |
| useMatchNavigation | totalMatches (empty/regex-escape/invalid), goToNextMatch (wrap), goToPreviousMatch (wrap), watch reset | ✅ Complete |
| useCodeHighlighting | renderPlainText (empty/escape/mark/line-numbers/truncate), detectLanguage, loadAndHighlight (ready/fallback/reject) | ✅ Complete |
| EditorSelect | Renders available editors, placeholder + default always present, emits editorSelect on change | ✅ Complete |
| SizeLimitOptions | Default init, prop propagation, contextLines clamping, update emit payload, disabled state | ✅ Complete |
| SymbolSearch | handleSymbolSearch (empty/no-dir/success/error), fetchAllSymbols (cache/dir/progress), selectSymbol, recentlySeenSymbols | ⚠️ Re-index button untested |
| ToastNotification | Renders toasts, type classes, pause/resume on hover, close button, progress bar | ✅ Complete |
| syntaxHighlightingService | isHighlightJsLoaded, loadHighlightJs (idempotent + load-failure error toast), detectLanguage (extensions/case/unknown), highlightCode (empty/large/truncate/mark/fallback), getHighlightJs | ✅ Complete |
| useSelectionManager | reactive selectedCount/allVisibleSelected, toggleSelected, toggleSelectAll, clearSelection, copy/export selected subset + all-fallback, export-format threading | ✅ Complete |
| useReplace | preview calls binding with apply=false, apply calls apply=true then re-runs search, regex-mode guard, apply-without-preview no-op, zero-change preview, missing-query guard | ⚠️ `replace-progress` untested |
| useSymbolSearch | Stale-response generation guard only (1 test) | ⚠️ Thin |
| searchProgress | `coerceProgress` only | ⚠️ `coerceResultBatch` untested |

### Backend Tests (39 Go test files)
| File | Focus Area | Coverage |
|---|---|---|
| gap_fixes_test.go | **NEW** — csvSafeCell leading-space bypass, MaxResults cap, protected subtree, symlink skip/non-traversal | ✅ Complete |
| helpers_test.go | parseLogLine/parseLogEntryMessage/isNoisyMessage, matchesPattern, getFullExtension/matchExtension, isKnownTextExtension, containsDotDotComponent, safeContextLinesBytes/bytesToStrings/searchContextLines, validateAndSetDefaults (including subtree + cap), rotateLogFileIfNeeded, ReadFileLog, GetDirectoryContents | ✅ Complete |
| replace_test.go | ReplaceInFiles: dry-run writes nothing, atomically applies, preserves file mode, rejects regex, rejects empty query, skips no-op replacements, case-sensitivity, multi-file, result determinism, file-vanished-between-preview-and-apply, multi-directory; writeFileAtomic rename-over-directory cleans temp | ✅ Complete |
| collection_index_test.go | collectionCacheKey sorting/no-collision/slice-collision-prevention, computeCollectionFingerprint (stable, changes on edit), collectionCache (miss/populate, eviction, stale fingerprint), collectFilesToProcess cached result equality, cache bypass on different filter | ✅ Complete |
| gitignore_test.go | Root-level helpers only (nested resolution lives in gitignore_nested_test.go): loadGitignoreMatcher (no files, root ignore, negation, .git/info/exclude, both sources), filterByGitignore (nil matcher, match/drop), collectFilesToProcess RespectGitignore on/off | ✅ Complete |
| ipc_validation_test.go | JSON serialization | ✅ Complete |
| app_test.go | App lifecycle, IsAppReady, GetInitialLogs/GetNewLogs | ✅ Complete |
| search_with_progress_test.go | Search engine | ✅ Complete |
| perf_regression_test.go | Performance baselines | ✅ Complete |
| optimization_test.go | Optimization paths | ✅ Complete |
| symbols_test.go | Symbol extraction over Go/TypeScript/Vue fixtures; TestGetPatternsForExtension iterates `symbolSupportedExtensions` (`.txt` as the negative case) so a new language cannot leave the slice and the switch disagreeing | ⚠️ No fixtures for `.py`/`.rs`/`.java`/`.cs`/`.rb` |
| export_test.go | CSV rendering, csvSafeCell formula guard, empty-results rejection | ⚠️ Dialog path is an unconditional `t.Skip` |
| system_integration_fixes_test.go | Editor bindings, JetBrains routing, snapshot count, path traversal, null bytes | ✅ Complete |
| search_fuzzy_test.go | Fuzzy near-miss phase: threshold (60% positional), best-window scoring, regex gating, exact-match exclusion, quota enforcement | ✅ Complete |
| search_fuzzy_calibration_test.go | Fuzzy calibration: measured false-positive rate on 2000 random lines (<5% ceiling, ~1% measured), single-edit near-miss sensitivity, unrelated-word rejection, near-miss discovery at scale (200 files) | ✅ Complete |
| editor_launch_fixes_test.go | Editor-launch hardening: appendPath aliasing safety, startAndReap zombie reaping, OpenInDefaultEditor path validation | ✅ Complete |
| search_streaming_batch_test.go | **NEW** — `resultBatcher` size flush + monotonic seq, empty flush burns no seq, no shared backing array between consecutive batches, `recordFailure` sample cap with exact count, `snapshotFailedPaths` copy semantics, empty path counted but not listed, end-to-end unreadable-file reporting on the terminal payload | ✅ Complete |
| gitignore_nested_test.go | **NEW** — nested precedence (deeper overrides shallower), own-directory-relative patterns, directory pruning, pruning abandoned on contents-rule negation, gate-off reads no ignore files, read-once-per-directory cost model, fingerprint invalidation on nested edit | ✅ Complete |
| app_shared_test.go | **NEW** — path-validation trust boundary: `sanitizePath` traversal rejection pre- and post-`Clean`, dots-in-filenames accepted, `validatePathForEditor` existence check, `validatePathForShowInFolder` parent resolution, `lookUpEditor` absolute-path/TOCTOU contract, `appendPath` shared-backing-array guard, `isSymbolSupportedExtension` across all 10 languages, `symbolCacheKey` normalization | ✅ Complete |

### End-to-End Tests (Playwright, 41 tests in 7 spec files)
`playwright-tests/` drives the app against an in-browser Wails mock backend
(`src/mocks/wailsMock.ts`, enabled via `VITE_WAILS_MOCK=1`): `flows.spec.ts`,
`find-replace.spec.ts`, `filetree-suggestions.spec.ts`, `enhancements.spec.ts`,
`search-options.spec.ts`, `advanced-search.spec.ts`, and `fuzzy-search.spec.ts`.
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
| Replace row hidden under regex | Regex mode hides the replace controls (backend rejects regex replace) | ✅ Complete |
| Replace preview + apply | Preview shows old→new diffs, apply writes, re-search reflects the change | ✅ Complete |

---

## Recently Closed Gaps

### ✅ Security & Perf Hardening (NEW)
Hardening from the 2026-08-30 audit, covered by `gap_fixes_test.go`:
- `csvSafeCell` — space-prefixed formula bypass (`" =2+2"` → `"' =2+2"`), tab-prefixed triggers, and the `TrimLeft(" ")` fix (tabs are triggers, not trimmable).
- `MaxResults` hard cap — 10000 enforced in `validateAndSetDefaults`; `very_large_values` in `data_validation_test.go` updated to expect rejection.
- Protected-directory subtrees — `/etc` now blocks `/etc/ssh` via `prefix+separator` without blocking `/etc-backup`.
- Symlink handling — file symlinks skipped (no `MaxFileSize` bypass / OOM), dir symlinks not traversed, base dir resolved via `EvalSymlinks`.
- Frontend — CSP meta in `index.html` (`default-src 'self'`), `debounce` + `DebouncedFn` in `fuzzyMatch.ts` (also restores `MAX_TEXT_LENGTH_FOR_FUZZY_SEARCH` for the `fuzzy_parity_test.go` tripwire).

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
  empty/non-existent/protected directory rejection (including subtrees and MaxResults cap)
- `rotateLogFileIfNeeded` — no-op for missing/small file, rotation on
  exceed, overwrite of previous .1 rotation
- `ReadFileLog` — path resolution, different names
- `GetDirectoryContents` — subdirectory listing, hidden dir skipping, file
  exclusion, non-existent path returns empty

### ✅ ExportSearchResults Pre-Dialog Contract (NEW)
`export_test.go` covers the binding's pre-dialog contract: empty/nil results
rejection, and CSV rendering including the `csvSafeCell` formula guard. The
dialog path itself is still open — see the open gaps below.

Three previously-untested pure utility modules now have dedicated specs:
- `errorUtils.spec.ts`: toErrorMessage for Error/string/object/null/undefined/
  number/non-string-message, custom fallback; asRecord for objects/arrays/
  null/primitives
- `fileUtils.spec.ts`: formatFilePath truncation rules, truncatePath with
  default and custom maxLength
- `toastUtils.spec.ts`: clipboard copy (empty/clipboard API/fallback), open
  file location (invalid/success/backslash/error re-throw), open in editor
  (invalid/success/error no-throw)

### ✅ Frontend Composables — useTheme, useEditorDetection, useKeyboardShortcuts, useMatchNavigation, useCodeHighlighting (NEW)
Five previously-untested composables now have dedicated specs:
- `useTheme.spec.ts`: readInitialTheme (localStorage/OS/throw), applyTheme,
  setTheme (persist/failure), toggleTheme, isDark
- `useEditorDetection.spec.ts`: default factories, event subscription
  (start/progress/complete/dedup), pull-based status race fix
- `useKeyboardShortcuts.spec.ts`: Ctrl+K/Cmd+K, Ctrl+Enter, ESC (typing vs
  not), field suppression, optional handlers, mount/unmount
- `useMatchNavigation.spec.ts`: totalMatches (empty/regex-escape/invalid),
  next/prev wrap, watch reset
- `useCodeHighlighting.spec.ts`: renderPlainText (empty/escape/mark/
  line-numbers/truncate), detectLanguage, loadAndHighlight

### ✅ Frontend Components — EditorSelect, SizeLimitOptions, SymbolSearch, ToastNotification (NEW)
Four previously-untested components now have dedicated specs:
- `EditorSelect.spec.ts`: conditional option rendering, placeholder + default
  always present, editorSelect emit
- `SizeLimitOptions.spec.ts`: default init, prop propagation, contextLines
  clamping, update emit payload, disabled state
- `SymbolSearch.spec.ts`: handleSymbolSearch (empty/no-dir/success/error),
  fetchAllSymbols (cache/dir/progress), selectSymbol, recentlySeenSymbols
- `ToastNotification.spec.ts`: renders toasts, type classes, pause/resume on
  hover, close button, progress bar

### ✅ Frontend Services — syntaxHighlightingService (NEW)
Previously untested service now has a dedicated spec:
- `syntaxHighlightingService.spec.ts`: isHighlightJsLoaded, loadHighlightJs
  (idempotent + load-failure error toast), detectLanguage (extensions/case/
  unknown), highlightCode (empty/large/truncate/mark/fallback), getHighlightJs

### ✅ Streamed Results & Failed-File Reporting (NEW)
`search_streaming_batch_test.go` covers the whole new streaming contract:
`resultBatcher` flushes at `resultBatchSize` with a monotonic `seq` from 1, a
terminal flush with nothing pending emits no event and burns no sequence number
(otherwise the frontend would treat `seq` 1 as delivered and drop the next
search's first batch), and a flushed batch never shares a backing array with the
one that follows it. `SearchState.recordFailure` / `snapshotFailedPaths`
(`models.go:80`, `:94`) are covered for the exact-count/capped-sample split, the
copy-not-live-slice return, and the empty-path case; the real pipeline is driven
against an unreadable file to prove the failure reaches the terminal `completed`
payload as both a count and a named path.

### ✅ Nested `.gitignore` (NEW)
`gitignore_nested_test.go` covers `ignoreStack`: deeper ignore files override
shallower ones, patterns resolve relative to their own `.gitignore`'s directory
rather than the search root (the bug the root-only implementation had), a plain
directory rule prunes the subtree, pruning is abandoned when a contents-rule
negation (`build/*` + `!build/keep.txt`) re-includes a file, `RespectGitignore`
off reads no ignore files at all, the memo map proves each ignore file is read
once per directory rather than once per file, and editing a nested `.gitignore`
invalidates the collection fingerprint.

### ✅ Path-Validation Trust Boundary (NEW)
`app_shared.go` is the trust boundary every editor launch and folder reveal
crosses, and it had **no direct tests** — only coverage through the
build-tagged platform wrappers, each of which runs on one OS. `app_shared_test.go`
closes it: `sanitizePath` rejects traversal in both the original and the
`filepath.Clean`-ed path, dots inside filenames are still accepted (the
component-based check, not a `..` substring ban), `validatePathForEditor`
requires the file to exist, `validatePathForShowInFolder` resolves to the parent
directory, `lookUpEditor` returns the resolved absolute path (the TOCTOU-closing
contract), and `appendPath` never mutates the shared `editorCatalog` args slices.
It also covers `isSymbolSupportedExtension` across all ten languages and
`symbolCacheKey` normalization.

### ✅ Test-Suite Self-Consistency Gaps (NEW)
Three defects in the tests themselves, each of which had made a real drift
undetectable:
- `TestGetPatternsForExtension` (`symbols_test.go:334-355`) drove off a
  hardcoded extension list with `.py` as its negative case, so it asserted
  Python was *unsupported* — and broke the moment Python became supported. It
  now iterates `symbolSupportedExtensions`, so the slice and the
  `getPatternsForExtension` switch cannot disagree silently. `.txt` is the
  negative case: text, so it reaches collection, but with no symbol grammar.
- The shared vitest mock (`frontend/tests/__mocks__/wailsjs/go/main/App.ts`)
  exported `SearchCode`, a binding that does not exist on the Go side, while
  omitting `GetAllSymbols`/`SearchSymbols` — which forced every symbol spec to
  re-declare them in a local `vi.mock`, so nothing detected the drift. The
  phantom export is gone and the symbol bindings are declared once centrally.
  `ValidateDirectory` defaults to `true` because `useSearch` now gates every
  search on it.
- `useSelectionManager.exportSelectedResults` hardcoded `"csv"` and ignored the
  requested format, so the JSON export button produced CSV.
  `useSelectionManager.spec.ts` gained a regression test asserting the requested
  format is threaded through to the binding.

### ✅ Previously Closed Gaps (from prior sessions)
- End-to-End UX Flows (Playwright harness): startup, search, preview, symbol
  search, tree, suggestions, case-sensitivity, and enhancements
- Symbol Search backend coverage (symbols_test.go)
### ✅ Backend + E2E — Find & Replace, Collection Cache, .gitignore (NEW)
Three backend capabilities landed together with full test coverage:
- `ReplaceInFiles` binding (`replace.go`): literal replace across matched lines,
  dry-run (`Apply=false`) vs atomic apply, regex-mode rejection, no-op skip,
  path sanitization, file-mode preservation, deleted-file skip, multi-directory.
  Covered by `replace_test.go` + E2E `find-replace.spec.ts`.
- Persistent collection cache (`collection_index.go`): fingerprint-validated,
  keyed by directory + filter-set; unambiguous key encoding prevents slice
  collisions. Covered by `collection_index_test.go`.
- `.gitignore` support (`gitignore.go`): root `.gitignore` + `.git/info/exclude`
  via go-gitignore, negation/`**`/anchoring honored, off by default. Covered by
  `gitignore_test.go`; nested resolution landed later (see above).
- Frontend: `useReplace.ts` composable, 5th search checkbox, replace controls in
  results header (hidden under regex mode).

### ✅ Backend + E2E — Fuzzy Search (NEW)

Backend phase-2 near-miss candidates via `searchFuzzyCandidates()` in
`SearchWithProgress`: threshold matches the frontend (`max(1, floor(len*0.6))`),
sliding-window best-window scoring, exact-match exclusion in fuzzy phase, regex
disables-fuzzy gating, quota enforcement capped by maxResults. E2E flow via
`fuzzy-search.spec.ts`: checkbox presence, fuzzy off/on result delta (4 exact vs
4+2), fuzzy badge rendering, regex bypasses fuzzy, typo recovery, long-query
threshold behavior.
- File Preview "Black Screen" bug fix (CodeModal v-if guard)
- "Search Returns Nothing" symbol binding fix
- Deterministic result ordering & cancel semantics
- Memoization edge cases (highlightMatch)
- Dead code removal (globalSearchWorkerPool, cachedCompileRegex)

---

## Open Gaps

#### 1. `ExportSearchResults` dialog path — entirely untested
**File**: `export.go`, `export_test.go:121-123`
`TestExportSearchResultsRequiresContext` is an unconditional `t.Skip` with an
empty body — it asserts nothing on any platform. So the dialog path
(`SaveFileDialog` invocation, format dispatch between CSV and JSON, the file
write, and the user-cancel branch) has **zero** coverage; only the pre-dialog
rejection of empty results is tested. `SaveFileDialog` panics rather than
erroring with a nil context, which is why the skip exists, but the format
dispatch and write are pure logic that could be extracted behind the dialog call
and tested directly. This is the JSON-vs-CSV path where
`useSelectionManager.exportSelectedResults` already shipped one format bug.

#### 2. `search_workers.go` — worker-coordination primitives
**File**: `search_workers.go:96` (`workerShouldContinue`), `:256` (`emitFileProgress`)
The `resultBatcher` and the failure-sample path in this file are now covered, but
these two are not: no test file names either function. `workerShouldContinue`
owns the maxResults compare-and-swap that cancels the shared context exactly once
(a double cancel or a missed one changes how many results a capped search
returns), and `emitFileProgress` self-detects the last file from its
post-increment count precisely because two workers racing between Load and Add
would otherwise both believe they were last. Both are concurrency contracts
asserted only in comments.

#### 3. `symbol_scan.go` — partially closed
**File**: `symbol_scan.go:60` (`isSymbolSupportedExtension`), `:73` (`shouldSkipDirForSymbolScan`)
`isSymbolSupportedExtension` is **now covered** by `app_shared_test.go:191-213`
across all ten extensions plus case-insensitivity. `shouldSkipDirForSymbolScan`
still has no direct test: `TestGetAllSymbols_SkipsNodeModules`
(`symbols_test.go:196-212`) exercises it indirectly for `node_modules` only, so
the entries added for the new languages (`__pycache__`, `.venv`, `venv`,
`.mypy_cache`, `.pytest_cache`, `target`, `.gradle`, `obj`) are unverified — a
typo in any of them would silently index build output.

#### 4. Five new symbol languages — patterns compiled, extraction unverified
**File**: `symbols.go:327` (`pyPatterns`), `:343` (`rustPatterns`), `:365` (`javaPatterns`), `:388` (`csPatterns`), `:411` (`rubyPatterns`)
`symbols_test.go` contains no `.py`, `.rs`, `.java`, `.cs`, or `.rb` fixture at
all — its extraction tests are Go, TypeScript, and Vue only.
`TestGetPatternsForExtension` proves each extension returns non-empty patterns
whose regexes compile; nothing proves any of those regexes matches the
declaration it was written for, or that the `_`-prefix skip keeps Python dunders
(`__init__`) while dropping `_helper`. Twenty-eight regexes are effectively
unexercised.

#### 5. `replace-progress` — emitted and consumed, asserted nowhere
**Files**: `replace.go`, `frontend/src/composables/useReplace.ts:66-87`
The backend emits the event and `useReplace` subscribes per call, coerces the
payload, and exposes a `progress` ref. No test on either side references
`replace-progress`: not `replace_test.go`, not `useReplace.spec.ts`, not the
Playwright specs. The per-call subscribe/teardown exists specifically so a stale
handler cannot repopulate `progress` after its operation finished — the exact
kind of lifecycle bug a test would catch and a manual pass would not.

#### 6. Frontend streaming path — untested in vitest
**Files**: `frontend/src/composables/searchProgress.ts` (`coerceResultBatch`), `useSearch.ts:338-347`
`searchProgress.spec.ts` imports only `coerceProgress` (`:2`), so
`coerceResultBatch` has no unit coverage. The append logic in `useSearch` — drop
`seq <= lastBatchSeq`, clamp to `maxResults`, reset `lastBatchSeq` per search,
ignore batches from a superseded generation — is likewise unasserted. The E2E
mock does emit two batches (`frontend/src/mocks/wailsMock.ts:338-339`), so the
happy-path append runs during Playwright flows, but nothing checks the dropping,
the clamp, or the generation guard. A silently-dropped first batch would look
identical to a slow search.

#### 7. New UI controls without specs
- `TreeViewPanel.vue` tree filter: `TreeViewPanel.spec.ts` has no `filterText`
  case, so the 150ms debounce, the hiding of unmatched roots, and the distinct
  no-match empty state are untested. `EnhancedTreeItem.spec.ts` covers the
  child's `filterText` prop, which is the path the panel feeds — not the panel.
- `SearchForm.vue:40-48` extension filter: `SearchForm.spec.ts` resolves
  `findComponent({ name: 'QueryInput' })`, which returns the *first* QueryInput
  (the query field at `:18`), so the second instance — the extension input — is
  never touched.
- `ProgressIndicator.vue:22-41` skipped-file summary: `ProgressIndicator.spec.ts`
  makes no assertion about `failedFiles` or `failedPaths`, so neither the count,
  the capped path list, nor the "…and N more" line is covered.
- `useSymbolSearch.spec.ts` holds exactly **one** test (`:21`, the stale-response
  generation guard). The composable has since grown `reindexSymbols`
  (`useSymbolSearch.ts:166`), which calls `ClearSymbolCache` and participates in
  that same generation guard, plus the SymbolSearch Re-index button — none of it
  tested.

#### 8. `GetDirectoryContents` bounds
**File**: `system_integration.go:195` (`maxDirectoryListing`), `:200` (`maxDirectoryDepth`)
`helpers_test.go:548-596` covers listing, hidden-directory skipping, file
exclusion, and the non-existent path. Neither bound is named by any test, so the
depth prune and the deliberate decision to return an **error** on listing
truncation rather than a silently partial tree are unverified.

#### 9. Frontend modules with no spec of their own
Established by diffing `frontend/src` against `frontend/tests`:
- `MatchNavigationControls.vue` — reached through `CodeModal.spec.ts:261`
  (`.line-input` presence) only
- `ModalFooter.vue` — reached through `CodeModal.spec.ts`'s
  `.modal-footer-info` assertions; the footer's three action buttons and the
  `canJumpToLine` gate are covered only by E2E
- `PaginationControls.vue`, `ExportActions.vue` — mounted inside
  `SearchResults.spec.ts` (full `mount`, not stubbed) but never asserted on
- `StartupLoader.vue`, `App.vue` — not imported by any spec
- `useLogViewer.ts` — collapse/expand exercised through
  `LogViewer.spec.ts:106-127`; the auto-scroll (tail) behavior is not

The same diff also flags `constants/appConstants.ts` and `constants/editors.ts`.
Both are pure data with no logic (four numeric constants; the `EDITOR_CATALOG`
array), and the catalog is already pinned from the Go side by
`editor_catalog_test.go` and the snapshot count in
`system_integration_fixes_test.go`. Not counted as gaps.

#### 10. InlineDiffView context rendering
**File**: `InlineDiffView.spec.ts`
Partially closed — empty context arrays and multi-match lines are covered.
Remaining edge cases:
- Single-line matches (match line with no surrounding lines)
- Multiple distinct matches on one line (>3 occurrences)
- Context rendering when the match is at the very first/last line

## Coverage

| Category | Measured | Gate | Status |
|---|---|---|---|
| Go statements | **80.0%** (`go tool cover -func` total, under `-race -covermode=atomic`) | 80% (`.github/workflows/build.yml:127-134`) | ✅ Passing, no margin |
| Go test files | 39 | — | — |
| Frontend vitest | 714 tests across 48 spec files; **coverage not measured** | thresholds declared, inert | ⚠️ Unmeasured |
| E2E | 41 Playwright tests across 7 spec files | — | — |

The Go number passes with no headroom: what breaks that gate in practice is a new
file landing untested, not a regression inside an existing one.

`frontend/vitest.config.ts:43-55` declares thresholds (lines/functions/statements
80, branches 70), but `npm test` is `vitest run` with no `--coverage`, so nothing
computes a number to compare against them. The frontend figure is therefore
**unmeasured**, not merely unenforced — the previous "~100%" entry in this table
was an estimate with nothing behind it, and gap 9 above lists seven source
modules with no spec at all, so it was also wrong. Upgrade: add a
`test:coverage` script running `vitest run --coverage` and call it from CI.

Known-open backend branch: `writeFileAtomic`'s Write/Chmod failure paths, which
need OS-level injection to reach.

---

Last Updated: 2026-09-02
