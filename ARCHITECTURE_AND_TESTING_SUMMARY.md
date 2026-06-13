# Architecture & Testing Summary

The Code Search app is a Wails desktop application: a Go backend handles file system search and system integration, a Vue 3 + TypeScript frontend renders the UI, and Wails bridges the two with generated TypeScript bindings.

Detailed documentation is split across the following files:

| Document | Contents |
| -------- | -------- |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Architecture overview, backend (source files, app struct, search engine, system integration, log polling), frontend (components, composables, services), communication channels, and performance & security. |
| [`docs/TESTING.md`](docs/TESTING.md) | Backend Go test suites and frontend Vitest specs (components, composables, utilities, infrastructure). |
| [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) | Setup, run/build commands, and code conventions. |

## Quick reference

### Backend source files

| File | Responsibility |
| ---- | -------------- |
| `main.go` | Entry point: app creation, polling server, Wails run. |
| `app_core.go` | `App` struct, lifecycle, search cancellation. |
| `models.go` | Shared data types (`SearchRequest`, `SearchResult`, etc.). |
| `search_engine.go` | `SearchWithProgress`, worker pool, streaming. |
| `system_integration.go` | Directory dialog, editor detection, `OpenIn*` methods. |
| `logger_utils.go` | Logger, `isBinary`, `matchesPattern`, validation. |
| `polling_server.go` | `PollingLogManager` and HTTP polling server. |
| `app.go` | Linux: `ShowInFolder` (`xdg-open`), open-in-editor. |
| `appWindows.go` | Windows: `ShowInFolder` (`cmd /c start`), open-in-editor. |

### Frontend components

| Component | Role |
| --------- | ---- |
| `App.vue` | Root shell. |
| `CodeSearch.vue` | Main orchestrator. |
| `StartupLoader.vue` | Startup loading state. |
| `ui/SearchForm.vue` | Search parameters and validation. |
| `ui/SearchResults.vue` | Paginated results (10/page). |
| `ui/ProgressIndicator.vue` | Real-time progress display. |
| `ui/CodeModal.vue` | File preview with syntax highlighting. |
| `ui/LogViewer.vue` | Collapsible log stream display. |
| `ui/ToastNotification.vue` | Toast notifications. |
| `ui/EnhancedTreeItem.vue` | Recursive file-tree component. |

### Testing commands

```bash
# Backend
go test -v ./...

# Frontend
cd frontend && npm test

# Full validation
bash run_tests.sh
```

> Note: `run_tests.sh` is stale — it `cd`s to an outdated path and does not run the Vitest suite. Run the commands above directly.
