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
 - Results marked with `~` badge showing similarity percentage from the frontend's re-scoring

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

**Security:** `renderDiffHtml` escapes each segment with `escapeHtml` before
assembly and sanitizes the result with DOMPurify, so `<`, `>`, and `&` in
matched source lines render as text rather than markup.

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

**ContextLines parameter:** Backend accepts a custom context line count. The
backend default is 2 lines before/after (`defaultContextLines` in
`search_context.go`); the UI seeds 3 and always sends it explicitly — see
*Technical Changes* below.

---

### 5. Symbol Search 🔎

**What it does:** Searches code symbols (functions, classes, variables, consts,
interfaces, types) by name across Go, TypeScript, TSX, JavaScript, Vue, Python,
Rust, Java, C#, and Ruby files under the selected directory.

**How to use:**
- Enter a symbol name to search by name, or click **Load All Symbols** to index every symbol under the directory
- **Re-index** clears the cached index and redoes whatever is on screen — for when files changed outside the app
- Results list each symbol's name, type, signature, and `file:line`
- Requires a selected directory (uses the search form's chosen directory)

**Progress:** Real per-file progress is reported via `symbol-progress` events, driving a live progress bar during indexing.

**Scope:** `symbolSupportedExtensions` in `symbol_scan.go` is the single source
of truth for the ten extensions (`.go .ts .tsx .js .vue .py .rs .java .cs .rb`).
`skipSymbolScanDirs` skips `node_modules`, `.git`, `vendor`, `build`, `dist`,
`bin`, `__pycache__`, `.venv`, `venv`, `.mypy_cache`, `.pytest_cache`,
`target`, `.gradle`, and `obj` — build output, dependency caches, and VCS
metadata never hold user-authored definitions.

**Naming:** a leading `_` is private-by-convention and skipped, but a fully
underscore-wrapped name is kept: Python's `__init__`/`__str__` are real API
surface and prime search targets, `_helper` is not.

**Technical details:**
- Frontend component `frontend/src/components/ui/SymbolSearch.vue` receives the selected directory as a prop and subscribes to `symbol-progress`
- Backed by Wails bindings `GetAllSymbols(directory, maxResults)`, `SearchSymbols(name, directory, maxResults)`, and `ClearSymbolCache()` (the Re-index button)
- Per-language patterns are precompiled once at package init (`goPatterns`, `tsPatterns`, `vuePatterns`, `pyPatterns`, `rustPatterns`, `javaPatterns`, `csPatterns`, `rubyPatterns` in `symbols.go`) and selected by `getPatternsForExtension`
- The persistent index routes every `get`/`set` through `symbolCacheKey()` (`filepath.Abs` + `filepath.Clean`, mirroring `collectionCacheKey`), so `/a/b` and `./b` from `/a` share one of the 8 slots instead of burning two on the same tree

**Limits — the scanner is line-at-a-time regex, not a parser:**
- A declaration split across lines is missed; only the line carrying the name is seen.
- Java and C# patterns require a leading modifier keyword, so members with none (package-private Java, implicitly-private C#) are missed **by design** — a pattern loose enough to catch them also matches every `if (`, `while (`, and `return foo(`.
- A C# property whose `{ get` sits on the next line is missed.
- `attr_accessor :a, :b` captures only `a`. `ponytail:` the extraction loop takes one submatch per pattern per line; upgrade is `FindAllStringSubmatch` in `extractSymbolsFromFile`, worth it only if multi-name declarations prove a real miss.

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
  `fsync` + rename, preserving the original file mode. No `.bak` files; the
  user's VCS is the undo path.
- Every matched path passes `sanitizePath` (path-traversal defense in depth),
  and an `Lstat` re-check refuses to write through a symlink swapped in since
  collection.

**Cancellation & progress:** a replace can take as long as a search and used to
give no feedback at all, so a large one looked frozen.
- `ReplaceInFiles` acquires its context from `a.createSearchContext()` — the
  same machinery and the same stored handle as `SearchWithProgress`. Two
  intended consequences: the collection walk aborts on cancel instead of
  scanning the whole tree, and **`CancelSearch` cancels a running replace too**,
  because both register through one handle.
- Progress arrives on the `replace-progress` event as a `ReplaceProgress`
  (`phase`, `processedFiles`, `totalFiles`, `currentFile`, `filesChanged`,
  `linesChanged`), throttled by the shared `progressEmitInterval` (50ms). The
  terminal event is forced past the throttle so a throttled last in-progress
  event cannot leave the UI showing a short count.
- `phase` is `staging` | `writing` | `cancelled` | `complete`. The two halves
  have very different stakes: **staging writes nothing** and is a clean abort,
  while a **cancel during writing leaves the already-written files on disk**.
  There is still **no rollback** — by design, the user's VCS is the undo path —
  so the returned error names the count (`replace cancelled: 3/9 files written
  before cancellation, no rollback`) alongside a zero `ReplaceResult`. A write
  failure mid-apply reports the same way.

**Technical details:**
- Backend binding `ReplaceInFiles(ReplaceRequest)` in `replace.go`, driven by
  the same `compileSearchPattern` (so case-sensitivity matches search) and the
  same `collectFilesToProcess` (so directory/filter/gitignore behavior matches
  search).
- Frontend `useReplace.ts` composable + controls in `SearchResults.vue`. The
  `replace-progress` subscription is registered per call, not for the
  composable's lifetime, so a stale handler cannot repopulate progress after
  its operation finished.

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
- `get` returns a copy of the cached slice, so callers can sort or append
  without corrupting the shared entry (same for `symbolIndexCache.get`).
- `ponytail:` a hit still pays one fingerprint (metadata) walk to detect
  staleness — the saved work is the binary-probe phase and re-filtering, which
  is real on mixed-extension trees and marginal on pure known-text trees.

### 11. .gitignore Support 📝

**What it does:** When enabled, collection honors the whole chain of
`.gitignore` files from the search root down to each file's own directory, plus
the root's `.git/info/exclude`.

**How to use:** Check **Respect .gitignore** in the search options.

**Semantics** (`gitignore.go`, `ignoreStack` / `ignoreLevel`):
- Pattern syntax stays entirely go-gitignore's job (negation `!`, `**`,
  anchoring, dir-only patterns). `ignoreStack` adds the cross-FILE rules a
  single matcher has no notion of.
