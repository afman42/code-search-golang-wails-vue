package main

import (
	"container/list"
	"context"
	"regexp"
	"sync"

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
	SearchSubdirs    bool     `json:"searchSubdirs"`    // Whether to search subdirectories (default true)
	UseRegex         *bool    `json:"useRegex"`         // Whether to treat query as regex. Pointer type (not plain bool): nil means "default to true" for backward compat with older callers that omit the field. The frontend always sends a concrete boolean, so nil only occurs for programmatic callers. When non-nil, the value (true/false) is used as-is.
	ExcludePatterns  []string `json:"excludePatterns"`  // Patterns to exclude from search (e.g., node_modules, *.log)
	AllowedFileTypes []string `json:"allowedFileTypes"` // List of file extensions that are allowed to be searched (if empty, all types allowed)
	ContextLines     int      `json:"contextLines"`     // Number of context lines before/after match (default 2)
	Directories      []string `json:"directories"`      // Additional directories to search (merged with Directory)
	FuzzySearch      bool     `json:"fuzzySearch"`      // Client-side fuzzy matching flag. The backend does NOT use this — fuzzy filtering happens entirely in the frontend (useSearch.ts post-processing). The field exists to make the IPC contract explicit so the field isn't silently dropped by Go's JSON deserialization (which ignores unknown fields by default).
}

// ProgressCallback is a function type for reporting search progress
type ProgressCallback func(current int, total int, bufferPath string)

// SearchProgress represents the progress of a search operation
type SearchProgress struct {
	ProcessedFiles int    `json:"processedFiles"`
	TotalFiles     int    `json:"totalFiles"`
	CurrentFile    string `json:"currentFile"`
	ResultsCount   int    `json:"resultsCount"`
	Status         string `json:"status"`
}

// SearchState holds the atomic counters for the search process
type SearchState struct {
	processedFiles   int32
	resultsCount     int32
	lastProgressNano int64 // Last progress-event emit time (UnixNano) for throttling
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
	ctx              context.Context
	logger           *logrus.Logger
	searchMu         sync.Mutex         // Guards access to searchCancel
	searchCancel     context.CancelFunc // Cancel function for active searches
	editorsMu        sync.RWMutex       // Guards access to availableEditors
	availableEditors EditorAvailability // Cache of available editors detected at startup
	ready            int32              // Set to 1 once startup() has run; read via IsAppReady
	patternCache     *LRUPatternCache   // LRU cache for compiled regex patterns
	symbolIndex      *symbolIndexCache  // Cached symbol indices per directory
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
