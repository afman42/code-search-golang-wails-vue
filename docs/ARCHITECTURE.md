# Architecture

This document describes how the Code Search application is structured. It is a Wails desktop app: a Go backend handles file-system operations, a Vue 3 + TypeScript frontend renders the UI, and Wails generates type-safe bindings between them.

## Overview

Three channels connect the frontend and backend:

1. **Wails bindings** — direct type-safe calls from Vue into exported Go methods (`SearchWithProgress`, `SelectDirectory`, `GetInitialLogs`, etc.).
2. **Wails events** — `EventsOn` / `EventsEmit` for search progress, symbol-scan progress, and editor detection.
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
| `models.go`              | All backend type definitions: `SearchRequest`, `SearchResult`, `SearchProgress`, `SearchState`, `EditorAvailability`, `SymbolInfo`, `LogMessage`, `PollingLogManager`, `App`, `LRUPatternCache`, `symbolIndexCache`, `collectStats`, `fileMeta`. |
| `symbols.go`             | Symbol-extraction engine: `GetAllSymbols`, `SearchSymbols`, `GetAllSymbolsWithProgress` (two-pass scan). Checks the persistent symbol index (`symbol_index.go`) before extracting; stores results on cache miss. |
| `symbol_index.go`        | Persistent symbol index: `symbolIndexCache` (in-memory, per-directory, keyed by file fingerprint = path+size+mtime hash). `computeDirectoryFingerprint`, `ClearSymbolCache` binding. Caps at 8 directories. |
| `search_engine.go`       | `SearchWithProgress`, worker pool, line-by-line streaming for large files, `CancelSearch`. Multi-directory collection (deduped). Progress events throttled to 50ms. Results sorted by path+line. Cancelled searches return empty (no misleading "completed"). |
| `export.go`              | `ExportSearchResults` Wails binding — opens a native `SaveFileDialog` and writes CSV or JSON. `renderResultsCSV` is a pure helper. |
| `file_collection.go`     | Two-phase file collection: `walkDirectoryTree` (single-threaded walk + cheap filters) and `probeBinaryInParallel` (worker pool for binary detection on unknown extensions). |
| `text_extensions.go`     | Set of ~170 known-text extensions (.go, .ts, .py, .md, .vue, .toml, .txt, etc.) that skip the binary detection probe entirely. Exposes `GetKnownTextExtensions()` — a Wails binding the frontend uses to populate the "Allowed File Types" dropdown from the same source of truth. See [`EXTENSIONS.md`](EXTENSIONS.md). |
| `system_integration.go`  | Directory dialog, directory validation, file reading, editor detection (22 editors), all `OpenIn*` methods, `OpenInEditorByName` dispatcher. |
| `logger_utils.go`        | Logger setup, `isBinary` (zero-allocation), `matchesPattern` (path-component matching), `validateAndSetDefaults`, `safeEmitEvent`. |
| `polling_server.go`      | `PollingLogManager` — in-memory log buffer, file tailing, noise filtering. No HTTP server. Entries are consumed by the frontend via Wails IPC bindings. |
| `app.go`                 | Linux build (`//go:build linux`): `ShowInFolder` (`xdg-open`), `openInEditor` helper. |
| `appWindows.go`          | Windows build (`//go:build windows`): `ShowInFolder` (`explorer`), `openInEditor` helper. |
| `appDarwin.go`           | macOS build (`//go:build darwin`): `ShowInFolder` (`open -R`), `openInEditor` helper, `OpenInDefaultEditor` (`open`). |

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
- **Cancel semantics**: if the context is cancelled before hitting the result limit, `SearchWithProgress` returns an empty slice and does *not* emit a misleading `completed` event — only the `cancelled` event from `CancelSearch` is emitted.
- **Progress**: counts and percentages are emitted via Wails events.
- **Binary detection**: `isBinary` reads the first 512 bytes — files with null bytes or < 50% printable characters are skipped unless `IncludeBinary` is set.

### Symbol search

`GetAllSymbols` / `SearchSymbols` (`symbols.go`, exposed as Wails bindings in `app_symbols.go`) extract symbol definitions — functions, classes, variables, constants, interfaces, types — from source files. Supported languages are Go (`.go`), TypeScript (`.ts`/`.tsx`), JavaScript (`.js`), and Vue (`.vue`); common build/dependency directories (`node_modules`, `.git`, `vendor`, `build`, `dist`, `bin`) are skipped.

`GetAllSymbolsWithProgress` performs a **two-pass scan**: it first enumerates all supported source files so the total is known up front, then extracts symbols file by file, invoking a progress callback after each file. `app_symbols.go` forwards that callback as a `symbol-progress` Wails event (`{processed, total, currentFile}`), so the frontend `SymbolSearch.vue` panel renders a real progress bar rather than a synthetic one.

### File collection (two-phase)

The collection phase (`collectFilesToProcess` in `file_collection.go`) is split into two phases for performance:

**Phase 1 — `walkDirectoryTree`** (single-threaded directory walk):