- Patterns resolve **relative to the directory holding the `.gitignore` that
  declared them**, not to the search root.
- A **deeper file overrides a shallower one**, and a deeper `!` re-includes
  what an ancestor ignored — git's "applicable patterns, shallowest to deepest,
  last match wins".
- An ignored directory is **pruned** from the walk (`filepath.SkipDir`) rather
  than descended and filtered file by file, mirroring git.
- Filtering is inline in the walk; the old root-only post-pass over the
  collected list is gone.
- Default OFF — behavior is byte-identical to before when unchecked.

**Cost:** one `os.ReadFile` per directory that actually has an ignore file,
never per file. `ignoreStack.chains` memoizes the resolved level list per
directory for the lifetime of one `walkDirectoryTree` call, so a directory
holding N files costs N map lookups. Directories with no rules share their
parent's slice by reference. A pruned subtree costs no `ReadDir` and no
ignore-file reads at all.

**Cache invalidation:** `computeCollectionFingerprint` folds in every nested
`.gitignore` it walks past, so editing `sub/pkg/.gitignore` invalidates the
collection cache — not just the root file.

**Known deviation from git:** pruning is abandoned as soon as an applicable
level negates anything, because `build/*` plus `!build/keep.txt` excludes the
*contents* of `build/` while the regex go-gitignore builds for `build/*`
matches the bare `build/` too — pruning there would swallow the re-included
file. Per-file matching resolves that case exactly, but the consequence is that
a nested negation can re-include a file inside an ignored directory, where git
seals excluded directories. The failure direction is over-inclusion, never a
dropped file. Negation-free trees (the `node_modules`/`vendor` case pruning
exists for) still prune.

- `ponytail:` only ignore files **inside the search tree** are read. A
  repository `.gitignore` above the search directory, the global
  `core.excludesFile` (`~/.config/git/ignore`), and a nested submodule's own
  `.git/info/exclude` are all skipped; the search root's own
  `.git/info/exclude` **is** honored. Upgrade: walk up from the search root to
  the enclosing `.git` directory and prepend those levels to the stack — worth
  it only if searching a subdirectory of a repo needs the repo's own rules.

### 12. Streamed Search Results 📡

**What it does:** Results render as they are found instead of all at once at the
end. Previously the whole result set buffered in the drain loop and the UI saw
nothing until the search finished — a slow search over a big tree showed a
spinner and an empty list the entire time.

**Technical details:**
- `search_engine.go` drains the worker channel through a `resultBatcher`
  (`search_workers.go`), which emits the `search-results` event carrying
  `SearchResultBatch{seq, results}` (`models.go`).
- A batch flushes at `resultBatchSize` (256 results) **or** once
  `progressEmitInterval` (50ms) has elapsed, whichever comes first — so a fast
  search does not emit one IPC event per match and a slow one still renders
  incrementally.
- `seq` is monotonic from 1 per search, so the frontend drops a replayed or
  out-of-order batch (`seq <= lastBatchSeq`) instead of duplicating rows.
- `flush` hands its slice to the payload and sets `pending` to `nil` rather
  than truncating it, so no two batches share a backing array.
