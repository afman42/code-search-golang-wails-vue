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

### 5. Symbol Search 🔎

**What it does:** Searches code symbols (functions, classes, variables, consts, interfaces, types) by name across Go/TS/TSX/JS/Vue files under the selected directory.

**How to use:**
- Enter a symbol name to search by name, or click **Load All Symbols** to index every symbol under the directory
- Results list each symbol's name, type, signature, and `file:line`
- Requires a selected directory (uses the search form's chosen directory)

**Progress:** Real per-file progress is reported via `symbol-progress` events, driving a live progress bar during indexing.

**Scope:** Skips `node_modules`, `.git`, `vendor`, `build`, `dist`, and `bin` directories.

**Technical details:**
- Frontend component `frontend/src/components/ui/SymbolSearch.vue` receives the selected directory as a prop and subscribes to `symbol-progress`
- Backed by Wails bindings `GetAllSymbols(directory, maxResults)` and `SearchSymbols(name, directory, maxResults)`

---

### 6. Search Progress & Modal Fixes 🩹

**"Searching…" overlay:** A spinner with a processed/total file count is shown while a search runs, giving clear feedback on long searches.

**File-preview modal fix:** The preview modal now only renders when open (previously it was always mounted, which could cover the screen). It is guarded so it mounts only while visible.

---

### 7. Design System & Responsive Layout 🎨

**What it does:** Establishes a shared set of CSS custom properties and migrates every component to it, so the UI is visually consistent and re-themeable from one place.

**Design-system tokens** (`frontend/src/style.css` `:root`):
- **Colors** — neutral palette, accent, success/danger/warning/info, dark-surface tokens (log viewer, history sidebar, dropdowns), sidebar tokens, code-preview tokens
- **Spacing scale** (`--space-*`), **radii** (`--radius-*`), **shadows** (`--shadow-*`), **font sizes**, and **transition durations**
- **Responsive breakpoints** — `--modal-max-width`/`--modal-max-height` scale down on small screens

**Component migration:** All `frontend/src/components/` styles use tokens instead of hard-coded hex/RGBA colors. Intentional dark surfaces (log viewer content, recent-search sidebar, suggestions dropdown) use dedicated dark-surface tokens; log-level colors and semantic diff colors are preserved as content styling.

**Responsive app grid** (`CodeSearch.vue`): The layout uses CSS grid with named areas (`sidebar` / `main`). The search-history sidebar is sticky and full-height while results scroll, and the whole layout stacks into a single column below 768px.

**CodeModal file preview improvements:**
- Working **line-number toggle** — hides/shows line numbers without re-mounting the highlight system
- **Match navigation** — prev/next buttons (also `Ctrl+↑` / `Ctrl+↓`) with a clamped `N / total` counter and correct disabled states at the boundaries
- **Jump-to-line flash** — a highlight pulse on the target line (uses `:deep()` so it works on highlighted content)
- Load placeholder shown only while the file is being processed (mutually exclusive with the highlighted code)

### 8. File Explorer Tree & Search Suggestions 🗂️

**File Explorer tree** (replaces the previous placeholder in `TreeViewPanel.vue`)
- The file-preview modal now has a working **Tree View** toggle. It renders a
  file explorer of every file the current search matched, built from the result
  paths (handles `/` and `\` separators, folders grouped before files, sorted
  alphabetically at each level).
- Directories expand/collapse in place (powered by the recursive
  `EnhancedTreeItem.vue`), with per-folder item counts and the currently-open
  file highlighted.
- Clicking any file loads it into the preview (via the `ReadFile` binding) and
  switches back to the file tab, including match highlighting for the active query.

**Recent-search suggestions dropdown** (`SearchSuggestions.vue`)
- Focus the query input to see recent searches stored in browser `localStorage`.
- The dropdown closes on **outside click** or **Escape** (document-level
  listeners that are cleaned up on unmount); it also hides when the input blurs.
- Selecting a suggestion fills the query and runs the search; deleting one
  removes it from both `localStorage` and the recent-search sidebar.

---

## Technical Changes

### Backend (`models.go`)

All backend type definitions live in `models.go`. `SearchRequest` carries the
context window; fuzzy filtering is purely client-side, so it is intentionally
absent from the Go request model:
```go
type SearchRequest struct {
    // ... existing fields ...
    ContextLines  int      `json:"contextLines"`  // Lines before/after match (default 3)
}
```

`ContextLines` is honored by both the small-file path and the line-by-line
streaming path, clamped to 1–10 (`searchContextLines` in `search_engine.go`).
A value of 0 means "use the default" (2 lines), which preserves behavior for
callers that leave the field unset.

### Frontend Types (`frontend/src/types/`)

All shared TypeScript types are centralized under `frontend/src/types/`.
- `search.ts` → `SearchResult`, `SearchRequest`, `SearchProgress`, `SearchState`, `EditorAvailability`, `EditorDetectionStatus`, `TreeItem`, `SymbolInfo`
- `recentSearch.ts` → `RecentSearch` (query + extension + directory)
- `logs.ts` → `LogEntry`
- `toast.ts` → `Toast`, `ToastOptions`, `ToastStore`
- `keyboard.ts` → `KeyboardShortcutHandlers`
- `syntax.ts` → `SyntaxHighlightOptions`

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

- **Total frontend tests:** 373 passing (26 spec files)
- **Backend tests:** All Go tests pass
- **E2E tests:** 9 Playwright flows pass (search → results → preview, symbol
  search, file explorer tree navigation, suggestions dropdown, case-sensitivity)
- **Build verification:** Production build compiles without errors

---

## Known Limitations

1. **macOS open-in-editor** relies on editor CLIs being on `PATH` (shared helper); apps without a CLI are opened via the system default. Folder reveal uses `open -R`.
2. **Fuzzy accuracy:** Heuristic-based scoring, may vary slightly from human intuition

---

## Migration Guide

**No breaking changes.** Existing functionality preserved:
- Default fuzzy search = OFF (backward compatible)
- Default context lines = 3 (same as before)
- All existing search options continue working identically

**Optional adoption:** Users can enable fuzzy search by checking the box.

---

## Future Work

- [x] E2E browser testing suite
- [ ] E2E: fuzzy search → inline diff view flow
- [ ] Go-frontend IPC validation tests
- [ ] Fuzzy score calibration studies
- [x] macOS folder reveal implementation
- [ ] Optional server-side fuzzy matching (for very large corpora)
