# Code Search

A cross-platform desktop app for searching text and regular expressions across code files. Built with [Wails](https://wails.io/) — Go backend + Vue 3 frontend.

## Features

**Search engine**
- Plain-text and regex search with case-sensitivity toggle
- File extension filter, file-type allow-lists, and exclude patterns (e.g. `node_modules`, `.git`, `*.log`)
- Min/max file size, max result limit
- Binary file inclusion (off by default with binary detection)
- **Streamed results**: matches render progressively as workers find them (`search-results` batches) instead of appearing only when the whole search finishes
- **Skipped-file reporting**: files that could not be read are counted exactly and a capped sample of their paths is listed, so "no matches here" is never confused with "could not look"
- Directory validated before the search runs, so a mistyped path fails with a specific message instead of an empty result set

**Results & preview**
- File path, line number, matched text, and surrounding context lines
- **Inline diff view**: Color-coded context (before/after) with line numbers and copy buttons
- Match highlighting with ReDoS protection
- Pagination (10 per page)
- Copy to clipboard, open in editor, reveal in file manager
- **Fuzzy search**: finds near-miss matches for typos (backend returns candidates, frontend scores/badges them; toggle via checkbox)
- File-preview modal with syntax highlighting via highlight.js (renders only when open)
- Modal match navigation (prev/next with Ctrl+↑/↓), jump-to-line via an inline input (offered for files over 50 lines, where the navigation controls mount), and a working line-number toggle
- **File Explorer tree** in the preview modal: browse the files found by the search (folders first, alphabetical), expand/collapse directories, filter by name, and click any file to load it in the preview
- **Find & Replace**: literal replacement across matched lines with a dry-run preview and explicit atomic apply. Cancellable, with per-phase progress; regex mode disables it and VCS is the undo path

**UI & design system**
- Central design-system token set in `frontend/src/style.css` (palette, spacing, radii, shadows, fonts, light/dark-surface and sidebar tokens) — all components reference tokens instead of hard-coded colors
- Responsive CSS grid app layout (`CodeSearch.vue`): sticky sidebar + content column, stacking to a single column on narrow screens
- **Recent-search suggestions**: a query-input dropdown lists recent searches on focus, closes on outside-click/Escape, and fills + runs the query when selected

**Symbol search**
- Search code symbols — functions, classes, variables, consts, interfaces, types — by name across Go, TypeScript, JavaScript, Vue, Python, Rust, Java, C#, and Ruby files under the selected directory
- "Load All Symbols" to index the whole tree; results show name, type, signature, and `file:line`
- Real per-file indexing progress streamed from the backend via `symbol-progress` events
- **Persistent symbol index**: extracted symbols are cached per directory (keyed by file fingerprints), so repeat searches are instant. A **Re-index** button clears the backend cache when files changed outside the app
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
- Path-traversal protection (raw-input `..` check + `filepath.Clean` + prefix guard) and symlink-aware collection (symlinks skipped, base dir resolved via `EvalSymlinks`)
- Protected-directory guard blocks system roots and their subtrees (`/etc` → `/etc/ssh`)
- Result cap at 10 000 matches (bounds memory; default 1 000)
- Query length cap at 2 000 chars, enforced in `validateAndSetDefaults` and mirrored in the frontend validator
- Input sanitization and CSV formula-injection guard (`csvSafeCell` trims leading spaces before checking `=+-@\t\r`)
- Content-Security-Policy meta in `frontend/index.html` (`default-src 'self'`) plus per-segment `escapeHtml` before DOMPurify in `diffUtils.renderDiffHtml`
- Real-time log streaming via Wails bindings (IPC — no HTTP server)
- Log file rotation at 10 MB (bounds disk usage on long-running installs)
- Recent searches persisted in browser `localStorage`
- **Two-phase file collection** (3.6x faster than single-pass):
  - Phase 1: single-threaded directory walk with cheap filters (extension, size, exclude patterns)
  - Phase 2: parallel binary detection via worker pool (only for unknown extensions)
- **Known-text extension shortcut**: ~170 text extensions (.go, .ts, .py, .md, .vue, .toml, .txt, etc.) skip the binary probe entirely — no open/read/close syscall
- **Persistent collection cache**: repeat searches in an unchanged directory skip the walk + binary probe (fingerprint-validated, filter-aware; see `collection_index.go`). Cache reads return a copy, so callers can sort/append without corrupting the entry
- **Respect .gitignore option**: honors the full chain of `.gitignore` files from the search root down to each file's own directory (deeper files override shallower ones, `!negation` re-includes) plus the root `.git/info/exclude`, via go-gitignore. Ignored directories are pruned during the walk, mirroring git
- **Known-text set drives the UI**: the backend's `GetKnownTextExtensions()` binding exposes the ~170-entry known-text set and the allow-list dropdown renders it, so the UI can no longer drift from the set that decides what gets collected
- **Table-driven editor dispatch**: `OpenInEditorByName` is the sole Wails binding for opening files in editors; the table-driven `editorCatalog` (command + args per editor) replaces 17 per-editor wrapper methods, with a `"JetBrains"` file-extension router
- **Zombie-safe process launching**: every external process (editors, `xdg-open`, `explorer`, `open`) starts via `startAndReap` (`Start` + async `Wait`), so short-lived helpers are reaped instead of leaking zombies; `appendPath` copies the shared editor args so concurrent launches can't corrupt each other
- **Shared symbol-scan constants**: `symbol_scan.go` holds the single source of truth for skip-dirs and supported extensions, used by both `symbols.go` and `symbol_index.go`
- **Zero-allocation path resolution**: absolute base directory computed once, not per file
- **`useLogStreaming` composable**: encapsulates log-parsing, polling interval, and lifecycle — keeps components thin and logic testable
- **Typed IPC boundary**: shared `errorUtils` helpers (`toErrorMessage`, `asRecord`) narrow untyped Wails payloads; no `any` in `src`, with `noUnusedLocals`/`noUnusedParameters` enforced
- **Debounced fuzzy helper**: `fuzzyMatch.ts` exports a `debounce` utility and `DebouncedFn` type for throttling expensive re-scoring
## Tech stack

| Layer         | Technology                                   |
| ------------- | -------------------------------------------- |
| Backend       | Go 1.25, logrus, nxadm/tail                  |
| Frontend      | Vue 3, TypeScript, Vite, highlight.js         |
| Bridge        | Wails v2 (generated TypeScript bindings)      |
| Backend tests | Go `testing` (39 test files, 80.0% statement coverage) |
| Frontend tests| Vitest + @vue/test-utils (48 test files, 714 tests) |
| E2E tests     | Playwright (41 flow tests across 7 specs, mock backend) |

## Quick start

```bash
# Prerequisites: Go 1.25+ (go.mod pins 1.25.0; golangci-lint v2 and staticcheck
# need Go >= 1.26, and CI's setup-go uses 1.26), Node 24.x+, Wails CLI
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
| Fuzzy Search        | Append near-miss typo candidates      | off     |
| Include Binary      | Include binary files in search        | off     |
| Respect .gitignore  | Exclude files matched by the `.gitignore` chain (root down to each file's own directory) + root `.git/info/exclude` | off |
| Max File Size       | Skip files larger than this           | 10 MB   |
| Min File Size       | Skip files smaller than this          | 0       |
| Max Results         | Stop after this many matches          | 1000 (cap 10000) |
| File Type Allow-List| Only search these extensions          | all     |
| Exclude Patterns    | Glob patterns to skip                 | none    |
| File Extension      | Single-extension filter (e.g. `.go`)  | none    |

`GetDirectoryContents` — used for directory listings rather than search — is bounded independently at 50 000 entries and 32 levels of depth, and returns an error rather than a silently truncated list when either bound is hit.
## Project structure

```
.
├── main.go                  # Entry point: log tailing + Wails app
├── app_core.go              # App struct, lifecycle, search cancellation, LRU pattern cache
├── models.go                # SearchRequest / SearchResult / types
├── search_engine.go         # SearchWithProgress orchestration, createSearchContext, CancelSearch
├── search_workers.go        # Worker pool: file processing, result/progress emission (50ms throttle)
├── search_streaming.go      # Line-by-line streaming path for files > 1 MB
├── search_fuzzy.go          # Fuzzy near-miss phase: sliding-window scoring + quota
├── search_context.go        # Shared context-window helpers + binary-probe buffer pool
├── file_collection.go       # Two-phase file collection: walk + parallel binary probe
├── collection_index.go      # Persistent collection cache (fingerprint + filter-keyed)
├── gitignore.go             # Root .gitignore + .git/info/exclude support (go-gitignore)
├── replace.go               # ReplaceInFiles binding: literal replace, dry-run + atomic apply
├── text_extensions.go       # ~170 known-text extensions + GetKnownTextExtensions binding
├── system_integration.go    # Directory dialog, editor detection (22 editors), ReadFile, OpenInEditorByName dispatcher
├── app_symbols.go           # Symbol-search Wails bindings (GetAllSymbols, SearchSymbols)
├── symbols.go               # Symbol extraction (Go/TS/JS/Vue) + progress scan + cache check
├── symbol_index.go          # Persistent symbol index: fingerprint-based dir cache
├── symbol_scan.go           # Shared skip-dirs set + symbol-supported-extensions (single source of truth)
├── export.go                # ExportSearchResults binding (CSV/JSON via SaveFileDialog)
├── logger_utils.go          # Logger, isBinary, pattern matching, validation, log rotation
├── polling_server.go        # Log buffer management + file tailing (no HTTP server)
├── app.go                   # Linux: ShowInFolder, open-in-editor, OpenInDefaultEditor
├── appWindows.go            # Windows: ShowInFolder, open-in-editor, OpenInDefaultEditor
├── appDarwin.go             # macOS: ShowInFolder, open-in-editor, OpenInDefaultEditor
├── app_shared.go            # Shared path validation + editor PATH lookup + zombie-safe runCommand + appendPath
├── *_test.go                # Backend test suites (36 files)
├── go.mod / go.sum
├── wails.json
├── .golangci.yml            # golangci-lint v2 config (errcheck/staticcheck narrowing)
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
    │   ├── composables/     # useSearch, useReplace, useEditorDetection, useLogStreaming, useLogViewer, useToast, useCodeHighlighting, useMatchNavigation, useFilePreview, useSelectionManager, useSymbolSearch, useTheme, useKeyboardShortcuts
    │   ├── services/        # syntax highlighting, app initialization
    │   ├── mocks/           # wailsMock.ts — browser stand-in for the Go backend (E2E/dev)
    │   ├── constants/ types/ utils/ assets/    # utils/diffUtils.ts, errorUtils.ts, fuzzyMatch.ts, localStorageUtils.ts, ...
    │   └── wailsjs/         # Generated Wails bindings
    ├── tests/               # Vitest specs (48 files), mocks, fixtures, services
    └── playwright-tests/    # Playwright E2E flow specs
```

## Testing

```bash
# Go backend
go test -race -covermode=atomic -coverprofile=coverage.out -timeout 600s ./...
go tool cover -html=coverage.out    # -covermode=atomic is required with -race

# Frontend
cd frontend && npm test
npx vitest              # watch mode

# Full validation (Go + Vitest + TypeScript check)
bash run_tests.sh

# Include Playwright E2E flows (needs system Chrome; starts vite with the mock backend)
RUN_E2E=1 bash run_tests.sh

# E2E only, or the mocked frontend in a browser for manual testing
cd frontend && npm run test:e2e
npm run dev:mock
```

Currently 80.0% Go statement coverage, against an 80% gate enforced in CI. See [`docs/TESTING.md`](docs/TESTING.md) for the full suite breakdown and the known gaps.

## Lint & vulnerability checks

```bash
gofmt -l .                # must print nothing
go vet ./...
golangci-lint run ./...   # errcheck + staticcheck + unused (config: .golangci.yml)
staticcheck ./...
govulncheck ./...
```

All five are clean on `main` and run in CI before the Go tests. `golangci-lint` v2 and `staticcheck` need Go ≥ 1.26 (`go install` switches toolchain automatically). See [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) for what `.golangci.yml` narrows and why.

## Documentation

| File | Contents |
| ---- | -------- |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Full architecture (backend, frontend, communication, security). |
| [`docs/FEATURES.md`](docs/FEATURES.md) | Feature reference (search, results, symbol search, UX). |
| [`docs/EXTENSIONS.md`](docs/EXTENSIONS.md) | File-extension system: backend known-text set, UI dropdown, language detection. |
| [`docs/TESTING.md`](docs/TESTING.md) | Test suites, coverage, and infrastructure. |
| [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) | Setup, build, run, and conventions. |

## Platform notes

CI builds **and tests** on `ubuntu-latest`, `windows-latest`, and `macos-latest` (`fail-fast: false`), which is what compiles the build-tagged platform files at all — `appDarwin.go` and `appWindows.go` are invisible to a Linux-only build.

- **Linux**: file manager and open-in-default-editor use `xdg-open` (paths validated before launch); directory dialog via Wails.
- **Windows**: file manager uses `explorer`; open-in-default-editor uses `ShellExecute` (no shell parsing, so no injection surface); directory dialog via Wails.
- **macOS**: directory selection works via Wails. Folder reveal uses `open -R` (Finder); open-in-editor needs editor CLIs on PATH (open-in-default-editor uses `open`).

## Troubleshooting

- **No results?** Check the directory exists, query isn't too strict, and extension/exclude filters aren't removing expected files. Files > 10 MB are skipped.
- **Slow on large trees?** Add exclude patterns like `node_modules` and `.git`. Lower max results or simplify expensive regexes.
- **Build issues?** Run `go mod tidy && cd frontend && npm install`. Update Wails CLI with `go install github.com/wailsapp/wails/v2/cmd/wails@latest`.

## License

MIT
