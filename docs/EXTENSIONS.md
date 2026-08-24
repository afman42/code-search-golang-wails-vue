# File extensions

The app tracks file extensions in three places, each with a distinct purpose. This document explains what each list does, where it lives, and how to add a new extension.

## The three extension lists

| List | Location | Purpose | Source of truth |
| ---- | -------- | ------- | --------------- |
| **Known-text set** | `text_extensions.go` → `knownTextExtensions` | Decide whether a file skips the binary-detection probe during collection | Backend map (single source) |
| **Allow-list UI** | `frontend/src/components/ui/PatternSelector.vue` | Suggest file types the user can filter search by | Hand-maintained list (`availableAllowOptions`); the backend set is loaded but not rendered |
| **Language detection** | `frontend/src/services/syntaxHighlightingService.ts` → `detectLanguage()` | Pick the highlight.js language for the preview modal | Hand-maintained map (extension → hljs language name) |

The language-detection map is separate because it answers a different question — not "is this text?" but "which syntax highlighter renders this?" — and not every text extension has a highlight.js language. The allow-list UI (below) currently uses its own curated list rather than the backend map.

## How they connect

```
┌─────────────────────────┐   GetKnownTextExtensions()    ┌──────────────────────────┐
│  text_extensions.go     │ ─────────────────────────────►│  useSearch.ts            │
│  knownTextExtensions    │   Wails binding (sorted,      │  data.knownTextExtensions│
│  (backend map)          │   no leading dot)             │  (loaded, not rendered)  │
└─────────────────────────┘                               └──────────────────────────┘
            │
            │  isKnownTextExtension(path)
            ▼
┌─────────────────────────┐
│  file_collection.go     │   skip binary probe for known
│  walkDirectoryTree      │   text extensions
└─────────────────────────┘
┌─────────────────────────┐
│  PatternSelector.vue    │   allow-list UI renders its own
│  availableAllowOptions  │   curated extension list
└─────────────────────────┘

┌─────────────────────────┐   detectLanguage(filePath)    ┌──────────────────────┐
│  syntaxHighlighting     │ ─────────────────────────────►│  CodeModal.vue       │
│  Service.ts             │   extension → hljs language   │  preview highlight    │
└─────────────────────────┘                               └──────────────────────┘
```

## Known-text set (backend)

**File**: `text_extensions.go`

The `knownTextExtensions` map holds ~170 extensions that are universally text and never need the 512-byte binary probe. Adding an entry here means files with that extension skip the `open` + `read` + `close` syscall during the collection phase — a measurable speedup on large trees.

```go
var knownTextExtensions = map[string]bool{
    ".go":   true,
    ".txt":  true,
    ".vue":  true,
    // ...
    ".wasm": false, // explicitly NOT text
}
```

- **Keys** include the leading dot, lowercased.
- **Values** are `true` for text, `false` for explicit non-text (only `.wasm`).
- The check is case-insensitive: `.GO` and `.go` both match.
- Any extension **not** in the map gets the binary probe — the safe default.

### Exposing the set to the frontend

`GetKnownTextExtensions()` is a Wails-bound method on `App`:

```go
func (a *App) GetKnownTextExtensions() []string
```

- Returns the extensions **without** the leading dot (`"go"`, not `".go"`).
- Sorted alphabetically.
- Omits entries marked `false` (`.wasm`).
- Callable from the frontend as `window.go.main.App.GetKnownTextExtensions()`.

## Allow-list UI (frontend)

**File**: `frontend/src/components/ui/PatternSelector.vue`

The allow/exclude pattern controls live in `PatternSelector.vue`, composed into `SearchForm.vue`. The allow-list dropdown offers a small curated set of common extensions:

```ts
const availableAllowOptions = ['.go', '.ts', '.tsx', '.js', '.vue', '.py', '.java', '.css', '.html'];
```

- The dropdown renders these options when no allow types are selected; a free-text input accepts any custom type (e.g. `min.js`, `tar.gz`, `backup.txt`) — multi-dot extensions work via `getFullExtension()` in `logger_utils.go`.
- Selected types flow into `SearchRequest.AllowedFileTypes` and filter files during the directory walk.

### Loading flow

`useSearch.ts` still calls the binding alongside editor detection on init, and stores the full set in state:

```ts
const fetchKnownTextExtensions = async () => {
  try {
    const exts = await GoGetKnownTextExtensions();
    if (Array.isArray(exts)) {
      data.knownTextExtensions = exts;
    }
  } catch (error: unknown) {
    console.error("Failed to load known text extensions:", error);
  }
};
void fetchKnownTextExtensions();
```

The loaded list is available in state for future UI use, but the allow-list dropdown does not currently render it — it shows the curated `availableAllowOptions` above. If the backend call fails, nothing breaks; the curated list and custom-type input still work.

## Language detection (frontend)

**File**: `frontend/src/services/syntaxHighlightingService.ts`

`detectLanguage(filePath)` maps a file extension to a highlight.js language name:

```ts
const languages: Record<string, string> = {
  go: "go",
  ts: "typescript",
  vue: "html",
  toml: "ini",
  txt: "plaintext",
  // ...
};
return languages[ext] || "text";
```

- Extensions not in the map fall through to `"text"` (plain rendering).
- The corresponding highlight.js language module must be registered in `loadHighlightJs()` for highlighting to apply; otherwise `hljs.getLanguage()` returns undefined and the modal falls back to escaped text.
- Language modules are imported lazily on first highlight, so there is no up-front bundle cost.

## Adding a new extension

To support a new text file type end-to-end:

1. **Backend — `text_extensions.go`**: add `".foo": true` to `knownTextExtensions`. This makes the collection phase skip the binary probe for `.foo` files. (To also offer the type in the allow-list dropdown, add it to `availableAllowOptions` in `PatternSelector.vue` — the UI list is independent of the backend set.)

2. **Frontend — `syntaxHighlightingService.ts`** (only if a highlighter exists): add an entry to the `languages` map in `detectLanguage()` (e.g. `foo: "ini"`), and import + register the corresponding highlight.js module in `loadHighlightJs()` (e.g. `await import("highlight.js/lib/languages/ini")` → `hljsModule.registerLanguage("ini", iniLang.default)`). If no highlight.js language fits, skip this step — the preview falls back to plain text.

3. **Tests**: if the extension should be covered by `TestIsKnownTextExtension` in `file_collection_test.go`, add it to the `textExts` slice. If `detectLanguage` gained a new mapping, add a case to `CodeModal.spec.ts`'s language-detection table.

## Multi-dot extensions

`getFullExtension()` in `logger_utils.go` extracts the full extension sequence from a filename — `file.min.js` returns `.min.js`, `archive.tar.gz` returns `.tar.gz`. `matchExtension()` checks both the final extension and the full sequence, so users can type `min.js` or `tar.gz` in the custom-type field to filter by compound extensions.
