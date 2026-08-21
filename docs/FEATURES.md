# New Features & Enhancements (v1.x)

## Overview

This document describes recent enhancements added to Code Search, including fuzzy matching, inline diff views, search history sidebar, and performance optimizations.

---

## Feature Highlights

### 1. Fuzzy Search 🔍

**What it does:** Finds matches even with typos or slight misspellings using Levenshtein-style similarity scoring.

**How to use:** 
- Toggle "Fuzzy Search" checkbox in SearchForm
- Backend returns near-miss candidates when enabled (exact matches first, then fuzzy near-misses)
- Results marked with `~` badge showing similarity percentage from the backend's scoring

**Example:** Searching for `"test messgae"` will find files containing `"test message"` with 90%+ similarity.

**Technical details:**
- Implemented in both `frontend/src/utils/fuzzyMatch.ts` (front-end re-scoring/badging) and `search_fuzzy.go` (backend phase-2 near-miss phase)
- Configurable via `data.fuzzySearch` boolean in `SearchRequest`
- Threshold: max(1, floor(query length * 0.6)) — mirrors frontend behavior
- Falls back to exact match when disabled or regex mode is on
- Works alongside existing regex mode (regex disables fuzzy phase)

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

### 9. Find & Replace 🔁

**What it does:** Replaces the search query across every matched line with a
literal replacement string, after a dry-run preview.

**How to use:**
- Run a search, then type a replacement into the **Replace matches with…** input
  in the results header.
- **Preview Replace** runs a dry-run (`Apply=false`) that returns every
  old→new line change and writes nothing.
- **Apply N** writes the changes atomically and re-runs the search so results
  reflect the new file contents.

**Safety:**
- **Literal-only** — regex mode disables the replace row (backend rejects
  regex replace with a clear error).
- **Preview == Apply** — both derive from the same matched `(file, line)` pairs;
  what you preview is exactly what gets written.
- **No-op skip** — lines whose replacement equals the original are never
  written.
- **Atomic writes** — each file is rewritten via a same-directory temp file +
  rename, preserving the original file mode. No `.bak` files; the user's VCS
  is the undo path.
- Every matched path passes `sanitizePath` (path-traversal defense in depth).

**Technical details:**
- Backend binding `ReplaceInFiles(ReplaceRequest)` in `replace.go`, driven by
  the same `compileSearchPattern` (so case-sensitivity matches search) and the
  same `collectFilesToProcess` (so directory/filter/gitignore behavior matches
  search).
- Frontend `useReplace.ts` composable + controls in `SearchResults.vue`.

### 10. Collection Cache ⚡

**What it does:** Repeat searches in an unchanged directory skip the file-collection
walk and binary probe entirely, served from an in-memory cache.

**Technical details:**
- `collection_index.go`: cache keyed by (directory + cheap-filter set),
  validated by a metadata fingerprint (path + size + modtime of every file).
- A cached entry is only reused when the directory AND the request's filter
  dimensions (extension, allowed types, excludes, size bounds, include-binary,
  gitignore) are unchanged — typing a new query with unchanged filters is a
  hit; changing a filter re-walks.
- Capped at 8 entries, oldest evicted; trees over 200k files skip caching.
- `ponytail:` a hit still pays one fingerprint (metadata) walk to detect
  staleness — the saved work is the binary-probe phase and re-filtering, which
  is real on mixed-extension trees and marginal on pure known-text trees.

### 11. .gitignore Support 📝

**What it does:** When enabled, files matched by the search directory's root
`.gitignore` and `.git/info/exclude` are excluded from collection.

**How to use:** Check **Respect .gitignore** in the search options.

**Technical details:**
- Delegates matching to `github.com/sabhiram/go-gitignore`, so real gitignore
  semantics (negation `!`, `**`, anchoring, dir-only patterns) are honored.
- Default OFF — behavior is byte-identical to before when unchecked.
- `ponytail:` root-level only; nested per-directory `.gitignore` files are not
  honored.

---

## Technical Changes

### Backend (`models.go`)

All backend type definitions live in `models.go`. `SearchRequest` carries the
context window and the client-side fuzzy flag:
```go
type SearchRequest struct {
    ContextLines  int      `json:"contextLines"`  // Lines before/after match (2 if unset)
    FuzzySearch   bool     `json:"fuzzySearch"`   // Enables backend fuzzy near-miss phase (60% threshold)
}
```

