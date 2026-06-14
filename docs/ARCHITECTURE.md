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
| `search_engine.go`       | `SearchWithProgress`, file collection, worker pool, line-by-line streaming for large files, `CancelSearch`. |
| `system_integration.go`  | Directory dialog, directory validation, file reading, editor detection (22 editors), all `OpenIn*` methods. |
| `logger_utils.go`        | Logger setup, `isBinary`, `matchesPattern` (path-component matching), `validateAndSetDefaults`, `safeEmitEvent`. |
| `polling_server.go`      | `PollingLogManager` and HTTP polling server. |
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

- **Worker pool** sized to CPU count for parallel file scanning.
- **Streaming** for files > 1 MB — no full-file reads into memory.
- **Size filtering** and binary detection skip files before expensive regex work.
- **Metadata reuse**: the directory walk records each file's absolute path and size once and hands them to the workers, avoiding a second `os.Stat`/`filepath.Abs` per file.
- **Context cancellation** for early termination.
- **Frontend**: pagination, 10,000-line preview cap.
- **Page-scoped highlighting**: search results are highlighted one page (10 rows) at a time rather than all results up front, so highlighting cost scales with page size, not total match count.
- **Syntax highlighting**: files > 1000 lines skip per-line `highlight()` calls (too slow with zero benefit on single-line snippets).

### Security

- **Path traversal**: paths are cleaned with `filepath.Clean` and validated for `..` components.
- **Input sanitization**: null bytes, command-injection characters, and dangerous patterns are rejected.
- **Allow-lists**: `AllowedFileTypes` restricts searched extensions.
- **Binary handling**: detected and skipped unless explicitly included.
- **Resource limits**: max file size (10 MB), max results (1000), min file size.
- **Frontend**: DOMPurify sanitizes all rendered HTML. Regex patterns are validated before use.