Walks the directory tree with `filepath.WalkDir` and applies cheap filters (extension, size, exclude patterns). Files are split into two slices:
- `textCandidates` — files with known-text extensions (skip binary probe) or `IncludeBinary=true`
- `binaryCheckCandidates` — files with unknown extensions that need the 512-byte binary probe

Optimizations applied during the walk:
- **Absolute base computed once**: `filepath.Abs(req.Directory)` is called once before the walk, not per file. Each file's `absPath` is resolved via `filepath.Clean` (absolute paths) or `filepath.Join(cwd, path)` (relative paths) — no per-file syscall.
- **Prefix-based traversal check**: replaces the per-file `filepath.Rel` + `..` check with a `strings.HasPrefix(absPath, baseDir + separator)` check — zero allocations.
- **Known-text extension shortcut**: ~170 text extensions (`.go`, `.ts`, `.py`, `.md`, `.json`, `.vue`, `.toml`, `.txt`, etc.) are recognized via `text_extensions.go`. Files with these extensions skip the binary probe entirely — no `open` + `read` + `close` syscall. The same set is exposed to the frontend via `GetKnownTextExtensions()` so the UI dropdown and the backend's collection logic share one source of truth (see [`EXTENSIONS.md`](EXTENSIONS.md)).

**Phase 2 — `probeBinaryInParallel`** (worker pool):

If `binaryCheckCandidates` is non-empty, a worker pool (sized to CPU count) runs the 512-byte binary detection probe on each candidate in parallel. Each worker reuses a pooled 512-byte buffer. Files that pass the probe (are text) are appended to the final list; binary files are counted as skipped.

On a tree of 2000 `.go` files (all known-text), Phase 2 is empty and the walk is the only cost. On a mixed tree with unknown extensions, Phase 2 parallelizes the binary probes across CPU cores.

**Benchmark impact** (Celeron N4000, 2 cores, 2000 `.go` files):

| Benchmark | Single-pass | Two-phase | Improvement |
|-----------|-------------|-----------|-------------|
| `CollectFilesToProcess` | 98 ms, 18772 allocs | 27 ms, 12779 allocs | **3.6x faster, 32% fewer allocs** |
| `SearchWithProgress` | 200 ms, 33781 allocs | 127 ms, 27782 allocs | **1.6x faster, 18% fewer allocs** |

### File-extension system

The app tracks file extensions in three places. Full details live in [`EXTENSIONS.md`](EXTENSIONS.md); the summary:

- **Known-text set** (`text_extensions.go` → `knownTextExtensions`) — ~170 extensions that skip the binary probe. The single source of truth for "is this file text?"
- **Allow-list dropdown** (`SearchForm.vue`) — renders from `data.knownTextExtensions`, which `useSearch.ts` loads via the `GetKnownTextExtensions()` Wails binding. The UI suggestion list and the backend's collection logic share one source, so they stay in sync.
- **Language detection** (`syntaxHighlightingService.ts` → `detectLanguage()`) — a separate map from extension to highlight.js language name, because the question "which highlighter?" is independent of "is this text?". Not every text extension has a highlight.js language; unmapped extensions fall back to plain text in the preview modal.

### System integration

- **Directory selection**: uses the cross-platform Wails `OpenDirectoryDialog`.
- **Editor detection**: probes 22 editor commands in parallel via `exec.LookPath`. Detected editors include VS Code, VSCodium, Sublime, Geany, JetBrains IDEs (GoLand, PyCharm, IntelliJ, WebStorm, PhpStorm, CLion, Rider — routed by file extension), Android Studio, Emacs, Neovim, Neovide, Vim, Code::Blocks, Dev-C++, Notepad++, Visual Studio, Eclipse, NetBeans.
- **Open-in-editor**: per-editor `OpenIn*` methods call `openInEditor` helper with the editor command and any flags.
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
| **`useSearch.ts`** | Central search state, calls Wails backend, handles progress events, and `localStorage` persistence of recent searches. On startup it also calls `GetKnownTextExtensions()` to populate `data.knownTextExtensions`, which `SearchForm.vue` renders as the "Allowed File Types" dropdown. Editor detection has been extracted into `useEditorDetection.ts` (separation of concerns). |
| **`useLogStreaming.ts`** | Encapsulates log streaming: Wails binding calls, log parsing, polling interval management, and lifecycle hooks. Exports `parseLogEntry()` for reuse. |
| **`useToast.ts`** | Reactive toast notification system with auto-dismiss, pause/resume with accurate remaining-time tracking, and convenience methods (`success`, `error`, `warning`, `info`). Exported as singleton `toastManager`. |
| **`useEditorDetection.ts`** | Editor detection, extracted from `useSearch` for separation of concerns. `startEditorDetection(availableEditors, status)` owns the editor-detection event subscription (`editor-detection-start`/`-progress`/`-complete`), the initial `GetEditorDetectionStatus()` pull, and cleanup. |
| **`useMatchNavigation.ts`** | Navigation between search results within a code view (next/previous match, scrolling). |
| **`useCodeHighlighting.ts`** | Syntax highlighting of code content and highlighting search query matches. |
| **`useTheme.ts`** | Dark/light theme state that sets `data-theme` on `<html>` (tokens in `style.css`), persists the choice to `localStorage`, and follows the OS `prefers-color-scheme` when unset. |

