# Architecture

This document describes how the Code Search application is structured. It is a Wails desktop app: a Go backend handles file-system operations, a Vue 3 + TypeScript frontend renders the UI, and Wails generates type-safe bindings between them.

## Overview

Three channels connect the frontend and backend:

1. **Wails bindings** — direct type-safe calls from Vue into exported Go methods (`SearchWithProgress`, `SelectDirectory`, `GetInitialLogs`, etc.).
2. **Wails events** — `EventsOn` / `EventsEmit` for search progress, streamed search-result batches, replace progress, symbol-scan progress, and editor detection.
3. **Log composable** — `useLogStreaming()` calls `GetInitialLogs()` / `GetNewLogs()` Wails bindings to stream log entries via IPC (no HTTP server).

```
┌──────────────────┐   Wails bindings + events       ┌──────────────────┐
│  Vue 3 frontend  │ ◄──────────────────────────────► │   Go backend     │
│  (UI / state)    │   (SearchWithProgress,            │ (search engine)  │
│                   │    SelectDirectory,              │                  │
│  composables:     │    GetInitialLogs, ...)          │                  │
│   useSearch       │                                  │                  │
│   useLogStreaming │                                  └──────────────────┘
│   useToast        │                                        │
└──────────────────┘                                         │ file system
                                                             ▼
                                                   ┌──────────────────┐
                                                   │  log file tail   │
                                                   └──────────────────┘
```

No HTTP polling server is involved. Log entries are delivered to the frontend via Wails IPC bindings (`GetInitialLogs`, `GetNewLogs`), avoiding CORS and mixed-content issues that arise in production Wails builds.

---

## Backend

### Source files

