# Code Search

A cross-platform desktop app for searching text and regular expressions across code files. Built with [Wails](https://wails.io/) — Go backend + Vue 3 frontend.

## Features

**Search engine**
- Plain-text and regex search with case-sensitivity toggle
- File extension filter, file-type allow-lists, and exclude patterns (e.g. `node_modules`, `.git`, `*.log`)
- Subdirectory toggle, min/max file size, max result limit
- Binary file inclusion (off by default with binary detection)

**Results & preview**
- File path, line number, matched text, and surrounding context lines
- Match highlighting with ReDoS protection
- Pagination (10 per page)
- Copy to clipboard, open in editor, reveal in file manager
- File-preview modal with syntax highlighting via highlight.js

**Under the hood**
- Parallel worker pool sized to CPU count
- Line-by-line streaming for files > 1 MB (flat memory usage)
- Early termination via context cancellation
- Path-traversal protection and input sanitization
- Real-time progress and log streaming over HTTP polling
- Recent searches persisted in browser `localStorage`

## Tech stack

| Layer         | Technology                                   |
| ------------- | -------------------------------------------- |
| Backend       | Go 1.25, logrus, nxadm/tail                  |
| Frontend      | Vue 3, TypeScript, Vite, highlight.js         |
| Bridge        | Wails v2 (generated TypeScript bindings)      |
| Backend tests | Go `testing`                                 |
| Frontend tests| Vitest + @vue/test-utils (jsdom)             |

## Quick start

```bash
# Prerequisites: Go 1.25+, Node 16.x+, Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone <repo-url> && cd code-search-golang-wails-vue
go mod tidy
cd frontend && npm install && cd ..

wails dev      # hot-reload development server
wails build    # production binary in build/bin/
```

## Usage

1. Click **Browse** to pick a directory (native OS dialog).
2. Enter a query, toggle **Regex Search** if needed.
3. Optionally set extension, exclude patterns, or other filters.
4. Click **Search Code** — progress updates in real time.

Results show the match with context. Click any result to open the file preview modal with syntax highlighting. Use the editor dropdown to open the file in a detected editor (VS Code, VSCodium, Sublime, JetBrains IDEs, Neovim, Emacs, and many more).

### Search options

| Option              | Description                           | Default |
| ------------------- | ------------------------------------- | ------- |
| Case Sensitive      | Distinguish upper/lower case          | off     |
| Regex Search        | Treat query as regular expression     | off     |
| Include Binary      | Include binary files in search        | off     |
| Search Subdirs      | Recurse into subdirectories           | on      |
| Max File Size       | Skip files larger than this           | 10 MB   |
| Min File Size       | Skip files smaller than this          | 0       |
| Max Results         | Stop after this many matches          | 1000    |
| File Type Allow-List| Only search these extensions          | all     |
| Exclude Patterns    | Glob patterns to skip                 | none    |

## Project structure

```
.
├── main.go                  # Entry point: polling server + Wails app
├── app_core.go              # App struct, lifecycle, search cancellation
├── models.go                # SearchRequest / SearchResult / types
├── search_engine.go         # SearchWithProgress, worker pool, streaming
├── system_integration.go    # Directory dialog, editor detection (22 editors)
├── logger_utils.go          # Logger, isBinary, pattern matching, validation
├── polling_server.go        # HTTP log polling (port 34116)
├── app.go                   # Linux: ShowInFolder, open-in-editor
├── appWindows.go            # Windows: ShowInFolder, open-in-editor
├── *_test.go                # Backend test suites (incl. editor_detection_test.go)
├── go.mod / go.sum
├── wails.json
├── ARCHITECTURE_AND_TESTING_SUMMARY.md  # Top-level summary
├── docs/
│   ├── ARCHITECTURE.md      # Full architecture documentation
│   ├── TESTING.md           # Testing documentation
│   └── DEVELOPMENT.md       # Development workflow
└── frontend/
    ├── src/
    │   ├── main.ts          # Entry point
    │   ├── App.vue          # Root component
    │   ├── components/      # CodeSearch, SearchForm, SearchResults,
    │   │                    # ProgressIndicator, CodeModal, LogViewer, ...
    │   ├── composables/     # useSearch, useToast, useSyntaxHighlighting
    │   ├── services/        # syntax highlighting, app initialization
    │   ├── constants/ types/ utils/ assets/
    │   └── wailsjs/         # Generated Wails bindings
    └── tests/               # Vitest specs, mocks, fixtures
```

## Testing

```bash
# Go backend
go test -v ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Frontend
cd frontend && npm test
npm run test:watch      # watch mode

# Full validation (Go + Vitest + TypeScript check)
bash run_tests.sh
```

See [`docs/TESTING.md`](docs/TESTING.md) for detailed test coverage info.

## Documentation

| File | Contents |
| ---- | -------- |
| [`ARCHITECTURE_AND_TESTING_SUMMARY.md`](ARCHITECTURE_AND_TESTING_SUMMARY.md) | Top-level summary with quick reference. |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Full architecture (backend, frontend, communication, security). |
| [`docs/TESTING.md`](docs/TESTING.md) | Test suites, coverage, and infrastructure. |
| [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) | Setup, build, run, and conventions. |

## Platform notes

- **Linux**: file manager uses `xdg-open`; directory dialog via Wails.
- **Windows**: file manager uses `cmd /c start`; directory dialog via Wails.
- **macOS**: directory selection works via Wails. Folder reveal and open-in-editor are **not yet implemented**.

## Troubleshooting

- **No results?** Check the directory exists, query isn't too strict, and extension/exclude filters aren't removing expected files. Files > 10 MB are skipped.
- **Slow on large trees?** Add exclude patterns like `node_modules` and `.git`. Lower max results or simplify expensive regexes.
- **Build issues?** Run `go mod tidy && cd frontend && npm install`. Update Wails CLI with `go install github.com/wailsapp/wails/v2/cmd/wails@latest`.

## License

MIT
