# File extensions

The app tracks file extensions in three places, each with a distinct purpose. This document explains what each list does, where it lives, and how to add a new extension.

## The three extension lists

| List | Location | Purpose | Source of truth |
| ---- | -------- | ------- | --------------- |
| **Known-text set** | `text_extensions.go` → `knownTextExtensions` | Decide whether a file skips the binary-detection probe during collection | Backend map (single source) |
| **Allow-list dropdown** | `frontend/src/components/ui/SearchForm.vue` | Suggest file types the user can filter search by | Loaded from backend via `GetKnownTextExtensions()` binding |
| **Language detection** | `frontend/src/services/syntaxHighlightingService.ts` → `detectLanguage()` | Pick the highlight.js language for the preview modal | Hand-maintained map (extension → hljs language name) |

The known-text set and the dropdown share one source: the backend map. The language-detection map is separate because it answers a different question — not "is this text?" but "which syntax highlighter renders this?" — and not every text extension has a highlight.js language.

## How they connect

```
┌─────────────────────────┐   GetKnownTextExtensions()    ┌──────────────────────┐
│  text_extensions.go     │ ─────────────────────────────►│  SearchForm.vue      │
│  knownTextExtensions    │   Wails binding (sorted,      │  dropdown <option>   │
│  (backend map)          │   no leading dot)             │  v-for="ext in ..."  │
└─────────────────────────┘                               └──────────────────────┘
            │
            │  isKnownTextExtension(path)
            ▼
┌─────────────────────────┐
│  file_collection.go     │   skip binary probe for known
│  walkDirectoryTree      │   text extensions
└─────────────────────────┘

┌─────────────────────────┐   detectLanguage(filePath)    ┌──────────────────────┐
│  syntaxHighlighting     │ ─────────────────────────────►│  CodeModal.vue       │
│  Service.ts             │   extension → hljs language   │  preview highlight    │
└─────────────────────────┘                               └──────────────────────┘
```

## Known-text set (backend)

**File**: `text_extensions.go`

The `knownTextExtensions` map holds ~150 extensions that are universally text and never need the 512-byte binary probe. Adding an entry here means files with that extension skip the `open` + `read` + `close` syscall during the collection phase — a measurable speedup on large trees.

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

## Allow-list dropdown (frontend)

**File**: `frontend/src/components/ui/SearchForm.vue`

The dropdown renders from `data.knownTextExtensions`, which the `useSearch` composable loads from the backend on startup:

```vue
<select id="allowed-filetypes" @change="addAllowedTypeFromSelect">
  <option value="">Add common type...</option>
  <option v-for="ext in data.knownTextExtensions" :key="ext" :value="ext">
    {{ ext }}
  </option>
</select>
```

- The list is empty on first paint, then populates when the backend call resolves.
- A free-text input next to the dropdown accepts any custom type (e.g. `min.js`, `tar.gz`, `backup.txt`) — multi-dot extensions work via `getFullExtension()` in `logger_utils.go`.
- Selected types flow into `SearchRequest.AllowedFileTypes` and filter files during the directory walk.

### Loading flow

`useSearch.ts` calls the binding alongside editor detection on init:

```ts
const fetchKnownTextExtensions = async () => {
  try {
    const exts = await GoGetKnownTextExtensions();
    if (Array.isArray(exts)) {
      data.knownTextExtensions = exts;
    }
  } catch (error: any) {
    console.error("Failed to load known text extensions:", error);
  }
};
void fetchKnownTextExtensions();
```

If the call fails, the dropdown stays empty and the custom-type input still works — the failure is non-fatal.

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

1. **Backend — `text_extensions.go`**: add `".foo": true` to `knownTextExtensions`. This makes the collection phase skip the binary probe for `.foo` files and automatically adds `foo` to the UI dropdown via the binding.

2. **Frontend — `syntaxHighlightingService.ts`** (only if a highlighter exists): add an entry to the `languages` map in `detectLanguage()` (e.g. `foo: "ini"`), and import + register the corresponding highlight.js module in `loadHighlightJs()` (e.g. `await import("highlight.js/lib/languages/ini")` → `hljsModule.registerLanguage("ini", iniLang.default)`). If no highlight.js language fits, skip this step — the preview falls back to plain text.

3. **Tests**: if the extension should be covered by `TestIsKnownTextExtension` in `file_collection_test.go`, add it to the `textExts` slice. If `detectLanguage` gained a new mapping, add a case to `CodeModal.spec.ts`'s language-detection table.

## Multi-dot extensions

`getFullExtension()` in `logger_utils.go` extracts the full extension sequence from a filename — `file.min.js` returns `.min.js`, `archive.tar.gz` returns `.tar.gz`. `matchExtension()` checks both the final extension and the full sequence, so users can type `min.js` or `tar.gz` in the custom-type field to filter by compound extensions.
