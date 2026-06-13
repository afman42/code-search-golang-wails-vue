# Code Search

A desktop application for searching text and regular expressions across code files. Built with [Wails](https://wails.io/) — a Go backend for fast, secure file system operations and a Vue 3 + TypeScript frontend for the UI.

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Installation](#installation)
- [Usage](#usage)
- [Development](#development)
- [Project Structure](#project-structure)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features

**Search**

- Plain-text and regular-expression search
- Case-sensitive / case-insensitive toggle
- Filter by file extension, including multi-part extensions (`.tar.gz`, `.min.js`)
- File-type allow-lists to restrict which extensions are searched
- Exclude patterns (e.g. `node_modules`, `.git`, `*.log`) via a multi-select dropdown plus custom entries
- Optional subdirectory search
- Configurable min/max file size and a maximum result count
- Optional inclusion of binary files (skipped by default via binary detection)

**Results**

- File path, line number, matched text, and surrounding context lines
- Match highlighting
- Pagination (10 results per page)
- Copy a matched line to the clipboard
- Open the file in a detected editor, or reveal it in the system file manager
- File-preview modal with syntax highlighting (highlight.js)

**Under the hood**

- Parallel file processing with a worker pool sized to available CPU cores
- Line-by-line streaming for files larger than 1 MB to keep memory usage flat
- Early termination once the result limit is reached, via context cancellation
- Path-traversal protection and input validation on the backend
- Real-time progress and log streaming over an HTTP polling channel
- Recent searches persisted in browser `localStorage`

## Tech Stack

| Layer         | Technology                                      |
| ------------- | ----------------------------------------------- |
| Backend       | Go 1.23, logrus, nxadm/tail                      |
| Frontend      | Vue 3, TypeScript, Vite, highlight.js            |
| Bridge        | Wails v2 (generated TypeScript bindings)         |
| Backend tests | Go `testing`                                     |
| Frontend tests| Vitest + @vue/test-utils (jsdom)                 |

## Installation

### Prerequisites

- **Go** 1.23 or higher
- **Node.js** 16.x or higher (with npm)
- **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Linux only**: a directory dialog helper is not required (Wails provides a native dialog), but revealing files in the file manager uses `xdg-open`

### Build from source

```bash
git clone <repository-url>
cd code-search-golang-wails-vue

# Backend dependencies
go mod tidy

# Frontend dependencies
cd frontend && npm install && cd ..

# Build a production binary
wails build
```

The executable is written to `build/bin/` with a platform-specific name (e.g. `code-search-golang` or `code-search-golang.exe`).

## Usage

1. Click **Browse** to choose a directory (native dialog provided by Wails).
2. Enter a search query. Enable **Regex Search** to treat it as a regular expression.
3. Optionally set a file extension, exclude patterns, or other options.
4. Click **Search Code**. Progress is shown in real time.

### Search options

| Option              | Description                                            | Default |
| ------------------- | ------------------------------------------------------ | ------- |
| Case Sensitive      | Treat upper/lower case as distinct                     | off     |
| Regex Search        | Interpret the query as a regular expression            | off     |
| Include Binary      | Include binary files in the search                     | off     |
| Search Subdirs      | Recurse into subdirectories                            | on      |
| Max File Size       | Skip files larger than this                            | 10 MB   |
| Min File Size       | Skip files smaller than this                           | 0       |
| Max Results         | Stop after this many matches                           | 1000    |
| File Type Allow-List| Only search these extensions (empty = all)             | empty   |
| Exclude Patterns    | Glob patterns to skip                                  | empty   |

### Working with results

- Results show the file path, line number, matched text, and context lines.
- Click a result to open the preview modal with syntax highlighting; large files are truncated to the first 10,000 lines in the preview.
- Use **Copy** to copy a matched line, or open the file in a detected editor / reveal it in the file manager.
- Use the pagination controls to move through pages of 10 results.

## Development

### Live development

```bash
wails dev
```

This starts a Vite dev server with hot reload for the frontend. Changes to Vue/TypeScript reload automatically; Go backend changes generally require restarting `wails dev`.

### Testing

**Backend (Go):**

```bash
go test -v ./...                                   # all tests
go test -v -run TestName                           # a single test
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

**Frontend (Vitest):**

```bash
cd frontend
npm test            # run once
npm run test:watch  # watch mode
```

## Project Structure

```
.
├── main.go                  # Entry point: starts polling server + Wails app
├── app_core.go              # App struct, lifecycle, search cancellation
├── models.go                # SearchRequest / SearchResult / progress types
├── search_engine.go         # SearchWithProgress, worker pool, streaming
├── system_integration.go    # Directory dialog, validation, editor detection
├── logger_utils.go          # Logging, isBinary, defaults, pattern matching
├── polling_server.go        # HTTP polling log server (port 34116)
├── app.go                   # Linux build: ShowInFolder, open-in-editor
├── appWindows.go            # Windows build: ShowInFolder, open-in-editor
├── *_test.go                # Backend test suites
├── go.mod / go.sum          # Go dependencies
├── wails.json               # Wails configuration
└── frontend/
    ├── src/
    │   ├── main.ts          # Frontend entry point
    │   ├── App.vue          # Root component
    │   ├── components/
    │   │   ├── CodeSearch.vue      # Main orchestrator
    │   │   ├── StartupLoader.vue
    │   │   └── ui/                 # SearchForm, SearchResults,
    │   │                           # ProgressIndicator, CodeModal, ...
    │   ├── composables/     # useSearch, useToast, useSyntaxHighlighting
    │   ├── services/        # syntax highlighting, etc.
    │   ├── constants/ types/ utils/ assets/
    │   └── wailsjs/         # Generated Wails bindings
    └── tests/               # Vitest specs, mocks, fixtures
```

## Configuration

Edit `wails.json` to change the application name, output filename, and frontend build commands. Window dimensions (1024×768) and title are set in `main.go`.

## Troubleshooting

**Search returns no results**

- Confirm the directory exists and is readable.
- Check the query for typos or an overly strict regex.
- Make sure extension and exclude-pattern filters aren't removing the files you expect.
- Files above the max file size (10 MB default) are skipped.

**Performance**

- For large trees, add exclude patterns such as `node_modules` and `.git`.
- Lower the max results, or simplify expensive regex patterns.
- Files over 1 MB are streamed line-by-line automatically.

**Platform notes**

- *Linux:* revealing files in the file manager uses `xdg-open`.
- *Windows:* uses `cmd /c start`; PowerShell ships with Windows 7+.
- *macOS:* directory selection works via the native Wails dialog. Revealing a file in Finder and "open in editor" are **not** implemented in the current build.

**Build issues**

- Run `go mod tidy` and `npm install` to resolve dependencies.
- Update the Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`.

## Contributing

1. Fork the repository and create a feature branch.
2. Set up the environment (`go mod tidy`, `cd frontend && npm install`).
3. Make your change, keeping frontend and backend concerns separate and preserving the existing input-validation / path-safety checks.
4. Add tests: Go tests for backend functions, Vitest specs for components and composables.
5. Run the suites and format the code:
   ```bash
   go test -v ./... && go fmt ./...
   cd frontend && npm test
   ```
6. Update `README.md` and `ARCHITECTURE_AND_TESTING_SUMMARY.md` if behavior changes.
7. Open a pull request with a clear description.

## License

MIT License — see the `LICENSE` file for details.

## Acknowledgments

- Built with [Wails](https://wails.io/)
- Frontend with [Vue 3](https://vuejs.org/) and TypeScript
- Syntax highlighting by [highlight.js](https://highlightjs.org/)
