package main

import (
	"sync"
	"sync/atomic"
)

// ---------------------------------------------------------------------------
// Search domain
// ---------------------------------------------------------------------------

// Search domain
// ---------------------------------------------------------------------------

// SearchResult represents a single match found in a file during a search operation.
// It contains the file path, line number where the match was found, and the content of that line.
type SearchResult struct {
	FilePath      string   `json:"filePath"`      // Full path to the file containing the match
	LineNum       int      `json:"lineNum"`       // Line number where the match was found (1-indexed)
	Content       string   `json:"content"`       // Content of the line containing the match
	MatchedText   string   `json:"matchedText"`   // The specific text that matched the query
	ContextBefore []string `json:"contextBefore"` // Lines before the match for context
	ContextAfter  []string `json:"contextAfter"`  // Lines after the match for context
}

// SearchRequest contains all parameters needed for a search operation.
// It defines what to search for and where to search.
type SearchRequest struct {
	Directory        string   `json:"directory"`        // Path to the directory to search in
	Query            string   `json:"query"`            // Text to search for
	Extension        string   `json:"extension"`        // File extension to filter by (empty means all extensions)
	CaseSensitive    bool     `json:"caseSensitive"`    // Whether the search should be case sensitive
	IncludeBinary    bool     `json:"includeBinary"`    // Whether to include binary files in search
	MaxFileSize      int64    `json:"maxFileSize"`      // Maximum file size in bytes (default 10MB if 0)
	MinFileSize      int64    `json:"minFileSize"`      // Minimum file size in bytes (default 0 if not specified)
	MaxResults       int      `json:"maxResults"`       // Maximum number of results to return (default 1000 if 0)
	UseRegex         bool     `json:"useRegex"`         // Whether to treat query as regex. Plain bool (was *bool): Go's zero value is false, so callers that omit the field get literal search — matching what the frontend always sends. The *bool/nil-default-true form was dropped because a nil pointer serializes to "useRegex": null, which is not assignable to the frontend's boolean type and broke the IPC contract.
	ExcludePatterns  []string `json:"excludePatterns"`  // Patterns to exclude from search (e.g., node_modules, *.log)
	AllowedFileTypes []string `json:"allowedFileTypes"` // List of file extensions that are allowed to be searched (if empty, all types allowed)
	ContextLines     int      `json:"contextLines"`     // Number of context lines before/after match (default 2)
	Directories      []string `json:"directories"`      // Additional directories to search (merged with Directory)
	FuzzySearch      bool     `json:"fuzzySearch"`      // When true (and UseRegex is false), the engine appends a second phase of fuzzy near-miss candidates after exact matches complete. Candidates are lines that do not match the exact pattern but contain a sliding-window alignment with the query (>=60% positional character matches). The frontend re-scores these and flags them with a fuzzy badge; enabling fuzzy never changes exact-match results. See search_fuzzy.go for full semantics.
	RespectGitignore bool     `json:"respectGitignore"` // When true, files ignored by the directory's root .gitignore and .git/info/exclude are excluded from collection. Default false — behavior is byte-identical to pre-feature when unset.
}

// SearchProgress represents the progress of a search operation
type SearchProgress struct {
	ProcessedFiles int    `json:"processedFiles"`
	TotalFiles     int    `json:"totalFiles"`
	CurrentFile    string `json:"currentFile"`
	ResultsCount   int    `json:"resultsCount"`
	FailedFiles    int    `json:"failedFiles"`
	Status         string `json:"status"`
	// FailedPaths lists the files that could not be read, capped at
	// maxFailedPathsReported. Only the terminal ("completed") event carries
	// it: attaching a growing array to every throttled in-progress event
	// would re-serialize the same paths dozens of times per search.
	FailedPaths []string `json:"failedPaths"`
}

// SearchState holds the atomic counters for the search process.
//
// Thread-safety contract: every field is either an atomic (processedFiles,
// resultsCount, failedFiles, lastProgressNano — touch only via sync/atomic)
// or guarded by failedMu (failedPaths). SearchState is always shared by
// pointer across the worker-pool goroutines and the drain goroutine, so any
// plain read/write of these fields is a data race. snapshotFailedPaths
// returns a copy so event payloads never share the guarded slice.
type SearchState struct {
	processedFiles   int32
	resultsCount     int32
	failedFiles      int32
	lastProgressNano int64 // Last progress-event emit time (UnixNano) for throttling

	// failedMu guards failedPaths. failedFiles is the unbounded count; this
	// slice is the bounded sample the UI can actually list.
	failedMu    sync.Mutex
	failedPaths []string
}

// recordFailure increments the failed-file counter and, while under
// maxFailedPathsReported, remembers the path so the UI can name what it
// skipped instead of showing a bare count.
func (s *SearchState) recordFailure(absPath string) {
	atomic.AddInt32(&s.failedFiles, 1)
	if absPath == "" {
		return
	}
	s.failedMu.Lock()
	if len(s.failedPaths) < maxFailedPathsReported {
		s.failedPaths = append(s.failedPaths, absPath)
	}
	s.failedMu.Unlock()
}

// snapshotFailedPaths returns a copy of the recorded failure sample so the
// caller can put it on an event payload without sharing the guarded slice.
func (s *SearchState) snapshotFailedPaths() []string {
	s.failedMu.Lock()
	defer s.failedMu.Unlock()
	if len(s.failedPaths) == 0 {
		return nil
	}
	out := make([]string, len(s.failedPaths))
	copy(out, s.failedPaths)
	return out
}

// SearchResultBatch is one incremental slice of results pushed to the frontend
// on the "search-results" event while a search is still running. Seq is a
// monotonic per-search counter starting at 1, so the frontend can drop a
// replayed or out-of-order batch instead of duplicating rows.
//
// Batches are additive: the frontend appends them in arrival order and does
// not re-sort. SearchWithProgress still returns the full sorted slice, which
// is the authoritative, deterministically-ordered result set — the batches
// are a progressive-render channel, not a replacement for it.
type SearchResultBatch struct {
	Seq     int            `json:"seq"`
	Results []SearchResult `json:"results"`
}

// ReplaceProgress reports the progress of a ReplaceInFiles run on the
// "replace-progress" event. A replace can take as long as a search but had no
// feedback channel at all, so a large one looked frozen.
//
// Phase distinguishes the two halves of the operation, which have very
// different stakes: "staging" writes nothing and is safely abortable, while a
// cancel during "writing" leaves already-written files on disk (there is no
// rollback by design — the user's VCS is the undo path).
type ReplaceProgress struct {
	Phase          string `json:"phase"` // staging | writing | cancelled | complete
	ProcessedFiles int    `json:"processedFiles"`
	TotalFiles     int    `json:"totalFiles"`
	CurrentFile    string `json:"currentFile"`
	FilesChanged   int    `json:"filesChanged"`
	LinesChanged   int    `json:"linesChanged"`
}
