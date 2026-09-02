package main

import (
	"container/list"
	"context"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/nxadm/tail"
	"github.com/sirupsen/logrus"
)

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

// SearchState holds the atomic counters for the search process
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

// ---------------------------------------------------------------------------
// Editors
// ---------------------------------------------------------------------------

// EditorAvailability holds information about which editors are available on the system
type EditorAvailability struct {
	VSCode          bool `json:"vscode"`
	VSCodium        bool `json:"vscodium"`
	Sublime         bool `json:"sublime"`
	JetBrains       bool `json:"jetbrains"`
	Geany           bool `json:"geany"`
	Neovim          bool `json:"neovim"`
	Vim             bool `json:"vim"`
	GoLand          bool `json:"goland"`
	PyCharm         bool `json:"pycharm"`
	IntelliJ        bool `json:"intellij"`
	WebStorm        bool `json:"webstorm"`
	PhpStorm        bool `json:"phpstorm"`
	CLion           bool `json:"clion"`
	Rider           bool `json:"rider"`
	AndroidStudio   bool `json:"androidstudio"`
	SystemDefault   bool `json:"systemdefault"`
	Emacs           bool `json:"emacs"`
	Neovide         bool `json:"neovide"`
	CodeBlocks      bool `json:"codeblocks"`
	DevCpp          bool `json:"devcpp"`
	NotepadPlusPlus bool `json:"notepadplusplus"`
	VisualStudio    bool `json:"visualstudio"`
	Eclipse         bool `json:"eclipse"`
	NetBeans        bool `json:"netbeans"`
}

// ---------------------------------------------------------------------------
// Symbol search
// ---------------------------------------------------------------------------

// SymbolInfo represents a symbol found in a source file.
type SymbolInfo struct {
	Name      string `json:"name"`      // Name of the symbol
	Type      string `json:"type"`      // Type: function, class, variable, const, etc.
	Line      int    `json:"line"`      // Line number where symbol is defined (1-indexed)
	File      string `json:"file"`      // File path containing the symbol
	Signature string `json:"signature"` // Full signature/declaration line
}

// SymbolProgressFunc receives incremental scan progress: how many source files
// have been processed, the total to process, and the file currently scanned.
type SymbolProgressFunc func(processed, total int, currentFile string)

// patternConfig holds regex patterns for symbol extraction per language
type patternConfig struct {
	regex     *regexp.Regexp
	nameIndex int
	keyword   string
}

// ---------------------------------------------------------------------------
// Find & replace
// ---------------------------------------------------------------------------

// FileReplacement describes one line change in one file produced by a
// ReplaceInFiles dry-run or apply.
type FileReplacement struct {
	FilePath string `json:"filePath"`
	LineNum  int    `json:"lineNum"`
	OldLine  string `json:"oldLine"`
	NewLine  string `json:"newLine"`
}

// ReplaceRequest carries the search to reuse (its matches define what gets
// replaced) plus the literal replacement string. Apply=false computes a dry-run
// and writes nothing; Apply=true writes the changes atomically.
type ReplaceRequest struct {
	Search      SearchRequest `json:"search"`
	Replacement string        `json:"replacement"`
	Apply       bool          `json:"apply"`
}

// ReplaceResult is the outcome of a dry-run or apply. FilesChanged/LinesChanged
// count only lines whose replacement actually differs from the original.
type ReplaceResult struct {
	Files        []FileReplacement `json:"files"`
	FilesChanged int               `json:"filesChanged"`
	LinesChanged int               `json:"linesChanged"`
}

// ---------------------------------------------------------------------------
// Log streaming
// ---------------------------------------------------------------------------

// LogMessage represents a message sent through the polling system
type LogMessage struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// PollingLogManager manages log entries for the Wails GetInitialLogs and
// GetNewLogs bindings. It tails the log file and maintains a bounded
// in-memory buffer. No HTTP server is involved — the frontend consumes
// entries via IPC (Wails bindings), not HTTP polling.
type PollingLogManager struct {
	logEntries []LogMessage
	mutex      sync.RWMutex
	tail       *tail.Tail
	lastRead   int           // Index to track where we last read up to
	baseIndex  int           // Base index to handle array rotation
	done       chan struct{} // Closed by Shutdown to signal TailFile's wait-loop to exit
	doneOnce   sync.Once     // Guards close(done) against double-close panic
}

// ---------------------------------------------------------------------------
// App and search-engine internals
// ---------------------------------------------------------------------------

// App struct holds the application context and provides methods for the frontend to call.
type App struct {
	ctx                 context.Context
	logger              *logrus.Logger
	searchMu            sync.Mutex          // Guards access to searchCancel
	searchCancel        *searchCancelHandle // Cancel handle for active searches
	editorsMu           sync.RWMutex        // Guards access to availableEditors
	availableEditors    EditorAvailability  // Cache of available editors detected at startup
	ready               int32               // Set to 1 once startup() has run; read via IsAppReady
	editorDetectionDone int32               // Set to 1 once detectAvailableEditors completes; read via GetEditorDetectionStatus
	patternCache        *LRUPatternCache    // LRU cache for compiled regex patterns
	symbolIndex         *symbolIndexCache   // Cached symbol indices per directory
	collectionIndex     *collectionCache    // Cached file-collection results per directory+fingerprint+filter
}

// searchCancelHandle wraps a cancel function so the stored cancel can be
// compared by pointer identity. context.CancelFunc values (closures) are only
// comparable to nil, so a pointer wrapper is the way to tell "this search's
// cancel" from a later search's.
type searchCancelHandle struct {
	cancel context.CancelFunc
}

// lruEntry pairs a cache key with its compiled regex so eviction can remove
// the backing map entry in O(1) instead of scanning the whole map for the key
// that maps to the back element of the LRU list.
type lruEntry struct {
	key   string
	value *regexp.Regexp
}

// LRUPatternCache is a thread-safe LRU cache for compiled regex patterns
type LRUPatternCache struct {
	mu      sync.RWMutex
	cache   map[string]*list.Element
	list    *list.List
	maxSize int64
}

// collectStats holds the counters gathered during the directory walk for
// logging at the end of collection. It's returned by walkDirectoryTree so
// the caller can log a single summary line without passing the App's logger
// deep into the walk.
type collectStats struct {
	filesCollected int
	filesSkipped   int
	dirsSkipped    int
}

// fileMeta carries the per-file metadata gathered during collection so the
// worker pool can process a file without repeating syscalls. The absolute path
// and size are computed once in collectFilesToProcess (file_collection.go);
// reusing them avoids a second os.Stat and filepath.Abs per file.
type fileMeta struct {
	absPath string
	size    int64
}
