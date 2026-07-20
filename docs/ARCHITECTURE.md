# Architecture

This document describes how the Code Search application is structured. It is a Wails desktop app: a Go backend handles file-system operations, a Vue 3 + TypeScript frontend renders the UI, and Wails generates type-safe bindings between them.

## Overview

Two communication channels connect the frontend and backend:

1. **Wails bindings** — direct type-safe calls from Vue into exported Go methods, plus Wails events for search progress and editor detection.
2. **HTTP polling** — a separate Go HTTP server on port **34116** that tails the log file and serves new entries to the frontend.

```
┌──────────────────┐   Wails bindings + events   ┌──────────────────┐
│  Vue 3 frontend  │ ◄──────────────────────────► │   Go backend     │
│  (UI / state)    │                             │ (search engine)  │
└──────────────────┘                             └──────────────────┘
         ▲                                                 │
         │ HTTP polling (/poll, /initial)                  │ file system
         │ port 34116                                      ▼
┌──────────────────┐                             ┌──────────────────┐
│  polling client  │ ◄──────────────────────────│  log file tail   │
└──────────────────┘                             └──────────────────┘
```

---

## Backend

### Source files

| File                     | Responsibility |
| ------------------------ | -------------- |
| `main.go`                | Entry point. Creates the app, ensures `logs/` directory, starts polling server (port 34116), runs Wails (title `code-search-golang`, 1024×768). |
| `app_core.go`            | `App` struct, `NewApp`, search-cancel helpers, shutdown, `ReadFileLog`. |
| `models.go`              | Data types: `SearchRequest`, `SearchResult`, `SearchProgress`, `EditorAvailability`. |
| `search_engine.go`       | `SearchWithProgress`, worker pool, line-by-line streaming for large files, `CancelSearch`. |
| `file_collection.go`     | Two-phase file collection: `walkDirectoryTree` (single-threaded walk + cheap filters) and `probeBinaryInParallel` (worker pool for binary detection on unknown extensions). |
| `text_extensions.go`     | Set of ~150 known-text extensions (.go, .ts, .py, .md, etc.) that skip the binary detection probe entirely. |
| `system_integration.go`  | Directory dialog, directory validation, file reading, editor detection (22 editors), all `OpenIn*` methods, `OpenInEditorByName` dispatcher. |
| `logger_utils.go`        | Logger setup, `isBinary` (zero-allocation), `matchesPattern` (path-component matching), `validateAndSetDefaults`, `safeEmitEvent`. |
| `polling_server.go`      | `PollingLogManager` and HTTP polling server (loopback-only, origin-restricted CORS). |
| `app.go`                 | Linux build (`//go:build linux`): `ShowInFolder` (`xdg-open`), `openInEditor` helper. |
| `appWindows.go`          | Windows build (`//go:build windows`): `ShowInFolder` (`explorer`), `openInEditor` helper. |

### App struct

```go
type App struct {
    ctx              context.Context
    logger           *logrus.Logger
    searchMu         sync.Mutex
    searchCancel     context.CancelFunc
    editorsMu        sync.RWMutex
    availableEditors EditorAvailability
}
```

### Search engine

`SearchWithProgress` is the core entry point:

- **Worker pool** sized to available CPU cores processes files concurrently.
- **Streaming**: files > 1 MB are read line-by-line with a 1 MB scanner buffer (flat memory usage).
- **Early termination**: once `MaxResults` is reached, the search context is cancelled and workers stop.
- **Progress**: counts and percentages are emitted via Wails events.
- **Binary detection**: `isBinary` reads the first 512 bytes — files with null bytes or < 50% printable characters are skipped unless `IncludeBinary` is set.

### File collection (two-phase)

The collection phase (`collectFilesToProcess` in `file_collection.go`) is split into two phases for performance:

**Phase 1 — `walkDirectoryTree`** (single-threaded directory walk):

Walks the directory tree with `filepath.WalkDir` and applies cheap filters (extension, size, exclude patterns). Files are split into two slices:
- `textCandidates` — files with known-text extensions (skip binary probe) or `IncludeBinary=true`
- `binaryCheckCandidates` — files with unknown extensions that need the 512-byte binary probe

Optimizations applied during the walk:
- **Absolute base computed once**: `filepath.Abs(req.Directory)` is called once before the walk, not per file. Each file's `absPath` is resolved via `filepath.Clean` (absolute paths) or `filepath.Join(cwd, path)` (relative paths) — no per-file syscall.
- **Prefix-based traversal check**: replaces the per-file `filepath.Rel` + `..` check with a `strings.HasPrefix(absPath, baseDir + separator)` check — zero allocations.
- **Known-text extension shortcut**: ~150 text extensions (`.go`, `.ts`, `.py`, `.md`, `.json`, etc.) are recognized via `text_extensions.go`. Files with these extensions skip the binary probe entirely — no `open` + `read` + `close` syscall.

**Phase 2 — `probeBinaryInParallel`** (worker pool):

If `binaryCheckCandidates` is non-empty, a worker pool (sized to CPU count) runs the 512-byte binary detection probe on each candidate in parallel. Each worker reuses a pooled 512-byte buffer. Files that pass the probe (are text) are appended to the final list; binary files are counted as skipped.

On a tree of 2000 `.go` files (all known-text), Phase 2 is empty and the walk is the only cost. On a mixed tree with unknown extensions, Phase 2 parallelizes the binary probes across CPU cores.

**Benchmark impact** (Celeron N4000, 2 cores, 2000 `.go` files):

