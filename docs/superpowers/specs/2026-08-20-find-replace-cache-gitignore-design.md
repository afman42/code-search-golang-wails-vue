# Design: Find & Replace, Collection Cache, .gitignore Support

Date: 2026-08-20
Status: Approved for implementation (pending spec review)

## Goal

Add three backend capabilities to the code-search app, with minimal in-place UI
for the two that need a user trigger, plus full test coverage:

1. **Find & Replace** — literal replace across matched lines, dry-run preview
   then explicit apply. No backups (VCS is undo).
2. **Collection cache** — fingerprint-validated cache of the file-collection
   walk, so repeat searches in an unchanged directory skip the walk + binary
   probe.
3. **.gitignore support** — optional filter honoring the repo's root
   `.gitignore` + `.git/info/exclude`, gated by a request field (default off).

Non-goals: undo/backups for replace, regex-capture replace, nested per-directory
`.gitignore`, on-disk cache persistence, fsnotify invalidation.

## Constraint

Original request said "do not change UI/UX". User softened this to **minimal
in-place UI additions**: no layout restructure. #1 adds one input + two buttons
in the results header; #3 adds one checkbox beside the existing four. #2 is
invisible.

## Guiding principle

Reuse existing plumbing; add nothing speculative. Every new file mirrors an
existing pattern already in the repo.

---

## Feature 1 — Find & Replace

### Semantics

- **Literal only.** Replacement is a literal string, applied with
  `pattern.ReplaceAllLiteralString(line, replacement)`. The `pattern` is the
  exact `*regexp.Regexp` the search already compiled (`compileSearchPattern`),
  so case-sensitivity is honored identically to search.
- **Regex rejected.** If `req.UseRegex` is true, `ReplaceInFiles` returns an
  error `"replace is literal-only; disable regex search to replace"`. The UI
  disables the replace controls when regex mode is on.
- **Preview == Apply.** Replacement targets only the `(file, line)` pairs the
  search matched. The dry-run computes the exact new line content; apply writes
  that same content. There is no separate apply-time recomputation that could
  diverge.
- **No-op skip.** A matched line whose replacement equals the original is
  excluded from the result and never written.

### Backend — new file `replace.go`

Models (added to `models.go`, domain section):

```go
type FileReplacement struct {
    FilePath string `json:"filePath"`
    LineNum  int    `json:"lineNum"`
    OldLine  string `json:"oldLine"`
    NewLine  string `json:"newLine"`
}

type ReplaceRequest struct {
    Search      SearchRequest `json:"search"`
    Replacement string        `json:"replacement"`
    Apply       bool          `json:"apply"`
}

type ReplaceResult struct {
    Files        []FileReplacement `json:"files"`
    FilesChanged int               `json:"filesChanged"`
    LinesChanged int               `json:"linesChanged"`
}
```

`ReplaceInFiles(req ReplaceRequest) (ReplaceResult, error)`:

1. Reject if `req.Search.UseRegex` → literal-only error.
2. Reject if `req.Search.Query == ""` → "query is required".
3. Get matches WITHOUT the event-emitting binding. `SearchWithProgress` emits
   `search-progress` events and installs the search cancel-func on the `App`;
   calling it from replace would corrupt an in-flight search's cancel/progress
   state and flicker the UI. Instead, replace collects files via
   `collectFilesToProcess` and matches lines with the compiled pattern directly
   (the same primitives search uses, minus the event/cancel side effects). To
   avoid duplicating the match loop, extract the pure matcher search already
   uses into a helper both call.
4. Compile the pattern via `a.compileSearchPattern(req.Search)`.
5. Group matches by their absolute file path (see multi-directory note below).
6. For each matched file:
   - `sanitizePath` + `containsDotDotComponent` on the path (mirror `ReadFile`);
     reject traversal, skip missing files.
   - Read the file, split into lines.
   - For each matched line number in that file: compute
     `new := pattern.ReplaceAllLiteralString(old, req.Replacement)`; if
     `new != old`, record a `FileReplacement` and stage the new line.
   - If `Apply` and the file has >= 1 staged change: write atomically.
7. Return `ReplaceResult` with all diffs and counts. When `Apply` is false, no
   file is written.

**Multi-directory note:** search may span `req.Search.Directory` +
`req.Search.Directories`. Replace operates on the absolute file paths the match
phase already resolved — it does NOT recompute paths against a single base — so
multi-root replace is correct. Gitignore rel-path resolution (Feature 3) keys
each file off the search root it was collected under, not one global base.

Atomic write helper (in `replace.go`):

