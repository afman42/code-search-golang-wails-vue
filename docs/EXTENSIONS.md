# File extensions

The app tracks file extensions in four places, each answering a different question. This document explains what each list does, where it lives, and how to add a new extension.

## The four extension lists

| List | Location | Question it answers | Source of truth |
| ---- | -------- | ------------------- | --------------- |
| **Known-text set** | `text_extensions.go` → `knownTextExtensions` | Does this file skip the binary-detection probe during collection? | Backend map (single source) |
| **Allow-list UI** | `frontend/src/components/ui/PatternSelector.vue` | Which file types can the user filter a search by? | The backend known-text set, via the `GetKnownTextExtensions` binding. A short `fallbackAllowOptions` list applies only when that binding fails |
| **Symbol-scan set** | `symbol_scan.go` → `symbolSupportedExtensions` | Can the symbol extractor parse this file? | Backend slice (single source), paired with `getPatternsForExtension` in `symbols.go` |
| **Language detection** | `frontend/src/services/syntaxHighlightingService.ts` → `detectLanguage()` | Which highlight.js language renders this in the preview? | Hand-maintained map (extension → hljs language name) |

The three sets deliberately differ. The known-text set is broad (~170 entries) because "is this text?" has a broad answer. The symbol-scan set is narrow (10 entries) because it needs a hand-written regex grammar per language. The language-detection map is separate again because not every text extension has a highlight.js language — unmapped ones fall back to plain text.

## How they connect

```
┌─────────────────────────┐   GetKnownTextExtensions()    ┌──────────────────────────┐
│  text_extensions.go     │ ─────────────────────────────►│  useSearch.ts            │
│  knownTextExtensions    │   Wails binding (sorted,      │  data.knownTextExtensions│
│  (backend map)          │   no leading dot)             └────────────┬─────────────┘
└─────────────────────────┘                                           │
            │                                                         │ prop
            │  isKnownTextExtension(path)                             ▼
            ▼                                          ┌──────────────────────────┐
┌─────────────────────────┐                            │  SearchForm.vue          │
│  file_collection.go     │   skip binary probe for     │  :knownTextExtensions    │
│  walkDirectoryTree      │   known text extensions     └────────────┬─────────────┘
└─────────────────────────┘                                          │ prop
                                                                     ▼
                                                      ┌──────────────────────────┐
                                                      │  PatternSelector.vue     │
                                                      │  allow-list dropdown     │
                                                      │  (fallback only on error)│
                                                      └──────────────────────────┘

┌─────────────────────────┐  isSymbolSupportedExtension  ┌──────────────────────────┐
│  symbol_scan.go         │ ────────────────────────────►│  symbols.go              │
│  symbolSupported        │  + getPatternsForExtension    │  extractSymbolsFromFile  │
│  Extensions             │                               └──────────────────────────┘
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

The allow/exclude pattern controls live in `PatternSelector.vue`, composed into `SearchForm.vue`. The allow-list dropdown renders the **backend** known-text set, so the options a user can pick are exactly the extensions the collection phase treats as text:

```ts
const availableAllowOptions = computed(() =>
  props.knownTextExtensions.length > 0
    ? props.knownTextExtensions.map((ext) => (ext.startsWith('.') ? ext : `.${ext}`))
    : fallbackAllowOptions,
);
```

- The binding returns extensions without a leading dot; the component adds one, because `matchExtension` accepts either form and every other extension shown in this UI is dotted.
- `fallbackAllowOptions` is a short hardcoded list (`.go .ts .tsx .js .vue .py .java .css .html`) used **only** when the prop is empty — a failed binding, or a mock that does not implement it. The dropdown stays usable either way.
- The dropdown renders when no allow types are selected; a free-text input accepts any custom type (e.g. `min.js`, `tar.gz`, `backup.txt`) — multi-dot extensions work via `getFullExtension()` in `logger_utils.go`.
- Selected types flow into `SearchRequest.AllowedFileTypes` and filter files during the directory walk.

### Loading flow

`useSearch.ts` calls the binding alongside editor detection on init and stores the full set in state:

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

From there it is passed down as a prop: `SearchForm.vue` receives `data` and forwards `:knownTextExtensions` to `PatternSelector.vue`. Failure is non-fatal by design — the fallback list and the custom-type input both still work.

## Symbol-scan set (backend)

**File**: `symbol_scan.go`

`symbolSupportedExtensions` lists the ten extensions the symbol extractor can parse: `.go .ts .tsx .js .vue .py .rs .java .cs .rb`. `isSymbolSupportedExtension(path)` is the case-insensitive predicate over it, and `symbols.go`'s `getPatternsForExtension` maps each extension to a precompiled `[]patternConfig`.

This set is much narrower than the known-text set because every entry needs a hand-written regex grammar. A `.md` or `.json` file is unambiguously text but has no symbol declarations to extract, so it belongs in `knownTextExtensions` and not here.

`skipSymbolScanDirs` in the same file is the companion skip-list, grouped by ecosystem: `node_modules`, `.git`, `vendor`, `build`, `dist`, `bin`, plus `__pycache__`, `.venv`, `venv`, `.mypy_cache`, `.pytest_cache` (Python), `target` (cargo and maven), `.gradle` (JVM), and `obj` (.NET).

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

1. **Backend — `text_extensions.go`**: add `".foo": true` to `knownTextExtensions`. This makes the collection phase skip the binary probe for `.foo` files, and — because the allow-list dropdown is driven by this set — it also appears as a selectable file type in the UI with no frontend change.

2. **Frontend — `syntaxHighlightingService.ts`** (only if a highlighter exists): add an entry to the `languages` map in `detectLanguage()` (e.g. `foo: "ini"`), and import + register the corresponding highlight.js module in `loadHighlightJs()` (e.g. `await import("highlight.js/lib/languages/ini")` → `hljsModule.registerLanguage("ini", iniLang.default)`). If no highlight.js language fits, skip this step — the preview falls back to plain text.

3. **Backend — `symbol_scan.go` + `symbols.go`** (only if the type has symbol declarations worth indexing): add `".foo"` to `symbolSupportedExtensions`, add a precompiled `fooPatterns` block to the package-level `var (...)` group in `symbols.go` (never `MustCompile` inside a function — that was a real performance bug), wire a `case ".foo":` into `getPatternsForExtension`, and make sure every `keyword` you use is mapped in `GetSymbolType`, or the type falls through to substring sniffing.

4. **Tests**: if the extension should be covered by `TestIsKnownTextExtension` in `file_collection_test.go`, add it to the `textExts` slice. If `detectLanguage` gained a new mapping, add a case to `CodeModal.spec.ts`'s language-detection table. `TestGetPatternsForExtension` iterates `symbolSupportedExtensions`, so a new symbol language is checked for patterns automatically — but add a fixture-based extraction test for it.

## Multi-dot extensions

`getFullExtension()` in `logger_utils.go` extracts the full extension sequence from a filename — `file.min.js` returns `.min.js`, `archive.tar.gz` returns `.tar.gz`. `matchExtension()` checks both the final extension and the full sequence, so users can type `min.js` or `tar.gz` in the custom-type field to filter by compound extensions.
