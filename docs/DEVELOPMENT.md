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
- **Tests**: Go tests for backend functions; Vitest specs for components and composables.
- **Security**: Keep input validation and path-safety checks intact when changing backend code.
- **Docs**: Update this file and the README when behavior changes.