```go
// writeFileAtomic writes content to path via a temp file in the same
// directory + os.Rename, preserving the original file mode. Same-dir temp
// guarantees Rename is on one filesystem (atomic). VCS is the undo path;
// no .bak is written.
func writeFileAtomic(path string, content []byte, mode os.FileMode) error
```

Uses `os.CreateTemp(filepath.Dir(path), ".cs-replace-*")`, writes, `Chmod`,
`Close`, `os.Rename(tmp, path)`. On any error before rename, remove the temp
file.

### Backend — binding registration

`ReplaceInFiles` is a method on `*App`, so Wails auto-binds it. No manifest edit;
`wails dev`/`wails build` regenerates `frontend/wailsjs/go/main/App.*`.

### Frontend — composable `frontend/src/composables/useReplace.ts`

State: `replacement: string`, `preview: ReplaceResult | null`, `isReplacing:
bool`.

- `previewReplace()` — builds a `ReplaceRequest` from the current `SearchState`
  (`apply: false`), calls `GoReplaceInFiles`, stores `preview`. Toast on error.
- `applyReplace()` — same request with `apply: true`; on success toast
  `"Replaced N lines in M files"`, clear `preview`, re-run `searchCode()` so
  results reflect the new file contents.
- Both guarded: no-op + toast if `data.useRegex` is true.

### Frontend — UI in `SearchResults.vue`

In the existing results header (`.results-header`), add a replace row:

```html
<div class="replace-row" v-if="!data.useRegex">
  <input v-model="replacement" placeholder="Replace matches with…"
         :disabled="isReplacing" />
  <button @click="previewReplace" :disabled="isReplacing || !replacement">Preview Replace</button>
  <button @click="applyReplace" :disabled="isReplacing || !preview?.filesChanged">
    Apply {{ preview?.linesChanged ?? 0 }}
  </button>
</div>
```

Preview diffs render through the existing `InlineDiffView` old→new coloring
(reuse, not new component). Styling uses existing design-system tokens
(`--space-*`, `--color-*`); no layout grid change.

### Risk & mitigation

- Data loss: mitigated by atomic write, preview==apply, literal-only, no-op
  skip, path sanitization. No backups by explicit decision — VCS is undo.

---

## Feature 2 — Collection cache

### Design

New file `collection_index.go`, mirroring `symbol_index.go`:

```go
type probedFile struct {
    absPath string
    size    int64
    isText  bool
}

type collectionEntry struct {
    fingerprint string
    files       []probedFile
    createdAt   time.Time
}

type collectionCache struct {
    mu      sync.RWMutex
    entries map[string]*collectionEntry // keyed by directory
}
```

- `maxCollectionEntries = 8` (matches `maxSymbolIndexEntries`).
- Cache holds the **unfiltered** probed file set: every regular file under the
  directory that survives always-on skips (hidden dirs, symlinks), each tagged
  with whether the binary probe said it is text. Cheap, per-request filters
  (extension, size, exclude, allowed types, includeBinary, gitignore) are NOT
  baked into the cache — they are applied to the cached list on retrieval.
- `computeCollectionFingerprint(directory)` — metadata walk (path + size +
  modtime, all extensions, same skip rules as the collection walk). Reuses the
  hashing shape of `computeDirectoryFingerprint`.
- `get(dir, fp)` / `set(dir, fp, files)` — same lock + oldest-eviction pattern
  as `symbolIndexCache`.
- `globalCollectionIndex *collectionCache` on `App`, `nil` in unit tests that
  construct `App` directly (no caching path), matching how `globalSymbolIndex`
  behaves.

### Integration in `collectFilesToProcess`

1. Compute fingerprint for the directory.
2. On cache hit (fingerprint matches): take the cached `[]probedFile`, apply the
   per-request filters in memory, return the surviving `[]fileMeta`. Skip
   `walkDirectoryTree` and `probeBinaryInParallel` entirely.
3. On miss/stale: run the existing walk + parallel probe, build the full
   unfiltered `[]probedFile` (known-text extensions are `isText: true` without a
   probe; includeBinary path marks all as text), store it, then apply filters.
4. Trees over a file cap (`maxCachedFiles = 200_000`) are not cached (bounds
   memory); they fall through to the existing path.

### Honest ceiling (recorded as a `ponytail:` comment in code)

A cache hit still pays one metadata (fingerprint) walk, because that is how
staleness is detected. The saved work is the binary-probe phase and the
filtering, not the directory traversal. Win is marginal on pure known-text
trees, real on mixed-extension trees. Upgrade path: fsnotify-based invalidation
to drop the per-search fingerprint walk — only if that walk becomes the measured
bottleneck.

### Refactor to enable caching

