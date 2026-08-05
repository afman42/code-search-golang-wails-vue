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
- **Inline diff view**: Color-coded context (before/after) with line numbers and copy buttons
- Match highlighting with ReDoS protection
- Pagination (10 per page)
- Copy to clipboard, open in editor, reveal in file manager
- **Fuzzy search**: Find matches despite typos (toggle via checkbox)
- File-preview modal with syntax highlighting via highlight.js (renders only when open)
- Modal match navigation (prev/next with Ctrl+↑/↓), jump-to-line with a flash highlight, and a working line-number toggle
- **File Explorer tree** in the preview modal: browse the files found by the search (folders first, alphabetical), expand/collapse directories, and click any file to load it in the preview

**UI & design system**
- Central design-system token set in `frontend/src/style.css` (palette, spacing, radii, shadows, fonts, light/dark-surface and sidebar tokens) — all components reference tokens instead of hard-coded colors
- Responsive CSS grid app layout (`CodeSearch.vue`): sticky sidebar + content column, stacking to a single column on narrow screens
- **Recent-search suggestions**: a query-input dropdown lists recent searches on focus, closes on outside-click/Escape, and fills + runs the query when selected

- **Symbol search**
- Search code symbols — functions, classes, variables, consts, interfaces, types — by name across Go, TypeScript, JavaScript, and Vue files under the selected directory
- "Load All Symbols" to index the whole tree; results show name, type, signature, and `file:line`
- Real per-file indexing progress streamed from the backend via `symbol-progress` events
- **Persistent symbol index**: extracted symbols are cached per directory (keyed by file fingerprints), so repeat searches are instant
- Click any symbol to jump to its file:line in the code preview modal with a flash highlight

**Export & multi-directory**
- **Multi-select results**: checkboxes on each result; copy selected lines or export selected/all as CSV or JSON via a native save-file dialog
- **Multi-directory search**: enter additional directory paths (one per line) to search across multiple project roots in one pass

**Log viewer**
- Live log streaming with level filter, text search, and pause-on-tail (auto-scroll toggle)

**Under the hood**
- Parallel worker pool sized to CPU count
- Line-by-line streaming for files > 1 MB (flat memory usage)
- Early termination via context cancellation
- Path-traversal protection and input sanitization
- Real-time log streaming via Wails bindings (IPC — no HTTP server)
- Recent searches persisted in browser `localStorage`
- **Two-phase file collection** (3.6x faster than single-pass):
  - Phase 1: single-threaded directory walk with cheap filters (extension, size, exclude patterns)
  - Phase 2: parallel binary detection via worker pool (only for unknown extensions)
- **Known-text extension shortcut**: ~170 text extensions (.go, .ts, .py, .md, .vue, .toml, .txt, etc.) skip the binary probe entirely — no open/read/close syscall
- **Single source of truth for file types**: the backend's known-text set drives the UI's "Allowed File Types" dropdown via a Wails binding — the suggestion list can't drift from what the backend actually treats as text
- **Zero-allocation path resolution**: absolute base directory computed once, not per file
- **`useLogStreaming` composable**: encapsulates log-parsing, polling interval, and lifecycle — keeps components thin and logic testable
- **Typed IPC boundary**: shared `errorUtils` helpers (`toErrorMessage`, `asRecord`) narrow untyped Wails payloads; no `any` in `src`, with `noUnusedLocals`/`noUnusedParameters` enforced

## Tech stack

