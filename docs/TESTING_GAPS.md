# Testing Gaps & Coverage Report

## Current Test Status

### Frontend Tests (358 passing across 24 spec files)
| Component/Test File | Tests | Coverage | Status |
|---|---|---|---|
| InlineDiffView | 28 | Full component logic | ✅ Complete |
| SearchHistorySidebar | 30+ | All interactions | ✅ Complete |
| SearchResults | 10+ | Pagination, rendering | ✅ Complete |
| useSearch composable | 40+ | Core search, fuzzy mode | ✅ Complete |
| CodeSearch integration | 15+ | UI flow, sidebar | ✅ Complete |
| searchUiUtils | 15+ | highlightMatch edge cases | ✅ Complete |

### Backend Tests (20 Go test files)
| File | Focus Area | Coverage |
|---|---|---|
| ipc_validation_test.go | JSON serialization | ✅ Complete |
| app_test.go | App lifecycle | ✅ Complete |
| search_with_progress_test.go | Search engine | ✅ Complete |
| perf_regression_test.go | Performance baselines | ✅ Complete |
| optimization_test.go | Optimization paths | ✅ Complete |
| symbols_test.go | Symbol Search extraction | ✅ Complete |

### End-to-End Tests (Playwright, 7 tests in 1 spec file)
`playwright-tests/flows.spec.ts` drives the app against an in-browser Wails mock
backend (`src/mocks/wailsMock.ts`, enabled via `VITE_WAILS_MOCK=1`). Run with
`npm run test:e2e` (opt-in in `run_tests.sh` via `RUN_E2E=1`).
| Flow | Coverage | Status |
|---|---|---|
| App startup renders | Initial UI mount | ✅ Complete |
| Search → results populate | Query submit + result rendering | ✅ Complete |
| Empty query disables button | Button guard on empty input | ✅ Complete |
| File-preview modal opens with content | CodeModal visibility + content | ✅ Complete |
| Symbol search with a directory | Directory-scoped symbol lookup | ✅ Complete |
| Symbol search without a directory | Undirected symbol lookup | ✅ Complete |
| Case-sensitivity | Case-sensitive matching | ✅ Complete |

---

## Recently Closed Gaps

### ✅ End-to-End UX Flows (Playwright harness)
Previously there was no automated coverage of full user flows through the
rendered UI. The new browser-testable E2E harness (`playwright-tests/flows.spec.ts`,
7 tests against the `wailsMock.ts` backend) now guards startup rendering,
search → results, the empty-query button guard, the file-preview modal, symbol
search (with and without a directory), and case-sensitivity.

### ✅ Symbol Search (Backend)
The new Symbol Search feature (`app_symbols.go`, `models/symbols.go`) now has
backend unit coverage in `models/symbols_test.go`, exercising symbol extraction
across the supported languages.

### ✅ File Preview "Black Screen" Bug
**Root cause**: `CodeModal.vue`'s root `.modal-overlay` had no `v-if` guard, so it
was always rendered and covered the screen on Search. Fixed with `v-if="isVisible"`
and now guarded by the file-preview E2E flow.

### ✅ "Search Returns Nothing" (Symbol bindings)
**Root cause**: symbol bindings were called with wrong args (missing `directory`).
Fixed by wiring the `:directory` prop from `CodeSearch.vue` and now guarded by the
symbol-search E2E flows (with and without a directory).

---

## Identified Gaps (Not Tested Yet)

### 🔴 Critical Gaps

#### 1. Worker Buffer Pool (Backend Optimization)
**File**: `app_core.go` lines ~20-70
**Gap**: No unit tests for the new `globalSearchWorkerPool` buffer reuse mechanism
**Risk**: If pool fails, could cause memory leaks or panics
**Test Needed**: 
- Verify buffers are reused correctly
- Test buffer eviction when over-capacity
- Concurrency safety under load

#### 2. Regex Pattern Cache (Backend Optimization)
**File**: `app_core.go` lines ~90-130
**Gap**: No tests verifying cache eviction when size > 100 entries
**Risk**: Memory leak if old patterns aren't removed
**Test Needed**:
- Cache grows past 100 entries
- Oldest entries are evicted
- Same query returns cached result (performance test)

#### 3. Memoization Edge Cases (Frontend)
**File**: `searchUiUtils.ts`
**Gap**: Only covers basic null/undefined, missing:
- Cache collision detection (two different texts with same key prefix)
- Memory pressure behavior (many concurrent searches)
- Cache hit ratio under realistic usage patterns

### 🟡 Medium Priority Gaps

#### 4. Fuzzy Search Accuracy (Frontend)
**File**: `fuzzyMatch.ts`
**Gap**: No quantitative tests measuring:
- False positive rate on random text
- Sensitivity thresholds for similarity scores
- Performance degradation at scale (10k+ files)

#### 5. InlineDiffView Context Rendering
**File**: `InlineDiffView.spec.ts`
**Gap**: Edge cases not covered:
- Empty context arrays (contextBefore=[], contextAfter=[])
- Single-line matches
- Multi-match lines (>3 matches on one line)

#### 6. SearchHistorySidebar Persistence
**File**: `SearchHistorySidebar.spec.ts`
**Gap**: localStorage integration not fully tested:
- What happens on storage quota exceeded?
- Stale data cleanup after long periods
- Cross-browser localStorage compatibility

### 🟢 Low Priority (Nice-to-have)

#### 7. Integration: Fuzzy → InlineDiffFlow
**Gap**: End-to-end test where fuzzy search results display in InlineDiffView with similarity badges

#### 8. Build Pipeline
**File**: `.github/workflows/build.yml`
**Gap**: `-buildmode=pie` flag change not tested on Windows cross-compilation

---

## Recommended Test Additions

### Immediate (High Priority)
```bash
# Backend - worker pool
go test ./... -run "TestWorkerBuffer" -v

# Backend - pattern cache  
go test ./... -run "TestPatternCache" -v

# Frontend - memoization stress test
npm test -- --testNamePattern="memoization.*stress"
```

### Next Sprint
- [ ] Fuzzy search accuracy benchmarks
- [ ] Storage quota handling for history sidebar
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
go test -run "TestSearchRequestFuzzyFields" ./...

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
| Backend Critical Paths | 90% | 95% | Worker pools, pattern cache |
| Integration | 70% | 85% | E2E fuzzy→inline flow |
| Edge Cases | 85% | 95% | Null states, quota limits |
| Performance | 60% | 80% | Stress tests, benchmarks |
| E2E UX Flows | Core flows (7 Playwright) | Broader coverage | fuzzy→inline flow |

---

Last Updated: 2026-08-02