`walkDirectoryTree` + `probeBinaryInParallel` are refactored so the merged,
unfiltered probed set can be produced and cached before per-request filtering.
The cheap filters (extension/size/exclude/allowed/includeBinary) move into a
single `filterProbedFiles(files, req, matcher)` function used by both the
cache-hit and cache-miss paths, keeping one filtering implementation (DRY).

---

## Feature 3 — .gitignore support

### Dependency

Add `github.com/sabhiram/go-gitignore` to `go.mod` (MIT, single file,
battle-tested). API used: `CompileIgnoreLines(lines ...string) *GitIgnore` and
`(*GitIgnore).MatchesPath(relPath string) bool`. `MatchesPath` expects a path
relative to the `.gitignore` location and normalizes OS separators internally.

### Request field

`SearchRequest.RespectGitignore bool` (`models.go` + `frontend/src/types/search.ts`),
default `false`. When false, behavior is byte-identical to today.

### Matcher construction

New helper (in `file_collection.go` or a small `gitignore.go`):

```go
// loadGitignoreMatcher builds a matcher from the directory's root .gitignore
// and .git/info/exclude. Missing files contribute no lines. Returns nil when
// no ignore source exists, so callers can skip matching entirely.
func loadGitignoreMatcher(directory string) *ignore.GitIgnore
```

Reads `<dir>/.gitignore` and `<dir>/.git/info/exclude`, concatenates their
lines, and calls `CompileIgnoreLines`. Built once per `collectFilesToProcess`
call.

### Filtering

Inside `filterProbedFiles`: when `req.RespectGitignore` and matcher is non-nil,
compute `rel, _ := filepath.Rel(dir, f.absPath)` — where `dir` is the single
directory this `collectFilesToProcess` call is walking (not a global base; the
function runs once per search root) — and drop the file if
`matcher.MatchesPath(rel)`. The matcher is loaded from that same `dir`, so
rel-path and ignore rules share one root.

### Ceiling (recorded as a `ponytail:` comment)

Root-level only: nested per-directory `.gitignore` files are NOT honored.
Upgrade path: collect the applicable ignore stack per directory during the walk.

### UI

Add a 5th checkbox "Respect .gitignore" to `SearchOptions.vue`, threaded through
`SearchForm.handleSearchOptionsUpdate` and the `useSearch` reactive `data` — the
exact pattern of the existing four toggles.

---

## Tests

### Go

`replace_test.go`:
- Dry-run (`Apply:false`) writes nothing — file content and modtime unchanged.
- Apply changes only matched lines; unmatched lines byte-identical.
- Atomic write preserves file mode.
- Regex mode → error, no write.
- Case-sensitive vs insensitive replace differ correctly.
- Path traversal in a matched file path → rejected.
- File with a match but a no-op replacement → untouched.
- Multi-file replace reports correct `FilesChanged` / `LinesChanged`.

`collection_index_test.go`:
- Cold miss populates the cache.
- Warm hit returns identical `[]fileMeta` to the uncached path (equality of the
  filtered set).
- Editing a file changes the fingerprint → re-walk (result reflects the edit).
- Eviction past `maxCollectionEntries`.
- Per-request filters (extension/exclude) applied correctly to a cache hit.

`file_collection_test.go` (extend): gitignore off = current behavior; on drops
ignored paths; negation `!` re-includes; no ignore files = no-op.

### Frontend (vitest)

- `useReplace.spec.ts`: preview populates from binding; apply calls binding with
  `apply:true`, then re-runs search; both no-op under regex mode.
- `SearchOptions.spec.ts`: gitignore toggle emits update.
- `SearchResults.spec.ts`: replace controls render; Apply disabled with no
  preview; whole row hidden under regex mode.

### E2E (Playwright)

One flow: search → type replacement → Preview Replace (diffs shown) → Apply →
results reflect changed content. `ReplaceInFiles` mocked in
`frontend/src/mocks/wailsMock.ts`.

---

## Files

New: `replace.go`, `collection_index.go`, `replace_test.go`,
`collection_index_test.go`, `frontend/src/composables/useReplace.ts`,
`frontend/tests/unit/composables/useReplace.spec.ts`.

Modified: `models.go`, `file_collection.go`, `search_engine.go` (cache hook),
`go.mod` / `go.sum`, `frontend/src/types/search.ts`,
`frontend/src/components/ui/SearchOptions.vue`,
`frontend/src/components/ui/SearchResults.vue`,
`frontend/src/components/ui/SearchForm.vue`,
`frontend/src/composables/useSearch.ts` (thread `respectGitignore`),
`frontend/src/mocks/wailsMock.ts`, regenerated `frontend/wailsjs/**`,
`docs/FEATURES.md`, `README.md`. Existing frontend spec files updated where the
new props/controls touch them.