`ContextLines` is honored by both the small-file path and the line-by-line
streaming path. `searchContextLines` in `search_context.go` resolves the value:
0 (or any value ≤ 0) means "unset" and falls back to `defaultContextLines` (2),
and values above `maxContextLines` (10) are capped at 10, so a request cannot
balloon result payloads with an arbitrarily large context window.

Note the frontend default differs: `useSearch` seeds `data.contextLines` with
`3`, so searches from the UI send an explicit 3 while a JSON payload that
omits the field gets the backend default of 2.
When enabled (`req.FuzzySearch == true`) and regex mode is off, `SearchWithProgress`
runs a second "fuzzy" phase after exact matches, emitting results as it goes.
This phase scores near-miss lines using a sliding-window best-window scorer that
maintains positional character alignment at the configured threshold. When disabled,
the backend still accepts the field in the JSON payload for contract clarity but doesn't run phase-2 scoring.

`UseRegex` is a `*bool` pointer (not a plain `bool`): `nil` means "default to true" for backward compatibility. The frontend always sends a concrete boolean.

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

- `frontend/tests/unit/components/InlineDiffView.spec.ts` (18 tests)
- `frontend/tests/unit/components/SearchHistorySidebar.spec.ts` (26 tests)
- `frontend/tests/unit/components/CodeSearch.integration.spec.ts` (15 tests)
- `frontend/tests/unit/composables/useSelectionManager.spec.ts` (11 tests: selection reactivity, select-all, copy/export subset)
- Updated `SearchResults.spec.ts` (now uses InlineDiffView assertions)
- Updated `useSearch.spec.ts` (added fuzzy search scenario tests)

### Test Coverage

- **Total frontend tests:** 695 passing (46 spec files)
- **Backend tests:** All Go tests pass (30 test files)
- **E2E tests:** 41 Playwright flows pass (search → results → preview, symbol
  search + line-jump navigation, file explorer tree navigation, suggestions
  dropdown, case-sensitivity, diff markers, batch export, multi-select,
  multi-directory, log viewer, regex/truncation/theme/clipboard/modal-footer
  options, pagination, match navigation, directory scoping, exclude patterns,
  fuzzy near-miss candidates with badges, find-replace preview + apply)
- **Build verification:** Production build compiles without errors

---

## Known Limitations

1. **macOS open-in-editor** relies on editor CLIs being on `PATH` (shared helper); apps without a CLI are opened via the system default. Folder reveal uses `open -R`.
2. **Fuzzy accuracy:** Heuristic-based scoring, may vary slightly from human intuition

---

## Migration Guide

**No breaking changes.** Existing functionality preserved:
- Default fuzzy search = OFF (backward compatible)
- Default context lines = 3 from the UI (backend falls back to 2 when the field is unset)
- All existing search options continue working identically

**Optional adoption:** Users can enable fuzzy search by checking the box.

---

## Future Work

- [x] E2E browser testing suite
- [x] E2E: fuzzy search → inline diff view flow
- [x] Go-frontend IPC validation tests
- [x] Fuzzy score calibration studies (measured <5% false-positive rate on 2000 random lines, ~1% actual; single-edit near-miss sensitivity verified)
- [x] macOS folder reveal implementation
- [ ] Optional server-side fuzzy matching (for very large corpora)
- [x] True line-level diff with match-range highlighting + long-line truncation
- [x] Symbol search → code preview navigation (jump to file:line)
- [x] Persistent symbol index (fingerprint-based cache)
- [x] Persistent file-collection cache (fingerprint-based, filter-aware)
- [x] Find & Replace (literal, dry-run preview + atomic apply)
- [x] .gitignore-aware collection (root + .git/info/exclude)
- [x] Multi-select copy + batch export CSV/JSON
- [x] Multi-directory search
- [x] Log viewer pause-on-tail + searchable log list
- [x] Progress event throttling (50ms debounce)
- [x] Window minimum size (800×600)
- [x] Table-driven editor dispatch (OpenInEditorByName replaces 17 wrappers)
- [x] Log file rotation (10 MB cap with .1 backup)
- [x] Shared symbol-scan constants (symbol_scan.go single source of truth)
- [x] Comprehensive test coverage (684 frontend + 30 backend test files)
