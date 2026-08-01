# New Features & Enhancements (v1.x)

## Overview

This document describes recent enhancements added to Code Search, including fuzzy matching, inline diff views, search history sidebar, and performance optimizations.

---

## Feature Highlights

### 1. Fuzzy Search 🔍

**What it does:** Finds matches even with typos or slight misspellings using Levenshtein-style similarity scoring.

**How to use:** 
- Toggle "Fuzzy Search" checkbox in SearchForm
- Client-side filtering after backend returns results
- Results marked with `~` badge showing similarity percentage

**Example:** Searching for `"test messgae"` will find files containing `"test message"` with 90%+ similarity.

**Technical details:**
- Implemented in `frontend/src/utils/fuzzyMatch.ts`
- Configurable via `data.fuzzySearch` boolean
- Falls back to exact match when disabled
- Works alongside existing regex mode

---

### 2. Inline Diff View 📄

**What it does:** Replaces standard result rows with enhanced component showing context before/after match with line numbers and copy functionality.

**Visual changes:**
- Matched line highlighted with yellow background
- Context lines color-coded (blue for before, purple for after)
- Line numbers visible on left side
- Copy button on each result line
- Diff hint showing context depth

**Access:** Automatically shown when viewing search results (no configuration needed)

**Performance:** Only renders for visible page (pagination preserves performance)

---

### 3. Search History Sidebar 📁

**What it does:** Persistent sidebar showing recently executed searches with click-to-re-search capability.

**Features:**
- Stores up to 5 recent searches in browser localStorage
- Click any entry to re-execute that search
- Remove individual searches via × button
- Clear all button to wipe history
- Collapsible sidebar (↔ toggle button)
- Active search highlighted in green

**Persistence:** Searches saved immediately after successful query execution

**Location:** Left side of main content area (replaces previous centered layout)

---

### 4. Performance Optimizations ⚡

**Paginated highlighting:** Previously highlighted ALL results (up to 1000), now only highlights visible 10 items per page.

**InlineDiffView efficiency:** Single component per result vs multiple v-html calls reduces DOM complexity.

**ContextLines parameter:** Backend now accepts custom context line count (default: 3 lines before/after).

---

## Technical Changes

### Backend (`models.go`)

Added two new fields to `SearchRequest`:
```go
type SearchRequest struct {
    // ... existing fields ...
    FuzzySearch   bool     `json:"fuzzySearch"`   // Enable typo-tolerant matching
    ContextLines  int      `json:"contextLines"`  // Lines before/after match (default 3)
}
```

### Frontend Types (`frontend/src/types/search.ts`)

Updated interfaces to support new features:
- `SearchResult` → adds `fuzzyMatch`, `similarityScore`
- `SearchRequest` → adds `fuzzySearch`, `contextLines` (required, not optional)
- `SearchState` → adds `fuzzySearch`, `contextLines`

### Component Architecture

**New components:**
- `frontend/src/components/ui/InlineDiffView.vue` - Enhanced result rendering
- `frontend/src/components/ui/SearchHistorySidebar.vue` - Recent searches panel

**Modified components:**
- `frontend/src/components/CodeSearch.vue` - Layout updated with sidebar integration
- `frontend/src/components/ui/SearchForm.vue` - Added fuzzy search toggle
- `frontend/src/components/ui/SearchResults.vue` - Uses InlineDiffView instead of raw HTML
- `frontend/src/composables/useSearch.ts` - Client-side fuzzy filtering logic

---

## Testing

### New Test Files

- `frontend/tests/unit/components/InlineDiffView.spec.ts` (28 tests)
- `frontend/tests/unit/components/SearchHistorySidebar.spec.ts` (30+ tests)
- `frontend/tests/unit/components/CodeSearch.integration.spec.ts` (15 tests)
- Updated `SearchResults.spec.ts` (now uses InlineDiffView assertions)
- Updated `useSearch.spec.ts` (added fuzzy search scenario tests)

### Test Coverage

- **Total frontend tests:** 296 passing
- **Backend tests:** All Go tests pass
- **Build verification:** Production build compiles without errors

---

## Known Limitations

1. **macOS open-in-editor:** Still not implemented (documented in README)
2. **E2E tests:** No Playwright/Cypress tests yet (future work)
3. **Fuzzy accuracy:** Heuristic-based scoring, may vary slightly from human intuition

---

## Migration Guide

**No breaking changes.** Existing functionality preserved:
- Default fuzzy search = OFF (backward compatible)
- Default context lines = 3 (same as before)
- All existing search options continue working identically

**Optional adoption:** Users can enable fuzzy search by checking the box.

---

## Future Work

- [ ] E2E browser testing suite
- [ ] Go-frontend IPC validation tests
- [ ] Fuzzy score calibration studies
- [ ] macOS folder reveal implementation
- [ ] Optional server-side fuzzy matching (for very large corpora)
