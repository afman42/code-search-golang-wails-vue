# Architecture & Testing Summary

A reference for how the Code Search application is structured and how it is tested. It is a Wails desktop app: a Go backend handles file system search and system integration, a Vue 3 + TypeScript frontend renders the UI, and Wails bridges the two with generated TypeScript bindings.

## Table of Contents

- [Overview](#overview)
- [Backend (Go)](#backend-go)
- [Frontend (Vue)](#frontend-vue)
- [Communication](#communication)
- [Performance](#performance)
- [Security](#security)
- [Testing](#testing)
- [Development Workflow](#development-workflow)

## Overview

Two communication channels connect the frontend and backend:

1. **Wails bindings** — direct, type-safe calls from Vue into exported Go methods, plus Wails events for search progress.
2. **HTTP polling** — a separate Go HTTP server on port **34116** that tails the log file and serves new entries to the frontend.

```
┌──────────────────┐   Wails bindings + events   ┌──────────────────┐
│  Vue 3 frontend  │ ◄─────────────────────────► │   Go backend     │
│  (UI / state)    │                             │ (search engine)  │
└──────────────────┘                             └──────────────────┘
         ▲                                                 │
         │ HTTP polling (/poll, /initial)                  │ file system
         │ log streaming, port 34116                       ▼
┌──────────────────┐                             ┌──────────────────┐
│  polling client  │ ◄──────────────────────────│  log file tail   │
└──────────────────┘                             └──────────────────┘
```

Design goals: keep search fast on large trees (parallelism + streaming), keep file operations safe (path validation, allow-lists), and keep the UI responsive (async calls, progress events, pagination).

## Backend (Go)

### Source files

| File                     | Responsibility                                                                 |
| ------------------------ | ------------------------------------------------------------------------------ |
| `main.go`                | Entry point. Creates the app, ensures `logs/`, starts the polling server (port 34116), runs Wails (title `code-search-golang`, 1024×768). |
| `app_core.go`            | `App` struct, `NewApp`, search-cancel helpers, shutdown, `ReadFileLog` (resolves log path). |
| `models.go`              | Data types: `SearchRequest`, `SearchResult`, `SearchProgress`, `SearchState`, `EditorAvailability`, `ProgressCallback`. |
| `search_engine.go`       | `SearchWithProgress`, `collectFilesToProcess`, `processFilesWithWorkers`, `processFileLineByLine`, `CancelSearch`. |
| `system_integration.go`  | `SelectDirectory`, `ValidateDirectory`, `ReadFile`, `GetDirectoryContents`, editor detection and `OpenIn*` methods. |
| `logger_utils.go`        | Logger setup, `isBinary`, `matchesPattern`, `matchExtension`, `validateAndSetDefaults`, `safeEmitEvent`. |
| `polling_server.go`      | `PollingLogManager` and the HTTP polling server.                               |
| `app.go`                 | Linux build (`//go:build linux`): `ShowInFolder` (`xdg-open`), open-in-editor. |
| `appWindows.go`          | Windows build (`//go:build windows`): `ShowInFolder` (`cmd /c start`), open-in-editor. |

### App struct

Defined in `app_core.go`:

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

### Key data types

`SearchRequest` (search parameters):

```go
type SearchRequest struct {
    Directory        string   `json:"directory"`
    Query            string   `json:"query"`
    Extension        string   `json:"extension"`
    CaseSensitive    bool     `json:"caseSensitive"`
    IncludeBinary    bool     `json:"includeBinary"`
    MaxFileSize      int64    `json:"maxFileSize"`      // default 10 MB
    MinFileSize      int64    `json:"minFileSize"`
    MaxResults       int      `json:"maxResults"`       // default 1000
    SearchSubdirs    bool     `json:"searchSubdirs"`
    UseRegex         *bool    `json:"useRegex"`
    ExcludePatterns  []string `json:"excludePatterns"`
    AllowedFileTypes []string `json:"allowedFileTypes"`
}
```

`SearchResult` (one match, with context):

```go
type SearchResult struct {
    FilePath      string   `json:"filePath"`
    LineNum       int      `json:"lineNum"`
    Content       string   `json:"content"`
    MatchedText   string   `json:"matchedText"`
    ContextBefore []string `json:"contextBefore"`
    ContextAfter  []string `json:"contextAfter"`
}
```

### Search engine

`SearchWithProgress` is the core entry point:

- **Worker pool** sized to the available CPU count processes files concurrently.
- **Streaming**: files larger than 1 MB are read line-by-line (`processFileLineByLine`) with a 1 MB scanner buffer, so memory stays flat regardless of file size.
- **Early termination**: once `MaxResults` is reached the search context is cancelled and workers stop.
- **Progress**: counts and percentages are emitted to the frontend via Wails events.

`isBinary` (`logger_utils.go`) reads the first 512 bytes and classifies a file as binary if it contains a null byte or fewer than 50% printable characters.

### System integration

- `SelectDirectory` uses the cross-platform Wails `OpenDirectoryDialog`, so directory selection works on Linux, Windows, and macOS.
- `ValidateDirectory` checks existence and access and applies path-safety rules.
- Editor support: `detectAvailableEditors` probes ~21 editor commands in parallel with `exec.LookPath`; per-editor `OpenIn*` methods launch VS Code, VSCodium, Sublime, the JetBrains IDEs (routed by file extension), and others.
- `ShowInFolder` is implemented for Linux (`xdg-open`) and Windows (`cmd /c start`). macOS folder reveal and open-in-editor are **not** implemented in the current build.

### Polling log server

`PollingLogManager` (`polling_server.go`):

- Runs an HTTP server on port 34116 (alongside the default Wails port 34115).
- Tails `logs/app.log` with `github.com/nxadm/tail` (`Follow: true`).
- Exposes `/initial` (recent lines) and `/poll` (new lines since the last request).
- Keeps a bounded buffer (max ~1000 entries, trimmed to ~750) and filters noisy lines such as `Skipping` and `Sending file`.
- Sends CORS headers so the frontend can poll it.

## Frontend (Vue)

Vue 3 + TypeScript, built with Vite. State and search logic live in composables; the UI is split into focused components.

### Components

| Component              | Role                                                            |
| ---------------------- | --------------------------------------------------------------- |
| `App.vue`              | Root shell.                                                     |
| `CodeSearch.vue`       | Main orchestrator; composes the search UI.                      |
| `StartupLoader.vue`    | Startup loading state.                                          |
| `ui/SearchForm.vue`    | Search parameters, validation, recent searches.                |
| `ui/SearchResults.vue` | Paginated results (10 per page) with copy / open actions.       |
| `ui/ProgressIndicator.vue` | Real-time progress display.                                 |
| `ui/CodeModal.vue`     | File preview with syntax highlighting and match navigation.     |

Other UI pieces include `EnhancedTreeItem.vue`, `ToastNotification.vue`, and `LogViewer.vue`.

### Types, constants, and utilities

| Directory / File       | Purpose                                                 |
| ---------------------- | ------------------------------------------------------- |
| `types/search.ts`      | TypeScript interfaces for `SearchResult`, `SearchRequest`, `SearchProgress`, `EditorAvailability`, `EditorDetectionStatus`, `SearchState`. |
| `types/wails.d.ts`     | Ambient type declarations for Wails-generated bindings. |
| `constants/appConstants.ts` | Default search params, storage keys, display limits.   |
| `constants/startupConstants.ts` | `APP_READY_TIMEOUT` (3 s).                          |
| `utils/fileUtils.ts`   | Path formatting, editor routing via `handleEditorSelect`. |
| `utils/localStorageUtils.ts` | `loadRecentSearches` / `saveRecentSearches` with error handling. |
| `utils/searchUiUtils.ts` | `highlightMatch`, `copyToClipboard`, `openFileLocation`, per-editor `openIn*` wrappers. |
| `utils/toastUtils.ts`  | `copyToClipboardWithToast`, `openFileLocationWithToast`, `openInEditorWithToast`. |
| `utils/uiUtils.ts`     | Generic `highlightMatch` and `copyToClipboard` helpers. |

### Composables and services

- `composables/useSearch.ts` — central search state, Wails binding calls, progress-event handling, editor-detection event subscription, and `localStorage` persistence of recent searches. `truncatedResults` is set when the result count hits the 1000 backend limit.
- `composables/useToast.ts` — reactive toast notification system with auto-dismiss, pause-on-hover, and convenience methods (`success`, `error`, `warning`, `info`). Exported as singleton `toastManager`.
- `composables/useSyntaxHighlighting.ts` — loads highlight.js on component mount and exposes `isSyntaxHighlightingReady` / `initializeSyntaxHighlighting`.
- `services/syntaxHighlightingService.ts` — dynamically imports and registers ~25 highlight.js language modules, detects language by file extension, highlights code with optional query-match highlighting (capped at 10,000 lines), and sanitizes output via DOMPurify.
- `services/appInitializationService.ts` — eagerly preloads highlight.js at app startup through `main.ts`.

### Frontend performance

- highlight.js is preloaded at startup via `appInitializationService` for instant previews (no lazy-load delay when opening a file).
- Pagination keeps the DOM small for large result sets.
- File previews are capped at 10,000 lines.
- All backend calls are async with explicit loading/progress states; Wails event listeners are cleaned up to avoid leaks.

## Communication

### Wails bindings

Wails generates TypeScript bindings (under `frontend/src/wailsjs/`) for the exported `App` methods, with interfaces derived from the Go structs. Calls are asynchronous and errors propagate from Go to TypeScript.

### Events

Search progress is delivered through Wails events (e.g. `search-progress`) with file counts and percentages. Editor detection reports progress through its own events. Listeners are removed when no longer needed.

### HTTP polling

Used for log streaming instead of WebSockets for simplicity and reliability. The frontend polls `/initial` once, then `/poll` at intervals; the server returns only entries newer than the client's last read position.

## Performance

**Backend**

- Worker pool sized to CPU count; load balanced across goroutines.
- Streaming for files > 1 MB; 1 MB scanner buffer.
- Size-based filtering and binary detection skip files before expensive work.
- Context cancellation for early termination and clean shutdown.

**Frontend**

- Code splitting and dynamic imports reduce the initial bundle.
- Pagination and preview truncation bound rendering cost.
- Async operations keep the UI responsive during long searches.

## Security

- **Path traversal**: paths are cleaned with `filepath.Clean` and validated before any operation.
- **Allow-lists**: `AllowedFileTypes` restricts which extensions are searched.
- **Binary handling**: binary files are detected and skipped unless explicitly included.
- **Resource limits**: max file size, max results, and min file size guard against runaway searches.
- **Read-only**: search performs no writes.
- **Editor launching**: file paths are validated before being passed to external editors.
- **Frontend**: content is sanitized before rendering; recent searches in `localStorage` are validated on read.

## Testing

### Backend

Go's standard `testing` package. Test suites (`*_test.go`):

`app_test.go`, `binary_file_test.go`, `data_validation_test.go`, `debug_search_test.go`, `edge_cases_test.go`, `error_recovery_test.go`, `extended_app_test.go`, `improved_features_test.go`, `memory_performance_test.go`, `read_file_test.go`, `search_with_progress_test.go`, `security_test.go`.

These cover search workflows, edge cases, error recovery, memory/performance behavior, file reading, and security checks (path traversal, input handling, binary detection).

```bash
go test -v ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### Frontend

Vitest with `@vue/test-utils` in a jsdom environment (config: `vitest.config.ts`). Specs live in `frontend/tests/`:

- `unit/components/` — `CodeModal.spec.ts`, `CodeModal.syntax.spec.ts`, `ProgressIndicator.spec.ts`, `SearchForm.spec.ts`, `SearchResults.spec.ts`.
- `unit/composables/` — `useSearch.spec.ts`, `useSearch.additional.spec.ts`, `useSearch.comprehensive.spec.ts`.
- `EnhancedTreeItem.spec.ts` — tree component with rendering, expansion, filtering, and edge-case tests.
- `setup.ts` — global test setup: preloads highlight.js, mocks `IntersectionObserver` and `scrollIntoView`, stubs clipboard fallback, resets mocks between tests.
- `__mocks__/wailsjs/` — fake Wails binding modules (`go/main/App`, `runtime`) so component tests run without a real Wails bridge.
- `fixtures/` — shared test data (e.g. `editorAvailability.ts`).

```bash
cd frontend
npm test
npm run test:watch
```

> Note: `run_tests.sh` is stale — it `cd`s to an outdated path and does not run the Vitest suite. Run the commands above directly.

## Development Workflow

### Prerequisites

- Go 1.23+
- Node.js 16.x+ with npm
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Setup

```bash
go mod tidy
cd frontend && npm install && cd ..
```

### Run and build

```bash
wails dev     # hot-reload development
wails build   # production binary in build/bin/
```

### Conventions

- Go: format with `go fmt ./...`; use context for cancellation; add godoc comments on exported symbols.
- Vue: composition API with composables for shared logic; TypeScript throughout.
- Keep input validation and path-safety checks intact when changing backend code.
- Update this document and the README when behavior changes.