| Benchmark | Before | After | Improvement |
|-----------|--------|-------|-------------|
| `CollectFilesToProcess` | 98 ms, 18772 allocs | 27 ms, 12779 allocs | **3.6x faster, 32% fewer allocs** |
| `SearchWithProgress` | 200 ms, 33781 allocs | 127 ms, 27782 allocs | **1.6x faster, 18% fewer allocs** |

### System integration

- **Directory selection**: uses the cross-platform Wails `OpenDirectoryDialog`.
- **Editor detection**: probes 22 editor commands in parallel via `exec.LookPath`. Detected editors include VS Code, VSCodium, Sublime, Atom, JetBrains IDEs (GoLand, PyCharm, IntelliJ, WebStorm, PhpStorm, CLion, Rider — routed by file extension), Android Studio, Emacs, Neovim, Neovide, Code::Blocks, Dev-C++, Notepad++, Visual Studio, Eclipse, NetBeans.
- **Open-in-editor**: per-editor `OpenIn*` methods call `openInEditor` helper with the editor command and any flags.
- **Show in folder**: Linux uses `xdg-open`, Windows uses `explorer`. macOS not yet implemented.

### Log polling server

`PollingLogManager` runs an HTTP server on port 34116:

- Binds to `127.0.0.1` only (loopback). The log stream is consumed solely by the local frontend, so this avoids LAN exposure and the Windows Defender Firewall prompt on first launch.
- Tails `logs/app.log` with `github.com/nxadm/tail`.
- `/initial` — returns the last 20 lines from the log file.
- `/poll` — returns new entries since the last poll.
- Bounded buffer (max ~1000 entries, trimmed to ~750).
- Filters noisy lines (`Skipping`, `Sending file`).
- Sends CORS headers for frontend access.

---

## Frontend

Vue 3 + TypeScript, built with Vite. State and search logic live in composables; UI is split into focused components.

### Components

| Component              | Role |
| ---------------------- | ---- |
| `App.vue`              | Root shell. |
| `CodeSearch.vue`       | Main orchestrator — composes the search UI. |
| `StartupLoader.vue`    | Loading state during initialization. |
| `ui/SearchForm.vue`    | Search parameters, validation, recent searches dropdown. |
| `ui/SearchResults.vue` | Paginated results (10/page) with copy, open-in-editor, and file-reveal actions. |
| `ui/ProgressIndicator.vue` | Real-time progress bar and status. |
| `ui/CodeModal.vue`     | File preview modal with syntax highlighting, match navigation, jump-to-line, tree view. Large files capped at 10,000 lines. |
| `ui/LogViewer.vue`     | Collapsible log viewer at the bottom of the screen. Shows a live stream from the backend log file, with a PREVIEW section showing last entries from `logs/app.log` when streaming is idle. |
| `ui/ToastNotification.vue` | Toast notifications with auto-dismiss and pause-on-hover. |
| `ui/EnhancedTreeItem.vue` | Recursive file-tree component with filtering and expand/collapse. |

### Composables

- **`useSearch.ts`** — central search state, calls Wails backend, handles progress events, editor-detection events, and `localStorage` persistence of recent searches.
- **`useToast.ts`** — reactive toast notification system with auto-dismiss, pause/resume with accurate remaining-time tracking, and convenience methods (`success`, `error`, `warning`, `info`). Exported as singleton `toastManager`.
- **`useSyntaxHighlighting.ts`** — loads highlight.js on mount.

### Services & utilities

- **`syntaxHighlightingService.ts`** — dynamically imports ~25 highlight.js language modules, detects language by file extension, highlights code with query-match highlighting. Large files (>1000 lines) skip per-line highlight.js calls for performance. Output is sanitized via DOMPurify.
- **`appInitializationService.ts`** — preloads highlight.js at startup.
- **`searchUiUtils.ts`** — `highlightMatch` (with ReDoS protection: >10KB text in regex mode returns text as-is), `copyToClipboard`, `openFileLocation`, per-editor `openIn*` wrappers.
- **`fileUtils.ts`** — path formatting, `handleEditorSelect` routing to the correct editor opener.
- **`toastUtils.ts`** — clipboard/file/editor operations with toast feedback.
- **`localStorageUtils.ts`** — recent searches persistence.

---

## Communication channels

| Channel | Mechanism | Purpose |
| ------- | --------- | ------- |
| Wails bindings | Generated TypeScript stubs in `frontend/wailsjs/` | Direct calls from Vue to Go methods (`SearchWithProgress`, `SelectDirectory`, `ReadFile`, `OpenIn*`, etc.) |
| Wails events | `EventsOn` / `EventsEmit` | Search progress, editor detection progress/completion |
| HTTP polling | `GET /initial` and `GET /poll` on `127.0.0.1:34116` | Log streaming from backend to LogViewer |

---

## Performance & security

### Performance

- **Two-phase file collection**: directory walk (single-threaded, cheap filters) + parallel binary detection (worker pool). See the [File collection](#file-collection-two-phase) section above.
- **Known-text extension shortcut**: ~150 text extensions skip the binary probe entirely — no `open`/`read`/`close` syscall per known-text file.
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
- **Log polling CORS**: `Access-Control-Allow-Origin` is reflected only for allowlisted origins (`http://localhost`, `http://127.0.0.1`, `http://wails.localhost`, `null`) — not a wildcard. Unknown origins get no CORS header.
- **Log polling bind**: loopback only (`127.0.0.1`), not exposed on the LAN.
- **Frontend**: DOMPurify sanitizes all rendered HTML. Regex patterns are validated before use.
