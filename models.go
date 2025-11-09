package main

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
	UseRegex         *bool    `json:"useRegex"`         // Whether to treat query as regex (default true for backward compatibility)
	ExcludePatterns  []string `json:"excludePatterns"`  // Patterns to exclude from search (e.g., node_modules, *.log)
	AllowedFileTypes []string `json:"allowedFileTypes"` // List of file extensions that are allowed to be searched (if empty, all types allowed)
}

// ProgressCallback is a function type for reporting search progress
type ProgressCallback func(current int, total int, bufferPath string)

// EditorAvailability holds information about which editors are available on the system
type EditorAvailability struct {
	VSCode        bool `json:"vscode"`
	VSCodium      bool `json:"vscodium"`
	Sublime       bool `json:"sublime"`
	Atom          bool `json:"atom"`
	JetBrains     bool `json:"jetbrains"`
	Geany         bool `json:"geany"`
	Neovim        bool `json:"neovim"`
	Vim           bool `json:"vim"`
	GoLand        bool `json:"goland"`
	PyCharm       bool `json:"pycharm"`
	IntelliJ      bool `json:"intellij"`
	WebStorm      bool `json:"webstorm"`
	PhpStorm      bool `json:"phpstorm"`
	CLion         bool `json:"clion"`
	Rider         bool `json:"rider"`
	AndroidStudio bool `json:"androidstudio"`
	SystemDefault bool `json:"systemdefault"`
	Emacs         bool `json:"emacs"`
	Neovide       bool `json:"neovide"`
	CodeBlocks    bool `json:"codeblocks"`
	DevCpp        bool `json:"devcpp"`
	NotepadPlusPlus bool `json:"notepadplusplus"`
	VisualStudio  bool `json:"visualstudio"`
	Eclipse       bool `json:"eclipse"`
	NetBeans      bool `json:"netbeans"`
}

// SearchProgress represents the progress of a search operation
type SearchProgress struct {
	ProcessedFiles int    `json:"processedFiles"`
	TotalFiles     int    `json:"totalFiles"`
	CurrentFile    string `json:"currentFile"`
	ResultsCount   int    `json:"resultsCount"`
}

// SearchState holds the atomic counters for the search process
type SearchState struct {
	processedFiles int32
	resultsCount   int32
}