| Layer         | Technology                                   |
| ------------- | -------------------------------------------- |
| Backend       | Go 1.25, logrus, nxadm/tail                  |
| Frontend      | Vue 3, TypeScript, Vite, highlight.js         |
| Bridge        | Wails v2 (generated TypeScript bindings)      |
| Backend tests | Go `testing` (23 test files)                 |
| Frontend tests| Vitest + @vue/test-utils (30 test files, 456 tests) |
| E2E tests     | Playwright (9 flow tests against a mocked backend) |

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
├── main.go                  # Entry point: log tailing + Wails app
├── app_core.go              # App struct, lifecycle, search cancellation, LRU pattern cache
├── models.go                # SearchRequest / SearchResult / types
├── search_engine.go         # SearchWithProgress, worker pool, streaming, progress throttle
├── file_collection.go       # Two-phase file collection: walk + parallel binary probe
├── text_extensions.go       # ~170 known-text extensions + GetKnownTextExtensions binding
├── system_integration.go    # Directory dialog, editor detection (22 editors), ReadFile
├── app_symbols.go           # Symbol-search Wails bindings (GetAllSymbols, SearchSymbols)
├── symbols.go               # Symbol extraction (Go/TS/JS/Vue) + progress scan + cache check
├── symbol_index.go          # Persistent symbol index: fingerprint-based dir cache
├── export.go                # ExportSearchResults binding (CSV/JSON via SaveFileDialog)
├── logger_utils.go          # Logger, isBinary, pattern matching, validation
├── polling_server.go        # Log buffer management + file tailing (no HTTP server)
├── app.go                   # Linux: ShowInFolder, open-in-editor
├── appWindows.go            # Windows: ShowInFolder, open-in-editor
├── appDarwin.go             # macOS: ShowInFolder, open-in-editor, OpenInDefaultEditor
├── *_test.go                # Backend test suites (23 files)
├── go.mod / go.sum
├── wails.json
├── run_tests.sh             # Full validation (Go + Vitest + tsc; RUN_E2E=1 adds Playwright)
├── docs/
│   ├── ARCHITECTURE.md      # Full architecture documentation
│   ├── FEATURES.md          # Feature reference
│   ├── EXTENSIONS.md        # File-extension system
│   ├── TESTING.md           # Testing documentation
│   ├── TESTING_GAPS.md      # Coverage gaps and targets
│   └── DEVELOPMENT.md       # Development workflow
└── frontend/
    ├── src/
    │   ├── main.ts          # Entry point (installs mock backend when VITE_WAILS_MOCK set)
    │   ├── App.vue          # Root component
    │   ├── style.css        # Design-system tokens: palette, spacing, radii, shadows, fonts, dark surfaces
    │   ├── components/      # UI components: SearchForm (+ modular children), SymbolSearch, LogViewer, CodeModal, InlineDiffView, ...
    │   ├── composables/     # useSearch, useEditorDetection, useLogStreaming, useToast, useCodeHighlighting, useMatchNavigation, useFilePreview, useTheme, useKeyboardShortcuts
    │   ├── services/        # syntax highlighting, app initialization
    │   ├── mocks/           # wailsMock.ts — browser stand-in for the Go backend (E2E/dev)
    │   ├── constants/ types/ utils/ assets/    # utils/diffUtils.ts, errorUtils.ts, fuzzyMatch.ts, localStorageUtils.ts, ...
    │   └── wailsjs/         # Generated Wails bindings
    ├── tests/               # Vitest specs, mocks, fixtures
    └── playwright-tests/    # Playwright E2E flow specs
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

# Include Playwright E2E flows (needs system Chrome; starts vite with the mock backend)
RUN_E2E=1 bash run_tests.sh

# E2E only, or the mocked frontend in a browser for manual testing
cd frontend && npm run test:e2e
npm run dev:mock
```

See [`docs/TESTING.md`](docs/TESTING.md) for detailed test coverage info.

## Documentation

| File | Contents |
| ---- | -------- |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Full architecture (backend, frontend, communication, security). |
| [`docs/FEATURES.md`](docs/FEATURES.md) | Feature reference (search, results, symbol search, UX). |
| [`docs/EXTENSIONS.md`](docs/EXTENSIONS.md) | File-extension system: backend known-text set, UI dropdown, language detection. |
| [`docs/TESTING.md`](docs/TESTING.md) | Test suites, coverage, and infrastructure. |
| [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) | Setup, build, run, and conventions. |

## Platform notes

- **Linux**: file manager uses `xdg-open`; directory dialog via Wails.
- **Windows**: file manager uses `explorer`; directory dialog via Wails.
- **macOS**: directory selection works via Wails. Folder reveal uses `open -R` (Finder); open-in-editor needs editor CLIs on PATH (open-in-default-editor uses `open`).

## Troubleshooting

- **No results?** Check the directory exists, query isn't too strict, and extension/exclude filters aren't removing expected files. Files > 10 MB are skipped.
- **Slow on large trees?** Add exclude patterns like `node_modules` and `.git`. Lower max results or simplify expensive regexes.
- **Build issues?** Run `go mod tidy && cd frontend && npm install`. Update Wails CLI with `go install github.com/wailsapp/wails/v2/cmd/wails@latest`.

## License

MIT
