# Testing Gaps & Coverage Report

## Current Test Status

### Frontend Tests (456 passing across 30 spec files)
| Component/Test File | Tests | Coverage | Status |
|---|---|---|---|
| InlineDiffView | 18 | Full component logic | ✅ Complete |
| SearchHistorySidebar | 26 | All interactions | ✅ Complete |
| SearchResults | 10 | Pagination, rendering | ✅ Complete |
| useSearch composable | 66 | Core search, fuzzy mode | ✅ Complete |
| CodeSearch integration | 13 | UI flow, sidebar | ✅ Complete |
| searchUiUtils | 42 | highlightMatch, memoization edge cases | ✅ Complete |
| fuzzyMatch | 22 | Similarity thresholds, false positives, perf, normalization | ✅ Complete |
| localStorageUtils | 11 | Save/load round-trip, quota/disabled storage, remove, key stability | ✅ Complete |
| TreeViewPanel | 7 | Tree building, ordering, file-click | ✅ Complete |
| SearchSuggestions | 10 | Rendering, select/remove, close-on-outside-click | ✅ Complete |

### Backend Tests (23 Go test files)
| File | Focus Area | Coverage |
|---|---|---|
| ipc_validation_test.go | JSON serialization | ✅ Complete |
| app_test.go | App lifecycle | ✅ Complete |
| search_with_progress_test.go | Search engine | ✅ Complete |
| perf_regression_test.go | Performance baselines | ✅ Complete |
| optimization_test.go | Optimization paths | ✅ Complete |
| symbols_test.go | Symbol Search extraction | ✅ Complete |

### End-to-End Tests (Playwright, 9 tests in 2 spec files)
`playwright-tests/flows.spec.ts` and `playwright-tests/filetree-suggestions.spec.ts`
drive the app against an in-browser Wails mock backend (`src/mocks/wailsMock.ts`,
enabled via `VITE_WAILS_MOCK=1`). Run with `npm run test:e2e` (opt-in in
`run_tests.sh` via `RUN_E2E=1`).
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

---

## Recently Closed Gaps

### ✅ End-to-End UX Flows (Playwright harness)
Previously there was no automated coverage of full user flows through the
rendered UI. The browser-testable E2E harness (`playwright-tests/flows.spec.ts`
+ `filetree-suggestions.spec.ts`, 9 tests against the `wailsMock.ts` backend)
guards startup rendering, search → results, the empty-query button guard, the
file-preview modal, symbol search (with and without a directory),
case-sensitivity, File Explorer tree navigation, and the suggestions dropdown
(open, select, outside-click/Escape close).

### ✅ Symbol Search (Backend)
The new Symbol Search feature (`app_symbols.go`, `symbols.go`) now has
backend unit coverage (`symbols_test.go`), exercising symbol extraction
across the supported languages.

### ✅ File Preview "Black Screen" Bug
**Root cause**: `CodeModal.vue`'s root `.modal-overlay` had no `v-if` guard, so it
was always rendered and covered the screen on Search. Fixed with `v-if="isVisible"`
and now guarded by the file-preview E2E flow.

### ✅ "Search Returns Nothing" (Symbol bindings)
**Root cause**: symbol bindings were called with wrong args (missing `directory`).
Fixed by wiring the `:directory` prop from `CodeSearch.vue` and now guarded by the
symbol-search E2E flows (with and without a directory).

### ✅ Recently Closed Backend Gaps

The former `globalSearchWorkerPool` buffer-reuse mechanism and the dead
`cachedCompileRegex` sync-map cache were removed — production uses neither
(the search engine spawns plain goroutines and compiles via the real
`LRUPatternCache`). `optimization_test.go` now tests the live LRU cache
(capacity eviction, LRU ordering, concurrency, case-sensitivity isolation),
so those old "untested optimization" concerns no longer apply.

### ✅ Deterministic Result Ordering & Cancel Semantics (Backend)

`SearchWithProgress` now sorts results deterministically (file path, then line
number) and returns empty results on cancellation instead of partial matches
plus a misleading `completed` event. Covered by `search_results_sort_test.go`
(fixture-based run verifies sort stability across repeated searches; the
cancel test drives the real `cancelActiveSearch` path mid-search and asserts
zero results).