- Batches carry **worker-completion order**. `SearchWithProgress`'s return
  value is still the authoritative, deterministically sorted result set; the
  frontend appends batches for immediate feedback and replaces the list with
  the resolved value when the search completes. Streaming also respects
  `maxResults`, which batches can overshoot by at most one batch as workers
  race the cap.

### 13. Skipped-File Reporting ⚠️

**What it does:** Names the files a search could not read. Before this, a search
that could not open half a tree looked identical to one that found nothing.

**Technical details:**
- `SearchProgress` carries `FailedPaths []string` alongside the existing exact
  `FailedFiles` count (`models.go`).
- `SearchState.recordFailure(absPath)` / `snapshotFailedPaths()` are guarded by
  a mutex; the listed sample is capped at `maxFailedPathsReported` (50) so a
  tree with 50k permission-denied files cannot balloon the payload. The
  **count stays exact** — only the list is capped.
- Only the terminal `"completed"` event carries `FailedPaths`: the sample is
  stable by then, and attaching a growing array to every throttled
  in-progress event would re-serialize the same paths dozens of times per
  search.
- `ProgressIndicator.vue` renders the count, the capped path list, and an
  `…and N more` line when the count exceeds the sample; `useSearch` also raises
  a warning toast when `failedFiles > 0`.

### 14. Extension Filter & Backend-Driven Allowed Types 🔌

**What it does:** Closes two dead-wiring gaps where plumbing existed with no
control attached to it.

**Extension filter:** `SearchRequest.extension` had always been threaded
through the request, the search state, and the recent-search history — with no
input anywhere in the UI. A history entry could therefore carry an extension
the user could neither set nor clear. `SearchForm.vue` now renders a
**File Extension** field (reusing `QueryInput` rather than a bespoke control).

**Allowed-types dropdown:** `PatternSelector.vue` hand-maintained a 9-item
`availableAllowOptions` list while the ~170-entry backend set fetched by
`GetKnownTextExtensions` sat unused in search state. The dropdown now takes a
`knownTextExtensions` prop fed from that binding, so it can no longer drift
from what the backend will actually collect. A short hardcoded
`fallbackAllowOptions` covers only the case where the binding fails or a mock
does not implement it. The free-text custom-type input is unchanged and still
accepts multi-dot extensions (`min.js`, `tar.gz`).

See [`EXTENSIONS.md`](EXTENSIONS.md) for the full extension-list architecture.

### 15. Tree Filter 🌲

**What it does:** Filters the preview modal's file-explorer tree by name.
`EnhancedTreeItem` already implemented child filtering, but nothing rendered a
filter input, so the code was unreachable.

**Technical details:**
- `TreeViewPanel.vue` renders a `type="search"` input whose value is committed
  through a 150ms debounce (`FILTER_DEBOUNCE_MS`), since filtering walks the
  whole tree — this retired the previously-unused `debounce` util.
- `filter-text` is passed down to `EnhancedTreeItem`, which filters its own
  children; unmatched **roots** are filtered in `TreeViewPanel` itself,
  otherwise a root with no match still rendered with an empty body.
- A no-match state is distinct from an empty search: `No files match “x”.`
  versus `No files found for this search.`

### 16. Bounded Directory Listing 🚧

**What it does:** `GetDirectoryContents` was an unbounded recursive walk. It now
caps at `maxDirectoryListing` (50 000 directories) and `maxDirectoryDepth` (32
levels below the requested root), and honors `a.ctx` cancellation so app
shutdown aborts the walk instead of holding the IPC call open over a huge tree.

**Technical details** (`system_integration.go`):
- Hitting the listing cap is an **error**, not a truncated success: the
  signature has no room for a "truncated" flag (it is a generated Wails binding
  contract), and a tree view silently missing most of the tree is worse than an
  actionable `choose a narrower directory` message.
- Depth pruning is reported by log only — pruning one pathological subtree
  still leaves a coherent listing.
- The cap is deliberately far below `maxCachedFiles`/`maxSymbolScanFiles`
  (200k): this list crosses the IPC bridge to render a UI tree, so the useful
  ceiling is the renderer's, not memory's.

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
- `search.ts` → `SearchResult`, `SearchRequest`, `SearchProgress` (incl. `failedPaths`), `SearchResultBatch`, `ReplacePhase`, `ReplaceProgress`, `SearchState`, `EditorAvailability`, `EditorDetectionStatus`, `TreeItem`, `SymbolInfo`
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