| File                     | Responsibility |
| ------------------------ | -------------- |
| `main.go`                | Entry point. Creates the app, ensures `logs/` directory, starts log file tailing, runs Wails (title `code-search-golang`, 1024×768, min 800×600). |
| `app_core.go`            | `App` struct, `NewApp`, search-cancel helpers, shutdown, `ReadFileLog`, `GetInitialLogs`, `GetNewLogs`, LRU pattern cache. |
| `app_symbols.go`         | Symbol-search Wails bindings: `GetAllSymbols(directory, maxResults)` and `SearchSymbols(name, directory, maxResults)`. Delegates to `symbols.go` and emits `symbol-progress` events during a full scan. Activates the persistent symbol index. |
| `symbol_scan.go`         | Shared constants for symbol extraction: `skipSymbolScanDirs` (`node_modules`, `.git`, `vendor`, `build`, `dist`, `bin`, `__pycache__`, `.venv`, `venv`, `.mypy_cache`, `.pytest_cache`, `target`, `.gradle`, `obj`), `symbolSupportedExtensions` (`.go`, `.ts`, `.tsx`, `.js`, `.vue`, `.py`, `.rs`, `.java`, `.cs`, `.rb`), `maxSymbolScanFiles` (200k), and helpers `isSymbolSupportedExtension()` / `shouldSkipDirForSymbolScan()`. Single source of truth — previously duplicated in `symbols.go` and `symbol_index.go`. |
| `models.go`              | All backend type definitions: `SearchRequest` (including `FuzzySearch` flag enabling near-miss phase and `UseRegex *bool` pointer for backward compat), `SearchResult`, `SearchProgress` (including `FailedPaths`, the capped sample of unreadable files), `SearchState` (plus `recordFailure` / `snapshotFailedPaths`, mutex-guarded), `SearchResultBatch` (streamed result slice), `ReplaceProgress`, `EditorAvailability`, `SymbolInfo`, `LogMessage`, `PollingLogManager`, `App`, `LRUPatternCache`, `symbolIndexCache`, `collectStats`, `fileMeta`. |
| `symbols.go`             | Symbol-extraction engine: `GetAllSymbols`, `SearchSymbols`, `GetAllSymbolsWithProgress` (two-pass scan via `filepath.WalkDir`). Checks the persistent symbol index (`symbol_index.go`) before extracting; stores results on cache miss. |
| `symbol_index.go`        | Persistent symbol index: `symbolIndexCache` (in-memory, per-directory, keyed by file fingerprint = path+size+mtime hash). `computeDirectoryFingerprint`, `symbolCacheKey` (`filepath.Abs` + `Clean` normalization applied inside `get`/`set`, mirroring `collectionCacheKey` so `/a/b` and `./b` are one entry, not two), `ClearSymbolCache` binding. Caps at 8 directories. |
| `search_engine.go`       | `SearchWithProgress` orchestration, `createSearchContext`, `CancelSearch`, `numCPU`. Multi-directory collection (deduped). Cancelled searches return empty (no misleading "completed"). |
| `search_workers.go`      | Worker pool: `processFilesWithWorkers`, `workerShouldContinue`, `processFile`, `emitFileResults`, `emitFileProgress` (progress events throttled to 50ms via CAS). Also `resultBatcher` — accumulates drained results and emits `search-results` batches at `resultBatchSize` (256) or every `progressEmitInterval`, whichever first — and `maxFailedPathsReported` (50), the cap on the listed failure sample. Results sorted by path+line. |
| `search_streaming.go`    | Line-by-line streaming path for files > 1 MB: `processFileLineByLine` + `streamingThreshold`. |
| `search_fuzzy.go`        | Fuzzy near-miss phase: threshold calculation (`max(1, floor(len*0.6))`), best-window sliding scoring, exact-match exclusion in fuzzy phase, regex gating, quota enforcement capped by maxResults. Called from `SearchWithProgress` when `FuzzySearch && !UseRegex && results < cap`. |
| `search_context.go`      | Context-window helpers shared by both search paths: `searchContextLines`, `safeContextLinesBytes`, `bytesToStrings`, and the `binaryCheckBufPool` scratch-buffer pool. |
| `export.go`              | `ExportSearchResults` Wails binding — opens a native `SaveFileDialog` and writes CSV or JSON. `renderResultsCSV` is a pure helper. |
| `file_collection.go`     | Two-phase file collection: `walkDirectoryTree` (single-threaded walk + cheap filters) and `probeBinaryInParallel` (worker pool for binary detection on unknown extensions). |
| `collection_index.go`    | Persistent collection cache: fingerprint-validated, keyed by directory + filter-set; repeat searches with unchanged filters skip the walk + binary probe. `computeCollectionFingerprint` folds in every nested `.gitignore` it walks past, so editing `sub/pkg/.gitignore` invalidates the cached collection. |
| `gitignore.go`           | Nested `.gitignore` support via the unexported `ignoreStack` / `ignoreLevel` types: the chain of ignore files from the search root down to each file's own directory, deeper overriding shallower, `!` negation re-including, patterns matched relative to their own `.gitignore`'s directory. Directory pruning via `filepath.SkipDir`. Gated by `SearchRequest.RespectGitignore`; pattern syntax stays go-gitignore's job. `loadGitignoreMatcher` + `filterByGitignore` remain as the flat single-directory helpers. See [Nested `.gitignore`](#nested-gitignore) below. |
| `replace.go`             | `ReplaceInFiles` binding — literal replace across matched lines, dry-run (`Apply=false`) vs atomic apply, reusing `compileSearchPattern` + `collectFilesToProcess`. Cancellable and progress-reporting: acquires its context via `a.createSearchContext()` and emits `replace-progress`. See [Replace cancellation](#replace-cancellation) below. |
| `text_extensions.go`     | Set of ~170 known-text extensions (.go, .ts, .py, .md, .vue, .toml, .txt, etc.) that skip the binary detection probe entirely. Exposes `GetKnownTextExtensions()` — a Wails binding the frontend loads into search state. See [`EXTENSIONS.md`](EXTENSIONS.md). |
| `system_integration.go`  | Directory dialog, directory validation, file reading, editor detection (22 editors), the `editorCatalog` table, and the sole `OpenInEditorByName` launch dispatcher. `GetDirectoryContents` is bounded by `maxDirectoryListing` (50,000) and `maxDirectoryDepth` (32). |
| `logger_utils.go`        | Logger setup (with size-based log rotation at 10 MB), `isBinary` (zero-allocation), `matchesPattern` (path-component matching), `validateAndSetDefaults`, `safeEmitEvent` (scoped panic recovery), `rotateLogFileIfNeeded`. |
| `polling_server.go`      | `PollingLogManager` — in-memory log buffer, file tailing, noise filtering. No HTTP server. Entries are consumed by the frontend via Wails IPC bindings. |
| `app.go`                 | Linux build (`//go:build linux`): `ShowInFolder` (`xdg-open`), `openInEditor` helper, `OpenInDefaultEditor` (`xdg-open`, path-validated). |
| `appWindows.go`          | Windows build (`//go:build windows`): `ShowInFolder` (`explorer`), `openInEditor` helper, `OpenInDefaultEditor` (`ShellExecute`, no shell parsing — injection-safe). |
| `appDarwin.go`           | macOS build (`//go:build darwin`): `ShowInFolder` (`open -R`), `openInEditor` helper, `OpenInDefaultEditor` (`open`). |
| `app_shared.go`          | Cross-platform editor-launch plumbing shared by the three build files: `validatePathForEditor` / `validatePathForShowInFolder` (empty-path + `..`-component checks on the raw input, then `filepath.Clean`), `lookUpEditor` (PATH probe), `runCommand`/`startAndReap` (Start + async Wait so short-lived helpers are reaped, no zombies), and `appendPath` (copies shared `editorCatalog` args so concurrent launches can't alias one backing array). |

### App struct

```go
type App struct {
    ctx              context.Context
    logger           *logrus.Logger
    searchMu         sync.Mutex
    searchCancel     context.CancelFunc
    editorsMu        sync.RWMutex
    availableEditors EditorAvailability
    ready            int32          // Set atomically after startup()
    patternCache     *LRUPatternCache   // LRU cache for compiled regex patterns
    symbolIndex      *symbolIndexCache  // Cached symbol indices per directory
}
```

### Search engine

`SearchWithProgress` is the core entry point:

- **Worker pool** sized to available CPU cores processes files concurrently.
- **Streaming**: files > 1 MB are read line-by-line with a 1 MB scanner buffer (flat memory usage).
- **Early termination**: once `MaxResults` is reached, the search context is cancelled and workers stop.
- **Deterministic output**: results are sorted by file path then line number before being returned, so ordering is stable regardless of worker completion order.
- **Streamed batches**: the drain loop feeds every result into a `resultBatcher` (`search_workers.go`) that emits `search-results` events as the search runs, so the UI renders progressively instead of waiting for the whole search. The invariant: **batches carry worker-completion order, the returned slice carries deterministic sorted order.** The returned slice remains the authoritative result set; the frontend appends batches for immediate feedback and then replaces the streamed rows with the resolved binding value when the search completes. Batches flush at `resultBatchSize` (256 results) or every `progressEmitInterval` (50 ms), whichever comes first, and `Seq` is monotonic from 1 per search so a replayed or out-of-order batch can be dropped rather than duplicating rows.
- **Cancel semantics**: if the context is cancelled before hitting the result limit, `SearchWithProgress` returns an empty slice and does *not* emit a misleading `completed` event — only the `cancelled` event from `CancelSearch` is emitted.
- **Progress**: counts and percentages are emitted via Wails events.
- **Failed-file reporting**: a file that cannot be read is recorded via `SearchState.recordFailure` — `FailedFiles` is an exact count, while `FailedPaths` is a mutex-guarded sample capped at `maxFailedPathsReported` (50). Only the terminal `completed` event carries `FailedPaths`; attaching a growing array to every throttled in-progress event would re-serialize the same paths dozens of times per search. A search that silently skipped unreadable files used to look identical to one that found nothing there.
- **Binary detection**: `isBinary` reads the first 512 bytes — files with null bytes or < 50% printable characters are skipped unless `IncludeBinary` is set.

### Symbol search

`GetAllSymbols` / `SearchSymbols` (`symbols.go`, exposed as Wails bindings in `app_symbols.go`) extract symbol definitions — functions, classes, variables, constants, interfaces, types — from source files. Ten languages are supported: Go (`.go`), TypeScript (`.ts`/`.tsx`), JavaScript (`.js`), Vue (`.vue`), Python (`.py`), Rust (`.rs`), Java (`.java`), C# (`.cs`), and Ruby (`.rb`). Each has a precompiled pattern set in `symbols.go` (`goPatterns`, `tsPatterns`, `vuePatterns`, `pyPatterns`, `rustPatterns`, `javaPatterns`, `csPatterns`, `rubyPatterns`), dispatched by `getPatternsForExtension`; `GetSymbolType` maps the matched keyword to a category (`fn`/`def`/`method` → function, `enum`/`trait`/`record`/`module`/`impl` → class, `property`/`attr` → variable). Build outputs, dependency caches, and VCS metadata are skipped via `skipSymbolScanDirs` (`node_modules`, `.git`, `vendor`, `build`, `dist`, `bin`, `__pycache__`, `.venv`, `venv`, `.mypy_cache`, `.pytest_cache`, `target`, `.gradle`, `obj`).

Underscore-prefixed names are skipped as private, but the skip is scoped so Python dunders survive: `__init__` is kept while `_helper` is not (`symbols.go:243-246`).

`GetAllSymbolsWithProgress` performs a **two-pass scan**: it first enumerates all supported source files so the total is known up front, then extracts symbols file by file, invoking a progress callback after each file. `app_symbols.go` forwards that callback as a `symbol-progress` Wails event (`{processed, total, currentFile}`), so the frontend `SymbolSearch.vue` panel renders a real progress bar rather than a synthetic one.

### File collection (two-phase)

The collection phase (`collectFilesToProcess` in `file_collection.go`) is split into two phases for performance:

**Phase 1 — `walkDirectoryTree`** (single-threaded directory walk):

Walks the directory tree with `filepath.WalkDir` and applies cheap filters (extension, size, exclude patterns, nested `.gitignore`). Files are split into two slices:
- `textCandidates` — files with known-text extensions (skip binary probe) or `IncludeBinary=true`
- `binaryCheckCandidates` — files with unknown extensions that need the 512-byte binary probe

Optimizations applied during the walk:
- **Absolute base computed once**: `filepath.Abs(req.Directory)` is called once before the walk, not per file. Each file's `absPath` is resolved via `filepath.Clean` (absolute paths) or `filepath.Join(cwd, path)` (relative paths) — no per-file syscall.
- **Prefix-based traversal check**: replaces the per-file `filepath.Rel` + `..` check with a `strings.HasPrefix(absPath, baseDir + separator)` check — zero allocations.
- **Known-text extension shortcut**: ~170 text extensions (`.go`, `.ts`, `.py`, `.md`, `.json`, `.vue`, `.toml`, `.txt`, etc.) are recognized via `text_extensions.go`. Files with these extensions skip the binary probe entirely — no `open` + `read` + `close` syscall. The set is exposed to the frontend via the `GetKnownTextExtensions()` binding and loaded into search state (see [`EXTENSIONS.md`](EXTENSIONS.md)).
- **Inline `.gitignore` filtering**: when `RespectGitignore` is set, the walk consults an `ignoreStack` per entry and prunes ignored directories with `filepath.SkipDir` (`file_collection.go:90-93`, `:126-138`). There is no longer a post-pass in `collectFilesToProcess` — the old root-only `filterByGitignore` call is gone (`file_collection.go:537-540`), which also keeps ignored files out of the binary-probe phase entirely instead of probing files git never looks at.

**Phase 2 — `probeBinaryInParallel`** (worker pool):

If `binaryCheckCandidates` is non-empty, a worker pool (sized to CPU count) runs the 512-byte binary detection probe on each candidate in parallel. Each worker reuses a pooled 512-byte buffer. Files that pass the probe (are text) are appended to the final list; binary files are counted as skipped.

On a tree of 2000 `.go` files (all known-text), Phase 2 is empty and the walk is the only cost. On a mixed tree with unknown extensions, Phase 2 parallelizes the binary probes across CPU cores.

**Benchmark impact** (Celeron N4000, 2 cores, 2000 `.go` files):

| Benchmark | Single-pass | Two-phase | Improvement |
|-----------|-------------|-----------|-------------|
| `CollectFilesToProcess` | 98 ms, 18772 allocs | 27 ms, 12779 allocs | **3.6x faster, 32% fewer allocs** |
| `SearchWithProgress` | 200 ms, 33781 allocs | 127 ms, 27782 allocs | **1.6x faster, 18% fewer allocs** |

### Nested `.gitignore`

`gitignore.go` resolves the whole chain of ignore files from the search root down to each file's own directory. Two unexported types carry it:

- **`ignoreLevel`** — one directory's compiled rules, plus the separator-terminated prefix to strip so patterns match relative to *that* directory rather than the search root. The prefix is computed once per level, so per-file matching allocates nothing.
- **`ignoreStack`** — resolves the effective verdict for a path from the levels between the search root and the path's own directory.

Precedence follows git: `ignoredIn` folds the chain shallowest level first, and each level that has an opinion overrides the previous one (`gitignore.go:136-149`). A deeper file therefore overrides a shallower one, and a deeper `!negation` re-includes what an ancestor ignored. Only the search root contributes `.git/info/exclude`, because git reads it once per repository rather than once per directory (`gitignore.go:179-187`).

**Directory pruning.** `ignoresDir` returns `filepath.SkipDir` for an ignored directory, which mirrors git (nothing under an excluded directory can be re-included) and is also the saving: a pruned subtree costs no `ReadDir` and no ignore-file reads. Only the *parent's* chain is consulted — a directory's own `.gitignore` cannot ignore the directory itself, and not reading it keeps a pruned subtree's ignore file unopened.

**Known deviation from git.** Pruning is abandoned as soon as any applicable level negates anything (`gitignore.go:112-123`). `build/*` plus `!build/keep.txt` excludes the *contents* of `build/` while keeping one file, yet the regex go-gitignore compiles for `build/*` matches the bare `build/` too — pruning there would swallow the re-included file. Per-file matching resolves that case exactly, so the walk descends instead. The consequence: a nested negation can re-include a file inside a directory git would have sealed. The failure direction is over-inclusion, never a dropped file, and negation-free trees (the `node_modules`/`vendor` case pruning exists for) still prune.

**Cost model.** `ignoreStack.chains` memoizes the resolved level list per directory, so each `.gitignore` is read and compiled exactly once per walk and a directory holding N files costs N map lookups rather than N reads (`gitignore.go:153-175`). A directory with no rules shares its parent's slice by reference; only directories that carry rules allocate. The cache's lifetime is a single `walkDirectoryTree` call, and because `filepath.WalkDir` invokes its callback sequentially it needs no lock. Nothing is read until a path is actually tested, so a cancelled or empty walk opens no ignore file at all.

**Cache invalidation.** `computeCollectionFingerprint` folds in every nested `.gitignore` it walks past, so editing `sub/pkg/.gitignore` invalidates the cached collection rather than serving files the new rule excludes.

**Ceiling** (recorded as a `ponytail:` note at `gitignore.go:24-29`): only ignore files *inside* the search tree are read. A repository `.gitignore` above the search directory, the global `core.excludesFile` (`~/.config/git/ignore`), and a nested submodule's own `.git/info/exclude` are all skipped; the root's `.git/info/exclude` is honored. Upgrade path: walk up from the search root to the enclosing `.git` directory and prepend those levels to the stack — worth it only if searching a subdirectory of a repo needs the repo's own rules.

### Replace cancellation

`ReplaceInFiles` acquires its context through `a.createSearchContext()` and releases it with `defer clearSearchCancel(handle); cancel()` (`replace.go:86-90`) — the same machinery, and the same stored handle, that `SearchWithProgress` uses. Two consequences, both intended:

- The collection walk aborts on cancel instead of scanning the whole tree (replace passes the context to `collectFilesToProcess`, where it previously passed `context.Background()`).
- **`CancelSearch` cancels a running replace**, because both register through the same slot. The handle is last-writer-wins and cleared by pointer identity (`app_core.go:98-109`), so a replace started during a search takes over that search's cancel slot, and each operation clears the slot only while it still owns it.

Cancellation is checked after collection, in the staging loop, and in the write loop. Staging writes nothing, so a cancel there is a clean abort: zero files touched. A cancel mid-write returns a zero `ReplaceResult` plus an error naming how many files were written before the abort (`replace.go:238-245`) — there is no rollback by design, the user's VCS is the undo path.

Progress rides the new `replace-progress` event carrying `ReplaceProgress{phase, processedFiles, totalFiles, currentFile, filesChanged, linesChanged}`, where `phase` is `staging`, `writing`, `cancelled`, or `complete`. Replace stages and writes on one goroutine, so a plain last-emit timestamp is equivalent to the CAS throttle `emitFileProgress` needs; `progressEmitInterval` is reused rather than re-declared so replace and search pace identically. The terminal event is forced past the throttle so a throttled last in-progress event cannot leave the UI showing a short count.

### File-extension system

The app tracks file extensions in three places. Full details live in [`EXTENSIONS.md`](EXTENSIONS.md); the summary:

- **Known-text set** (`text_extensions.go` → `knownTextExtensions`) — ~170 extensions that skip the binary probe. The single source of truth for "is this file text?"
- **Allow-list UI** (`PatternSelector.vue`) — the "Allowed file types" dropdown is fed the backend set: `useSearch.ts` calls `GetKnownTextExtensions()` into `data.knownTextExtensions`, `SearchForm.vue` passes it down as the `knownTextExtensions` prop, and the component renders it (dotted). A short hardcoded `fallbackAllowOptions` list applies only when the binding fails, so the dropdown stays usable offline. The two lists are no longer independent (see [`EXTENSIONS.md`](EXTENSIONS.md)).
- **Language detection** (`syntaxHighlightingService.ts` → `detectLanguage()`) — a separate map from extension to highlight.js language name, because the question "which highlighter?" is independent of "is this text?". Not every text extension has a highlight.js language; unmapped extensions fall back to plain text in the preview modal.

### System integration

- **Directory selection**: uses the cross-platform Wails `OpenDirectoryDialog`.
- **Editor detection**: probes 22 editor commands in parallel via `exec.LookPath`. Detected editors include VS Code, VSCodium, Sublime, Geany, JetBrains IDEs (GoLand, PyCharm, IntelliJ, WebStorm, PhpStorm, CLion, Rider — routed by file extension), Android Studio, Emacs, Neovim, Neovide, Vim, Code::Blocks, Dev-C++, Notepad++, Visual Studio, Eclipse, NetBeans.
- **Open-in-editor**: the sole Wails binding is `OpenInEditorByName(name, filePath)`, a table-driven dispatcher backed by the `editorCatalog` (command + args per editor). The `"JetBrains"` catalog entry is a special case that routes to the appropriate JetBrains IDE via `getJetBrainsEditor` (file-extension-based). The previous 17 per-editor `OpenInX` wrapper methods were removed in favor of this single dispatcher.
- **Show in folder**: Linux uses `xdg-open`, Windows uses `explorer`, and macOS uses `open -R` (Finder reveal).

### Log streaming (Wails bindings + composable)

The frontend LogViewer uses two Wails bindings on the `App` struct, consumed through the `useLogStreaming` composable:

- **`GetInitialLogs()`** — returns the last 20 entries from the polling manager's in-memory buffer (called on mount).
- **`GetNewLogs()`** — returns entries added since the last call (polled on a 1-second interval while streaming is active). Each call advances a per-manager read cursor.

The `useLogStreaming` composable (`frontend/src/composables/useLogStreaming.ts`) encapsulates:
- Log parsing helpers (resolve structured JSON, filter noise, extract level/message/timestamp)
- Polling interval management (start/stop/toggle)
- Reactive state (`logs`, `previewLogs`, `isStreaming`, `filteredLogs`)
- Lifecycle hooks (auto-start on mount, auto-stop on unmount)
- An exported `parseLogEntry()` function for direct use in templates and tests

The `LogViewer.vue` component is a thin wrapper that calls the composable and wires the result to the template.

### Log buffer management

`PollingLogManager` manages the in-memory log buffer. It tails `logs/app.log` with `github.com/nxadm/tail` and maintains:

- Bounded buffer (max ~1000 entries, trimmed to ~750) to prevent memory bloat.
- **On-disk log rotation**: `rotateLogFileIfNeeded` renames `logs/app.log` to `logs/app.log.1` (overwriting any previous rotation) when the file exceeds 10 MB at startup. This bounds disk usage to ~2× the cap on long-running installs.
- Noise filtering: messages containing `Skipping` or `Sending file` are dropped (these are per-file progress lines that flood the log during search and add no value in the UI).
- No HTTP server — entries are delivered to the frontend via Wails IPC bindings.

---

## Frontend

Vue 3 + TypeScript, built with Vite. State and search logic live in composables; UI is split into focused components.

### Components

| Component              | Role |
| ---------------------- | ---- |
| `App.vue`              | Root shell. |
| `CodeSearch.vue`       | Main orchestrator — composes the search UI. |
| `StartupLoader.vue`    | Loading state during initialization. |
| `ui/SearchForm.vue`    | Search parameters, validation, recent searches dropdown. Composed of modular child components: `ActionButtons`, `DirectoryPicker`, `QueryInput`, `SearchOptions`, `SizeLimitOptions`, `PatternSelector`, `EditorStatusDisplay` (plus `SearchSuggestions.vue`, `TreeViewPanel.vue`). |
| `ui/SearchResults.vue` | Paginated results (10/page) with copy, open-in-editor, and file-reveal actions. |
| `ui/ProgressIndicator.vue` | Real-time progress bar and status. |
| `ui/CodeModal.vue`     | File preview modal with syntax highlighting, match navigation (prev/next + `Ctrl+↑`/`Ctrl+↓`), jump-to-line with flash highlight, working line-number toggle, tree view. Large files capped at 10,000 lines. |
| `ui/LogViewer.vue`     | Collapsible log viewer at the bottom of the screen. Uses `useLogStreaming` composable for all streaming logic. |
| `ui/ToastNotification.vue` | Toast notifications with auto-dismiss and pause-on-hover. |
| `ui/EnhancedTreeItem.vue` | Recursive file-tree component with filtering and expand/collapse. |
| `ui/SymbolSearch.vue`  | Symbol-search panel — name search and "Load All Symbols". Receives a `:directory` prop from `CodeSearch.vue` (the search form's selected directory) and subscribes to `symbol-progress` events to drive its progress bar. |

### Design system

`frontend/src/style.css` defines the design-system tokens in `:root`:

- **Color tokens** — neutral palette, accent palette (+ `--color-accent-rgb` for `rgba()` reuse), success/danger/warning/info, and dedicated dark-surface tokens (`--color-surface-dark*`, `--color-border-dark`, `--color-text-dark*`), sidebar tokens, and code-preview tokens.
- **Spacing / radius / shadow / typography / transition tokens** — `--space-*`, `--radius-*`, `--shadow-*`, `--font-size-*`, `--transition-*`.
- **Modal/responsive tokens** — `--modal-max-width` / `--modal-max-height`, tightened at `640px` and `768px` breakpoints.

All component `<style>` blocks consume these tokens instead of hard-coded colors, so the palette and spacing are re-themeable centrally. Intentional dark panels (LogViewer content, SearchHistorySidebar, SearchSuggestions) keep their dark look via the dark-surface tokens; log-level and semantic diff colors remain content styling.

**App layout grid:** `CodeSearch.vue` uses CSS grid with named areas (`sidebar` / `main`). The history sidebar is sticky and full-height while the content column scrolls; below `768px` the grid collapses to a single column that stacks the sidebar above the main content.

### Composables

| Composable | Responsibility |
| ---------- | -------------- |
| **`useSearch.ts`** | Central search state, calls Wails backend, handles progress events, and `localStorage` persistence of recent searches. On startup it also calls `GetKnownTextExtensions()` to populate `data.knownTextExtensions` (loaded for reference; the allow-list UI renders its own curated list in `PatternSelector.vue`). Editor detection has been extracted into `useEditorDetection.ts` (separation of concerns). |
| **`useLogStreaming.ts`** | Encapsulates log streaming: Wails binding calls, log parsing, polling interval management, and lifecycle hooks. Exports `parseLogEntry()` for reuse. |
| **`useToast.ts`** | Reactive toast notification system with auto-dismiss, pause/resume with accurate remaining-time tracking, and convenience methods (`success`, `error`, `warning`, `info`). Exported as singleton `toastManager`. |
| **`useEditorDetection.ts`** | Editor detection, extracted from `useSearch` for separation of concerns. `startEditorDetection(availableEditors, status)` owns the editor-detection event subscription (`editor-detection-start`/`-progress`/`-complete`), the initial `GetEditorDetectionStatus()` pull, and cleanup. |
| **`useMatchNavigation.ts`** | Navigation between search results within a code view (next/previous match, scrolling). |
| **`useCodeHighlighting.ts`** | Syntax highlighting of code content and highlighting search query matches. |
| **`useTheme.ts`** | Dark/light theme state that sets `data-theme` on `<html>` (tokens in `style.css`), persists the choice to `localStorage`, and follows the OS `prefers-color-scheme` when unset. |

> Recent-search history is managed inline by `useSearch.ts` (`addToRecentSearches`) with persistence helpers in `localStorageUtils.ts` / the shared `RecentSearch` type in `types/recentSearch.ts`.

### Services & utilities

- **`syntaxHighlightingService.ts`** — dynamically imports ~40 highlight.js language modules, detects language by file extension via `detectLanguage()`, highlights code with query-match highlighting. The extension→language map covers all common text types (programming languages, markup, config, docs, build files); unmapped extensions fall back to plain text. Large files (>1000 lines) skip per-line highlight.js calls for performance. Output is sanitized via DOMPurify. See [`EXTENSIONS.md`](EXTENSIONS.md) for the full extension system.
- **`searchUiUtils.ts`** — `highlightMatch` (with ReDoS protection: >10KB text in regex mode returns text as-is), `buildSearchRequest` (the single place a `SearchState` becomes a `SearchRequest`, shared by search and replace), and `openInEditor` — one dispatcher that routes to `OpenInDefaultEditor` for the `"default"` key and `OpenInEditorByName` for every cataloged editor. There are no per-editor wrappers on either side of the bridge.
- **`fileUtils.ts`** — path formatting, `handleEditorSelect` routing to the correct editor opener.
- **`toastUtils.ts`** — clipboard/file/editor operations with toast feedback.
- **`localStorageUtils.ts`** — recent searches persistence.
- **`errorUtils.ts`** — typed boundary-narrowing helpers: `toErrorMessage(unknown)` and `asRecord(unknown)`, used to coerce untyped Wails event payloads and caught errors into safe types without `any`.

---

## Communication channels

| Channel | Mechanism | Purpose |
| ------- | --------- | ------- |
| Wails bindings | Generated TypeScript stubs in `frontend/wailsjs/` | Direct calls from Vue to Go methods (`SearchWithProgress`, `CancelSearch`, `ReplaceInFiles`, `SelectDirectory`, `ValidateDirectory`, `ReadFile`, `OpenInEditorByName`, `OpenInDefaultEditor`, `ExportSearchResults`, `GetInitialLogs`, `GetNewLogs`, `GetAllSymbols`, `SearchSymbols`, `ClearSymbolCache`, `GetKnownTextExtensions`) |
| Wails events | `EventsOn` / `EventsEmit` | Search progress (`search-progress`, carrying `SearchProgress` — `FailedPaths` on the terminal `completed` event only), streamed result batches (`search-results`, carrying `SearchResultBatch{seq, results}`), replace progress (`replace-progress`, carrying `ReplaceProgress{phase, …}`), symbol-scan progress (`symbol-progress`), editor detection (`editor-detection-start`/`-progress`/`-complete`), and the one-shot `app-ready` |
| Log composable | `useLogStreaming()` calls `GetInitialLogs()` / `GetNewLogs()` | Log streaming via IPC (no HTTP server) |

---

## Performance & security

### Performance

- **Two-phase file collection**: directory walk (single-threaded, cheap filters) + parallel binary detection (worker pool). See the [File collection](#file-collection-two-phase) section above.
- **Known-text extension shortcut**: ~170 text extensions skip the binary probe entirely — no `open`/`read`/`close` syscall per known-text file. The set is exposed to the frontend via the `GetKnownTextExtensions()` binding and loaded into search state.
- **Zero-allocation path resolution**: absolute base directory and CWD computed once before the walk; per-file `absPath` uses `filepath.Clean` or `filepath.Join` instead of `filepath.Abs`.
- **Prefix-based traversal check**: replaces per-file `filepath.Rel` with a `strings.HasPrefix` check — zero allocations.
- **Worker pool** sized to CPU count for parallel file scanning.
- **Streaming** for files > 1 MB — no full-file reads into memory.
- **Size filtering** and binary detection skip files before expensive regex work.
- **Metadata reuse**: the directory walk records each file's absolute path and size once and hands them to the workers, avoiding a second `os.Stat`/`filepath.Abs` per file.
- **Context cancellation** for early termination.
- **Frontend**: pagination, 10,000-line preview cap.
- **Page-scoped highlighting**: search results are highlighted one page (10 rows) at a time rather than all results up front, so highlighting cost scales with page size, not total match count.
- **Syntax highlighting**: files > 1000 lines skip per-line `highlight()` calls (too slow with zero benefit on single-line snippets).
- **Debounced fuzzy re-scoring**: `fuzzyMatch.ts` exports a `debounce` helper (`DebouncedFn` type) so expensive `findFuzzyMatches` re-scoring can be throttled on rapid input.
### Security

- **Path traversal**: paths are cleaned with `filepath.Clean` and validated via prefix check against the separator-terminated base directory. The `..` component check runs on the raw input before cleaning. Editor/folder-reveal launches validate via `validatePathForEditor` / `validatePathForShowInFolder` (same raw-input `..` check + existence check) before any process is spawned.
- **Symlink handling**: `walkDirectoryTree` skips symlink files (`ModeSymlink`) so a link to a huge file cannot bypass `MaxFileSize` and OOM via `ReadFile`. The base directory is resolved with `filepath.EvalSymlinks` so a symlinked root cannot escape the prefix guard.
- **Protected directories**: `validateAndSetDefaults` blocks system roots and their subtrees (`/etc` and `/etc/ssh`, `/` and `/usr`, etc.) via `prefix+separator` check without blocking similar prefixes like `/etc-backup`.
- **Process launching**: all external processes go through `startAndReap` (no zombie leaks) and `appendPath` (no shared-slice aliasing between concurrent launches).
- **Input sanitization**: null bytes are rejected. Shell metacharacters (`|`, `&`, `;`, `` ` ``, `$(`) are NOT filtered — they are valid in Unix filenames and `ReadFile` never passes paths to a shell.
- **Allow-lists**: `AllowedFileTypes` restricts searched extensions.
- **Binary handling**: detected and skipped unless explicitly included. Known-text extensions skip the probe; unknown extensions get the 512-byte probe in parallel.
- **Resource limits**: max file size (10 MB), max results (1000 default, hard cap 10000 to bound memory), min file size, query length (2000 chars, rejected in `validateAndSetDefaults`; the frontend validator mirrors the same cap so the error surfaces before the IPC call).
- **CSV export**: `csvSafeCell` trims leading spaces before checking formula triggers (`=+-@\t\r`) and prefixes `'` so Excel/LibreOffice treat the cell as text (space-prefixed ` =2+2` bypass fixed).
- **Frontend**: DOMPurify sanitizes all rendered HTML. `diffUtils.renderDiffHtml` also escapes each segment with `escapeHtml` before assembly, so `<`/`>`/`&` in matched source lines are inert even before sanitization. Regex patterns are validated before use. `frontend/index.html` ships a `Content-Security-Policy` meta (`default-src 'self'`) limiting script/style/connect sources in the Wails webview.
- **Cache aliasing**: `collectionCache.get` and `symbolIndexCache.get` return copies of their entries, so callers that sort or append cannot mutate shared cache state.
---

## Testing & development

The frontend can run against a mock Wails backend, so the full UI is exercisable in a plain browser without the Go process:

- **Browser mock backend** (`frontend/src/mocks/wailsMock.ts`) — installs `window.go.main.App` and `window.runtime`, mirroring the real Wails bindings and events. It is loaded from `main.ts` via a lazy dynamic import **only** when `VITE_WAILS_MOCK` is set, so it is tree-shaken out of production builds. The `dev:mock` npm script (`VITE_WAILS_MOCK=1 vite`) runs the app in this mode.
- **Playwright E2E** (`playwright.config.js`, 7 specs under `playwright-tests/`: `flows.spec.ts`, `filetree-suggestions.spec.ts`, `enhancements.spec.ts`, `search-options.spec.ts`, `advanced-search.spec.ts`, `fuzzy-search.spec.ts`, `find-replace.spec.ts`) — drives the Vue frontend end to end against the mock backend using system Chrome, auto-starting Vite in mock mode. 41 flows cover startup, search → results, disabled empty-query, the file-preview modal, symbol search (with and without a directory), case-sensitivity, the File Explorer tree, the suggestions dropdown, regex patterns + truncation + theme + clipboard + modal-footer actions, pagination, match navigation, directory scoping + exclude patterns, fuzzy near-miss candidates with badges, and find-replace preview + apply. The modal-footer flow targets `/mock/big` because "Jump to Line" is only offered for files over `LINE_JUMP_MIN_LINES` (50) — that is when `MatchNavigationControls` mounts and there is an input to focus. Run via the `test:e2e` script; `run_tests.sh` gates E2E behind `RUN_E2E=1`. `retries` is 2 in CI and 0 locally, and `reuseExistingServer` is disabled in CI so a stale server can never serve the tests.
- **CI** (`.github/workflows/build.yml`) — the test job runs on a 3-OS matrix (`ubuntu-latest`, `windows-latest`, `macos-latest`) with `fail-fast: false`. That matrix is what compiles the build-tagged platform files at all: `appDarwin.go` (`//go:build darwin`) and `appWindows.go` (`//go:build windows`) are invisible to a Linux-only build. Go tests run `-race -covermode=atomic -coverprofile=coverage.out` on all three; lint, frontend, and cross-compile steps are gated to ubuntu with `if: matrix.os == 'ubuntu-latest'`. Non-Linux runners get a stub `frontend/dist/index.html` because `main.go`'s `//go:embed all:frontend/dist` cannot compile without an embed target and `frontend/dist` is gitignored. A tag push matching `v*` triggers the release job, which reuses the build job's artifacts via `gh release create`.