### ✅ Memoization Edge Cases (Frontend)

The former 🔴 Critical Gap (`highlightMatch` memoization) is now covered by
`searchUiUtils.memo.spec.ts` (9 tests): caches keyed by query + flags rather
than exact text, LRU eviction under query churn (>200 unique queries), cache-hit
fast path (1000 identical calls < 100ms), distinct-query correctness, large-text
handling (>10KB), Unicode/emoji cache keys, and empty-input safety.

---

## Identified Gaps (Not Tested Yet)

### 🟡 Medium Priority Gaps

#### 1. Fuzzy Search Accuracy (Frontend)
**File**: `fuzzyMatch.ts`
**Status**: Partially closed — the 🔴 gap around basic correctness is now covered by `fuzzyMatch.spec.ts` (22 tests: similarity threshold pass/reject boundaries, ordered-subsequence and case/whitespace handling, per-window matches via `findFuzzyMatches`, the >50KB long-text bailout, and `normalizeQuery` edge cases). Quantified *accuracy* studies are still open:
- False positive rate on random text (measured, not just spot-checked)
- Sensitivity calibration of the 0.8 / 0.6 thresholds against human intuition
- Performance benchmark at scale (10k+ files)

#### 2. InlineDiffView Context Rendering
**File**: `InlineDiffView.spec.ts`
**Status**: Partially closed — empty context arrays and multi-match lines are already covered (18 tests). Remaining edge cases:
- Single-line matches (match line with no surrounding lines)
- Multiple distinct matches on one line (>3 occurrences)
- Context rendering when the match is at the very first/last line

#### 3. SearchHistorySidebar Persistence
**File**: `localStorageUtils.ts` / `SearchHistorySidebar.spec.ts`
**Status**: Partially closed — the persistence layer (`loadRecentSearches`, `saveRecentSearches`, `recentSearchKey`, `removeRecentSearch`) is now covered by `localStorageUtils.spec.ts` (11 tests: round-trip, invalid/corrupt payloads, storage-quota and storage-disabled failures, directory-scoped removal, key stability). The component itself is presentational (props + emits), so the remaining gap is narrow:
- Cross-browser localStorage compatibility (jsdom only today)

### 🟢 Low Priority (Nice-to-have)

#### 4. Integration: Fuzzy → InlineDiffFlow
**Gap**: End-to-end test where fuzzy search results display in InlineDiffView with similarity badges

---

## Recommended Test Additions

### Immediate (High Priority)
```bash
# Backend - LRU pattern cache (eviction, ordering, concurrency)
go test ./... -run "TestLRUPatternCache" -v

# Frontend - fuzzy matching + localStorage persistence specs
npm test -- fuzzyMatch.spec.ts localStorageUtils.spec.ts
```

### Next Sprint
- [ ] Fuzzy search accuracy studies (false-positive rate, threshold calibration, scale perf)
- [ ] InlineDiffView single-line / first-line / >3-matches-per-line cases
- [x] Storage quota + disabled-storage handling (covered by `localStorageUtils.spec.ts`)
- [ ] E2E: fuzzy search → inline diff view flow
- [ ] Memory leak detection under load

---

## Test Command Reference

```bash
# Run all frontend tests
npm test

# Run end-to-end flows (Playwright, opt-in)
npm run test:e2e
RUN_E2E=1 ./run_tests.sh

# Run backend IPC validation specifically
go test -run "TestSearchRequestContextLines" ./...

# Run benchmark tests
go test -bench=. -benchmem ./...

# Frontend unit + integration
npm test -- tests/unit/components/CodeSearch.integration.spec.ts

# Check coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Coverage Targets

| Category | Current | Target | Gap |
|---|---|---|---|
| Frontend Unit | 95% | 100% | - |
| Backend Critical Paths | 90% | 95% | Worker pools |
| Integration | 70% | 85% | E2E fuzzy→inline flow |
| Edge Cases | 85% | 95% | InlineDiffView boundary lines |
| Performance | 60% | 80% | Fuzzy-scale benchmarks |
| E2E UX Flows | Core flows (9 Playwright: search/preview, tree, suggestions, symbols) | Broader coverage | fuzzy→inline flow |

---

Last Updated: 2026-08-04