> Recent-search history is managed inline by `useSearch.ts` (`addToRecentSearches`) with persistence helpers in `localStorageUtils.ts` / the shared `RecentSearch` type in `types/recentSearch.ts`.

### Services & utilities

- **`syntaxHighlightingService.ts`** — dynamically imports ~35 highlight.js language modules, detects language by file extension via `detectLanguage()`, highlights code with query-match highlighting. The extension→language map covers all common text types (programming languages, markup, config, docs, build files); unmapped extensions fall back to plain text. Large files (>1000 lines) skip per-line highlight.js calls for performance. Output is sanitized via DOMPurify. See [`EXTENSIONS.md`](EXTENSIONS.md) for the full extension system.
- **`appInitializationService.ts`** — preloads highlight.js at startup.
- **`searchUiUtils.ts`** — `highlightMatch` (with ReDoS protection: >10KB text in regex mode returns text as-is), `copyToClipboard`, `openFileLocation`, per-editor `openIn*` wrappers.
- **`fileUtils.ts`** — path formatting, `handleEditorSelect` routing to the correct editor opener.
- **`toastUtils.ts`** — clipboard/file/editor operations with toast feedback.
- **`localStorageUtils.ts`** — recent searches persistence.
- **`errorUtils.ts`** — typed boundary-narrowing helpers: `toErrorMessage(unknown)` and `asRecord(unknown)`, used to coerce untyped Wails event payloads and caught errors into safe types without `any`.

---

## Communication channels

| Channel | Mechanism | Purpose |
| ------- | --------- | ------- |
| Wails bindings | Generated TypeScript stubs in `frontend/wailsjs/` | Direct calls from Vue to Go methods (`SearchWithProgress`, `SelectDirectory`, `ReadFile`, `OpenIn*`, `GetInitialLogs`, `GetNewLogs`, `GetAllSymbols`, `SearchSymbols`) |
| Wails events | `EventsOn` / `EventsEmit` | Search progress (`search-progress`), symbol-scan progress (`symbol-progress`), editor detection (`editor-detection-start`/`-progress`/`-complete`) |
| Log composable | `useLogStreaming()` calls `GetInitialLogs()` / `GetNewLogs()` | Log streaming via IPC (no HTTP server) |

---

## Performance & security

### Performance

- **Two-phase file collection**: directory walk (single-threaded, cheap filters) + parallel binary detection (worker pool). See the [File collection](#file-collection-two-phase) section above.
- **Known-text extension shortcut**: ~170 text extensions skip the binary probe entirely — no `open`/`read`/`close` syscall per known-text file. The same set drives the frontend's "Allowed File Types" dropdown via the `GetKnownTextExtensions()` binding.
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

### Security

- **Path traversal**: paths are cleaned with `filepath.Clean` and validated via prefix check against the separator-terminated base directory. The `..` component check runs on the raw input before cleaning.
- **Input sanitization**: null bytes are rejected. Shell metacharacters (`|`, `&`, `;`, `` ` ``, `$(`) are NOT filtered — they are valid in Unix filenames and `ReadFile` never passes paths to a shell.
- **Allow-lists**: `AllowedFileTypes` restricts searched extensions.
- **Binary handling**: detected and skipped unless explicitly included. Known-text extensions skip the probe; unknown extensions get the 512-byte probe in parallel.
- **Resource limits**: max file size (10 MB), max results (1000), min file size.
- **Frontend**: DOMPurify sanitizes all rendered HTML. Regex patterns are validated before use.

---

## Testing & development

The frontend can run against a mock Wails backend, so the full UI is exercisable in a plain browser without the Go process:

- **Browser mock backend** (`frontend/src/mocks/wailsMock.ts`) — installs `window.go.main.App` and `window.runtime`, mirroring the real Wails bindings and events. It is loaded from `main.ts` via a lazy dynamic import **only** when `VITE_WAILS_MOCK` is set, so it is tree-shaken out of production builds. The `dev:mock` npm script (`VITE_WAILS_MOCK=1 vite`) runs the app in this mode.
- **Playwright E2E** (`playwright.config.js`, `playwright-tests/flows.spec.ts` + `playwright-tests/filetree-suggestions.spec.ts`) — drives the Vue frontend end to end against the mock backend using system Chrome, auto-starting Vite in mock mode. Flows cover startup, search → results, disabled empty-query, the file-preview modal, symbol search (with and without a directory), case-sensitivity, the File Explorer tree, and the suggestions dropdown. Run via the `test:e2e` script; the default `run_tests.sh` stays browser-free and only runs the Playwright stage when `RUN_E2E=1` is set.