- `search_streaming_batch_test.go` — batcher sequencing, empty-flush no-op, no shared backing array between batches, failure-sample cap, end-to-end unreadable-file reporting
- `gitignore_nested_test.go` — nested precedence, own-directory-relative patterns, directory pruning, contents-rule negation, gate-off behavior, read-once cost model, fingerprint invalidation
- `app_shared_test.go` — path-validation trust boundary: `sanitizePath` traversal rejection both pre- and post-`Clean`, dots-in-filenames still accepted, `validatePathForEditor` existence check, `validatePathForShowInFolder` parent resolution, `lookUpEditor` absolute-path/TOCTOU contract, `appendPath` shared-backing-array guard, `isSymbolSupportedExtension` across all ten languages, `symbolCacheKey` normalization
- `frontend/tests/unit/components/InlineDiffView.spec.ts`
- `frontend/tests/unit/components/SearchHistorySidebar.spec.ts`
- `frontend/tests/unit/components/CodeSearch.integration.spec.ts`
- `frontend/tests/unit/composables/useSelectionManager.spec.ts` (selection reactivity, select-all, copy/export subset, export-format threading)
- Updated `SearchResults.spec.ts` (now uses InlineDiffView assertions)
- Updated `useSearch.spec.ts` (fuzzy-search scenarios, plus microtask flushes for the async directory-validation guard)

Per-spec test counts are deliberately absent here. They were duplicated across
four documents and nine of them disagreed; the totals below are the only
numbers maintained.

### Test Coverage

- **Total frontend tests:** 714 passing (48 spec files)
- **Backend tests:** all Go tests pass (39 test files), clean under `-race`.
  Total statement coverage measured at 80.0%, which clears the 80% CI gate
  (`.github/workflows/build.yml`).
- **E2E tests:** 41 Playwright flows pass across 7 spec files (search → results
  → preview, symbol search + line-jump navigation, file explorer tree
  navigation, suggestions dropdown, case-sensitivity, diff markers, batch
  export, multi-select, multi-directory, log viewer,
  regex/truncation/theme/clipboard/modal-footer options, pagination, match
  navigation, directory scoping, exclude patterns, fuzzy near-miss candidates
  with badges, find-replace preview + apply)
- **Build verification:** production build compiles without errors
  (`vue-tsc --noEmit` clean)
- `ponytail:` `frontend/vitest.config.ts` declares coverage thresholds
  (lines/functions/statements 80, branches 70) but they are **inert**: `npm
  test` is `vitest run` with no `--coverage`, so nothing computes coverage to
  compare against them. Upgrade: add a `test:coverage` script and run it in CI,
  the way the Go gate already does.

---

## Known Limitations

1. **macOS open-in-editor** relies on editor CLIs being on `PATH` (shared helper); apps without a CLI are opened via the system default. Folder reveal uses `open -R`.
2. **Fuzzy accuracy:** heuristic-based scoring, may vary slightly from human intuition.
3. **Ignore-file scope:** only `.gitignore` files inside the search tree are read (plus the root's `.git/info/exclude`). A repo `.gitignore` above the search directory, the global `core.excludesFile`, and a submodule's own `.git/info/exclude` are not honored — see feature 11 for the upgrade path.
4. **Nested-negation pruning:** when an applicable ignore level negates anything, directory pruning is abandoned, so a nested negation can re-include a file inside an ignored directory where git would seal it. Over-inclusion only; a file is never dropped.
5. **No replace rollback:** a cancel or write failure mid-apply leaves the already-written files written. The returned error names the count; the user's VCS is the undo path.

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
- [x] .gitignore-aware collection (nested per-directory chain + root .git/info/exclude)
- [x] Multi-select copy + batch export CSV/JSON
- [x] Multi-directory search
- [x] Log viewer pause-on-tail + searchable log list
- [x] Progress event throttling (50ms debounce)
- [x] Window minimum size (800×600)
- [x] Table-driven editor dispatch (OpenInEditorByName replaces 17 wrappers)
- [x] Log file rotation (10 MB cap with .1 backup)
- [x] Shared symbol-scan constants (symbol_scan.go single source of truth)
 - [x] Streamed search results (`search-results` batches, seq-ordered)
 - [x] Skipped-file reporting (capped `failedPaths` sample + exact count)
 - [x] Cancellable replace with `replace-progress` phases
 - [x] Symbol search for Python, Rust, Java, C#, Ruby (10 extensions total)
 - [x] Symbol re-index button (`ClearSymbolCache`) + normalized cache key
 - [x] Extension filter input + backend-driven allowed-types dropdown
 - [x] Tree filter in the preview modal's file explorer
 - [x] Bounded `GetDirectoryContents` (50k entries, depth 32, errors on cap)
 - [x] 3-OS CI matrix (`-race` tests on Linux/Windows/macOS) + tag-triggered release job
 - [x] Comprehensive test coverage (714 frontend tests across 48 spec files, 39 Go test files, 80.0% Go statement coverage)
