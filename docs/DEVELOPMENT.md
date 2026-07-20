# Development

## Setup

```bash
go mod tidy
cd frontend && npm install && cd ..
```

## Run

```bash
wails dev     # hot-reload development server (Vite + Go)
wails build   # production binary in build/bin/
```

## Conventions

- **Go**: format with `go fmt ./...`; use context for cancellation; add godoc on exported symbols.
- **Vue**: Composition API with `<script setup>`; TypeScript throughout.
- **Composables**: Extract reusable state and logic into composables under `frontend/src/composables/`. Composables should encapsulate a single responsibility (e.g., `useSearch` for search state, `useLogStreaming` for log streaming) and expose a clean return interface. Lifecycle hooks (`onMounted`, `onUnmounted`) belong in composables, not components, so the logic is testable independently of the template.
- **Tests**: Go tests for backend functions; Vitest specs for components and composables. Composables should have their own test file (e.g., `useLogStreaming.spec.ts`) covering exported functions and state transitions; component tests should focus on template rendering and user interaction.
- **Security**: Keep input validation and path-safety checks intact when changing backend code.
- **File extensions**: The backend's `knownTextExtensions` map in `text_extensions.go` is the single source of truth for which file types are text. The frontend loads it via the `GetKnownTextExtensions()` Wails binding — do not hand-maintain a parallel extension list in the UI. The language-detection map in `syntaxHighlightingService.ts` is separate (it maps extensions to highlight.js languages, not to text/binary). See [`EXTENSIONS.md`](EXTENSIONS.md) for the full system and the steps to add a new extension.
- **Log streaming**: Log entries are delivered to the frontend via Wails IPC bindings (`GetInitialLogs`, `GetNewLogs`), not HTTP polling. The `useLogStreaming` composable encapsulates all streaming logic. Do not reintroduce HTTP polling — Wails bindings avoid CORS/mixed-content issues in production builds and are always available (same process).
- **Docs**: Update this file, the README, and the relevant `docs/` page when behavior changes.
