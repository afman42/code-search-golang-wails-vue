# Architecture & Testing

This document describes how the Code Search application is structured and tested. It is a Wails desktop app: a Go backend handles file-system operations, a Vue 3 + TypeScript frontend renders the UI, and Wails generates type-safe bindings between them.

## Contents

- [Overview](#overview)
- [Backend](#backend)
  - [Source files](#source-files)
  - [App struct](#app-struct)
  - [Search engine](#search-engine)
  - [System integration](#system-integration)
  - [Log polling server](#log-polling-server)
- [Frontend](#frontend)
  - [Components](#components)
  - [Composables](#composables)
  - [Services & utilities](#services--utilities)
- [Communication channels](#communication-channels)
- [Performance & security](#performance--security)
- [Testing](#testing)

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
| `system_integration.go`  | Directory dialog, directory validation, file reading, editor detection (22 editors, Neovim included), all `OpenIn*` methods. |
| `logger_utils.go`        | Logger setup, `isBinary`, `matchesPattern` (path-component matching), `validateAndSetDefaults`, `safeEmitEvent`. |
| `polling_server.go`      | `PollingLogManager` and HTTP polling server. |
| `app.go`                 | Linux build (`//go:build linux`): `ShowInFolder` (`xdg-open`), `openInEditor` helper. |
| `appWindows.go`          | Windows build (`//go:build windows`): `ShowInFolder` (`cmd /c start`), `openInEditor` helper. |

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
- **Show in folder**: Linux uses `xdg-open`, Windows uses `cmd /c start`. macOS not yet implemented.

### Log polling server

`PollingLogManager` runs an HTTP server on port 34116:

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
| HTTP polling | `GET /initial` and `GET /poll` on `:34116` | Log streaming from backend to LogViewer |

---

## Performance & security

### Performance

- **Worker pool** sized to CPU count for parallel file scanning.
- **Streaming** for files > 1 MB — no full-file reads into memory.
- **Size filtering** and binary detection skip files before expensive regex work.
- **Context cancellation** for early termination.
- **Frontend**: pagination, 10,000-line preview cap, lazy highlight.js loading.
- **Syntax highlighting**: files > 1000 lines skip per-line `highlight()` calls (too slow with zero benefit on single-line snippets).

### Security

- **Path traversal**: paths are cleaned with `filepath.Clean` and validated for `..` components.
- **Input sanitization**: null bytes, command-injection characters, and dangerous patterns are rejected.
- **Allow-lists**: `AllowedFileTypes` restricts searched extensions.
- **Binary handling**: detected and skipped unless explicitly included.
- **Resource limits**: max file size (10 MB), max results (1000), min file size.
- **Frontend**: DOMPurify sanitizes all rendered HTML. Regex patterns are validated before use.

---

## Testing

### Backend (Go)

13 test files covering search workflows, edge cases, error recovery, memory/performance, file reading, and security:

- `app_test.go`, `binary_file_test.go`, `data_validation_test.go`, `debug_search_test.go`, `edge_cases_test.go`, `editor_detection_test.go`, `error_recovery_test.go`, `extended_app_test.go`, `improved_features_test.go`, `memory_performance_test.go`, `read_file_test.go`, `search_with_progress_test.go`, `security_test.go`.

Notable coverage:
- Editor detection: `isEditorAvailable` with existing/non-existent commands, `countAvailableEditors` (including Neovim count, JetBrains derived flag), `GetAvailableEditors`, `GetEditorDetectionStatus`, `openInEditor` error handling.
- Path traversal protection: validated across multiple attack vectors.
- Input validation: regex patterns, directory paths, numeric limits, exclude patterns.
- Binary file detection: null bytes, non-printable content.

```bash
go test -v ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### Frontend (Vitest)

12 test files with 197+ tests across components, composables, and utilities:

- `unit/components/` — `CodeModal.spec.ts` (24 tests), `CodeModal.syntax.spec.ts`, `LogViewer.spec.ts` (15 tests: collapse/expand, preview logs, placeholder, filtering, log parsing), `ProgressIndicator.spec.ts`, `SearchForm.spec.ts`, `SearchResults.spec.ts`.
- `unit/composables/` — `useSearch.spec.ts`, `useSearch.additional.spec.ts`, `useSearch.comprehensive.spec.ts`, `useToast.spec.ts` (17 tests: add/remove, pause/resume, idempotent operations, concurrent staggered durations, rapid add/remove cycles).
- `unit/utils/` — `searchUiUtils.spec.ts` (23 tests: literal/regex matching, case sensitivity, ReDoS protection, XSS sanitization, lookahead, word boundaries, null/overflow inputs).
- `EnhancedTreeItem.spec.ts` — tree rendering, expansion, filtering, edge cases.

**Test infrastructure** (`frontend/tests/`):
- `setup.ts` — preloads highlight.js, mocks `IntersectionObserver`, `scrollIntoView`, clipboard fallback.
- `__mocks__/wailsjs/` — fake Wails binding modules so component tests run without a real bridge.
- `fixtures/` — shared test data (e.g. `editorAvailability.ts` with all 22 editor fields).

```bash
cd frontend
npm test               # run once
npm run test:watch     # watch mode
```

### Full validation

```bash
bash run_tests.sh      # Runs Go tests + Vitest + TypeScript check
```

---

## Development

### Setup

```bash
go mod tidy
cd frontend && npm install && cd ..
```

### Run

```bash
wails dev     # hot-reload development server (Vite + Go)
wails build   # production binary in build/bin/
```

### Conventions

- **Go**: format with `go fmt ./...`; use context for cancellation; add godoc on exported symbols.
- **Vue**: Composition API with `<script setup>`; TypeScript throughout.
- **Tests**: Go tests for backend functions; Vitest specs for components and composables.
- **Security**: Keep input validation and path-safety checks intact when changing backend code.
- **Docs**: Update this file and the README when behavior changes.
