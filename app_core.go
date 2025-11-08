package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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
type ProgressCallback func(current int, total int, filePath string)

// App struct holds the application context and provides methods for the frontend to call.
type App struct {
	ctx          context.Context
	searchCancel context.CancelFunc // Cancel function for active searches
}

// NewApp creates a new App application struct.
// This function is called during application initialization.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Emit an app-ready event to notify the frontend that the app is initialized
	// We can safely emit this event since we're in the startup context
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "app-ready", map[string]interface{}{
			"status":    "ready",
			"timestamp": time.Now().Unix(),
		})
	}
}

// isBinary checks if content appears to be binary by looking for null bytes
// and a high proportion of non-text characters
func (a *App) isBinary(content []byte) bool {
	// Check for null bytes which usually indicate binary content
	if len(content) > 0 && strings.Contains(string(content[:min(512, len(content))]), "\x00") {
		return true
	}

	// Count printable vs non-printable characters in first part of file
	// For UTF-8 text, we need to be more lenient as many Unicode characters have high bytes
	printableCount := 0
	for i, b := range content {
		if i >= 512 { // Only check first 512 bytes for performance
			break
		}
		// Printable ASCII range (space through ~) and common whitespace
		// Also consider high-byte values as potentially printable for UTF-8
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' || (b >= 127 && b <= 255) {
			printableCount++
		}
	}

	// If less than 70% of characters are printable, consider it binary
	// For UTF-8 content, we'll be more lenient
	if len(content) > 0 {
		return float64(printableCount)/float64(min(512, len(content))) < 0.5
	}
	return false
}

// GetDirectoryContents returns a list of all directory paths in the specified path.
// This function recursively walks the directory tree and collects all directories.
func (a *App) GetDirectoryContents(path string) ([]string, error) {
	var items []string

	// Walk the directory tree and collect all directories
	err := filepath.WalkDir(path, func(itemPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip unreadable items and continue
		}
		if d.IsDir() {
			items = append(items, itemPath) // Only add directories, not files
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return items, nil
}

// ValidateDirectory checks if a directory exists and is accessible for reading.
// This function is useful for validating user-provided directory paths before performing operations.
func (a *App) ValidateDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("directory does not exist: %s", path)
		}
		return false, err
	}

	if !info.IsDir() {
		return false, fmt.Errorf("path is not a directory: %s", path)
	}

	// Try to read the directory to ensure it's accessible
	_, err = os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("directory is not accessible: %s", path)
	}

	return true, nil
}

// processFileLineByLine processes a file line by line to avoid loading large files into memory
func (a *App) processFileLineByLine(ctx context.Context, filePath string, pattern *regexp.Regexp, maxResults int, includeBinary bool) ([]SearchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// If not including binary files, check if this file is binary and skip if it is
	// Read only the first portion of the file for binary detection
	if !includeBinary {
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err == nil && n > 0 && a.isBinary(buffer[:n]) {
			return []SearchResult{}, nil // Return empty results for binary files
		}
		// Reset file pointer back to beginning for processing
		file.Seek(0, 0)
	}

	var results []SearchResult
	scanner := bufio.NewScanner(file)

	// Set a larger buffer for very long lines (1MB)
	buf := make([]byte, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 1024*1024)

	lineNum := 1
	linesProcessed := 0
	for scanner.Scan() && len(results) < maxResults {
		line := scanner.Text()
		if pattern.MatchString(line) {
			result := SearchResult{
				FilePath:      filePath,
				LineNum:       lineNum,
				Content:       strings.TrimSpace(line),
				MatchedText:   "",         // Will be set later with actual matched text
				ContextBefore: []string{}, // Context lines are not collected in streaming mode
				ContextAfter:  []string{},
			}
			// Set the matched text from the actual match
			matches := pattern.FindString(line)
			if matches != "" {
				result.MatchedText = matches
			}
			results = append(results, result)
		}

		lineNum++
		linesProcessed++

		// Check for context cancellation every 100 lines to avoid performance impact
		if linesProcessed%100 == 0 {
			select {
			case <-ctx.Done(): // Use the specific search context to check for cancellation
				// Context was cancelled externally
				return results, nil
			default:
				// Continue processing
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// SearchProgress represents the progress of a search operation
type SearchProgress struct {
	ProcessedFiles int    `json:"processedFiles"`
	TotalFiles     int    `json:"totalFiles"`
	CurrentFile    string `json:"currentFile"`
	ResultsCount   int    `json:"resultsCount"`
}

// SearchWithProgress performs a search and emits progress updates to the frontend
func (a *App) SearchWithProgress(req SearchRequest) ([]SearchResult, error) {
	// Validate and set defaults for parameters
	validatedReq, err := a.validateAndSetDefaults(req)
	if err != nil {
		return nil, err
	}
	req = validatedReq

	// If query is empty, return empty results instead of error to maintain compatibility
	if req.Query == "" {
		return []SearchResult{}, nil
	}

	// Prepare search pattern based on case sensitivity and regex requirements
	pattern, err := a.compileSearchPattern(req)
	if err != nil {
		return nil, err
	}

	// Get the base directory for path traversal check
	absDir, err := filepath.Abs(req.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for directory: %v", err)
	}
	baseDir := filepath.Clean(absDir) + string(filepath.Separator)

	// Collect all files to process based on search criteria
	filesToProcess, err := a.collectFilesToProcess(req, pattern, baseDir)
	if err != nil {
		return nil, err
	}

	totalFiles := len(filesToProcess)

	// Emit initial progress
	a.safeEmitEvent("search-progress", map[string]interface{}{
		"processedFiles": 0,
		"totalFiles":     totalFiles,
		"currentFile":    "",
		"resultsCount":   0,
		"status":         "started",
	})

	// Create search context with cancellation
	ctx, cancel := a.createSearchContext()
	defer func() {
		// Clear the cancel function when the search completes
		a.searchCancel = nil
		cancel()
	}()

	// Process files using worker pool
	resultsChan, searchState := a.processFilesWithWorkers(ctx, filesToProcess, req, pattern, baseDir, totalFiles)

	// Collect results
	var results []SearchResult
	for result := range resultsChan {
		results = append(results, result)

		// Check if we've reached the result limit
		if len(results) >= req.MaxResults {
			// The context is already cancelled by the workers, but we'll do it again just in case
			if a.searchCancel != nil {
				a.searchCancel()
			}
			// Trim results to max results if somehow we got more
			if len(results) > req.MaxResults {
				results = results[:req.MaxResults]
			}
			break
		}
	}

	// Emit final progress
	a.safeEmitEvent("search-progress", map[string]interface{}{
		"processedFiles": int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":     totalFiles,
		"currentFile":    "",
		"resultsCount":   len(results),
		"status":         "completed",
	})

	return results, nil
}

// matchesPattern checks if a path matches an exclude pattern
func (a *App) matchesPattern(path string, pattern string) bool {
	// First try exact match
	if path == pattern {
		return true
	}

	// Try filepath.Match for glob patterns
	matched, err := filepath.Match(pattern, filepath.Base(path))
	if err != nil {
		// If pattern is invalid, don't match
		return false
	}
	if matched {
		return true
	}

	// Check if path contains the pattern (for directory patterns like "node_modules")
	basePath := filepath.Base(path)
	dirPath := filepath.Dir(path)

	// Check if pattern matches directory components
	if strings.Contains(dirPath, pattern) || strings.Contains(basePath, pattern) {
		return true
	}

	return false
}

// Helper function to get number of CPUs
func numCPU() int {
	n := runtime.NumCPU()
	if n < 2 {
		return 2 // Use at least 2 workers for parallelism
	}
	return n
}

// safeEmitEvent safely emits a Wails event, ignoring errors when not in proper context
func (a *App) safeEmitEvent(eventName string, data interface{}) {
	// If context is nil, we can't emit events
	if a.ctx == nil {
		return
	}

	// Simple check to see if we're in a proper Wails context
	// We can only emit events when we're in a proper Wails context
	// In test environments or when not in a Wails context, ctx.Done() will panic
	// So we'll just return without emitting if we can't safely check the context
	defer func() {
		if r := recover(); r != nil {
			// We're not in a proper Wails context, don't emit events
			return
		}
	}()

	// Check if the context is still valid
	select {
	case <-a.ctx.Done():
		// Context is cancelled, don't emit
		return
	default:
		// Context is active, but we still need to be cautious with EventsEmit
		// Try to emit the event but catch any panics from EventsEmit
		func() {
			defer func() {
				if r := recover(); r != nil {
					// EventsEmit panicked, which means we're not in a proper Wails context
					return
				}
			}()

			wailsRuntime.EventsEmit(a.ctx, eventName, data)
		}()
	}
}

// ReadFile reads the content of a file and returns it as a string.
// This function is used by the frontend to read file contents for display in the modal.
func (a *App) ReadFile(filePath string) (string, error) {
	// Validate input
	if filePath == "" {
		return "", fmt.Errorf("file path is required")
	}

	// Sanitize the input path to prevent directory traversal attacks
	cleanPath := filepath.Clean(filePath)

	// Validate that the path does not contain traversal sequences
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid file path: contains directory traversal")
	}

	// Check if file exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", cleanPath)
	}

	// Read file content with size limit to prevent memory issues
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %v", err)
	}

	// Limit file size to prevent memory issues (e.g., 50MB)
	maxReadSize := int64(50 * 1024 * 1024) // 50MB
	if fileInfo.Size() > maxReadSize {
		return "", fmt.Errorf("file too large to read: %s (size: %d, max: %d)", cleanPath, fileInfo.Size(), maxReadSize)
	}

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return string(content), nil
}

// CancelSearch cancels any active search operation by calling the cancel function
func (a *App) CancelSearch() error {
	if a.searchCancel != nil {
		a.searchCancel()
		// Emit cancellation progress event
		a.safeEmitEvent("search-progress", map[string]interface{}{
			"processedFiles": 0,
			"totalFiles":     0,
			"currentFile":    "",
			"resultsCount":   0,
			"status":         "cancelled",
		})
		return nil
	}
	// If there's no active search to cancel, return an appropriate message
	return fmt.Errorf("no active search to cancel")
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getFullExtension extracts the full extension from a file path
// For example: "file.min.js" returns ".min.js", "archive.tar.gz" returns ".tar.gz"
func getFullExtension(path string) string {
	base := filepath.Base(path)

	// If there's no dot, return empty string
	if !strings.Contains(base, ".") {
		return ""
	}

	// Find the first dot and return everything after it
	firstDotIndex := strings.Index(base, ".")
	if firstDotIndex == -1 {
		return ""
	}

	return base[firstDotIndex:]
}

// matchExtension checks if a file path matches an extension requirement
// This handles both single extensions (like "js") and full extensions (like "min.js", "tar.gz")
func matchExtension(path string, requestedExt string) bool {
	if requestedExt == "" {
		return true
	}

	// First try to match the final extension (current behavior for backward compatibility)
	finalExt := strings.TrimPrefix(filepath.Ext(path), ".")
	if strings.EqualFold(finalExt, requestedExt) {
		return true
	}

	// Then try to match the full extension sequence
	fullExt := strings.TrimPrefix(getFullExtension(path), ".")
	if strings.EqualFold(fullExt, requestedExt) {
		return true
	}

	return false
}

// validateAndSetDefaults validates the search request and sets default values
func (a *App) validateAndSetDefaults(req SearchRequest) (SearchRequest, error) {
	// Set default values for optional parameters
	modifiedReq := req
	if modifiedReq.MaxFileSize == 0 {
		modifiedReq.MaxFileSize = 10 * 1024 * 1024 // 10MB default
	}
	if modifiedReq.MaxResults == 0 {
		modifiedReq.MaxResults = 1000 // 1000 results default
	}

	// Validate that directory path doesn't contain traversal sequences before resolution
	cleanPath := filepath.Clean(modifiedReq.Directory)
	pathParts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range pathParts {
		if part == ".." {
			return req, fmt.Errorf("invalid directory path: contains traversal sequences")
		}
	}

	// Validate directory exists before starting the search
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return req, fmt.Errorf("directory does not exist: %s", cleanPath)
	}

	// Get absolute path for internal processing
	absDir, err := filepath.Abs(modifiedReq.Directory)
	if err != nil {
		return req, fmt.Errorf("failed to get absolute path for directory: %v", err)
	}

	// Additional check: prevent searching system-critical directories
	// This helps prevent system hangs when traversal resolves to high-level directories
	// Only block exact matches of critical system directories (not parent directories like /tmp)
	var protectedPaths []string
	if runtime.GOOS == "windows" {
		protectedPaths = []string{
			"C:\\", "C:\\Windows", "C:\\Windows\\System32", "C:\\Windows\\System", 
			"C:\\Program Files", "C:\\Program Files (x86)", "C:\\Users", "C:\\Documents and Settings",
		}
	} else {
		protectedPaths = []string{"/", "/usr", "/bin", "/sbin", "/lib", "/lib64", "/proc", "/sys", "/dev", "/etc"}
	}
	cleanBaseDir := filepath.Clean(absDir)
	for _, protected := range protectedPaths {
		if cleanBaseDir == protected {
			return req, fmt.Errorf("searching in protected system directory not allowed: %s", cleanBaseDir)
		}
	}

	return modifiedReq, nil
}

// compileSearchPattern prepares the search pattern based on case sensitivity and regex requirements
func (a *App) compileSearchPattern(req SearchRequest) (*regexp.Regexp, error) {
	var pattern *regexp.Regexp
	var err error

	// Determine if we should use regex mode (default to true for backward compatibility)
	useRegex := true
	if req.UseRegex != nil {
		useRegex = *req.UseRegex
	}

	if useRegex {
		// If using regex, use the query as-is (with case sensitivity flag)
		searchPattern := req.Query
		if !req.CaseSensitive {
			// Use the (?i) flag for case insensitive matching
			searchPattern = "(?i)" + req.Query
		}
		pattern, err = regexp.Compile(searchPattern)
	} else {
		// For literal search, escape special regex characters
		escapedQuery := regexp.QuoteMeta(req.Query)
		if req.CaseSensitive {
			// For case sensitive literal search
			pattern, err = regexp.Compile(escapedQuery)
		} else {
			// For case insensitive literal search
			pattern, err = regexp.Compile("(?i)" + escapedQuery)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("invalid search pattern: %v", err)
	}

	return pattern, nil
}

// collectFilesToProcess walks the directory tree and collects all files to process based on search criteria
func (a *App) collectFilesToProcess(req SearchRequest, pattern *regexp.Regexp, baseDir string) ([]string, error) {
	var filesToProcess []string

	err := filepath.WalkDir(req.Directory, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// If there's an error accessing a file/directory, skip it and continue
			return nil
		}

		// Check for path traversal during walk
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil // Skip if we can't get absolute path
		}
		relPath, err := filepath.Rel(baseDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
			// This path is outside the base directory - skip it
			if d.IsDir() {
				return filepath.SkipDir // Skip the entire subdirectory
			}
			return nil
		}

		if d.IsDir() {
			// Skip hidden directories that start with a dot (e.g., .git, .vscode)
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply file extension filter if specified
		if req.Extension != "" {
			if !matchExtension(path, req.Extension) {
				return nil
			}
		}

		// If allow list is specified, check if the file type is allowed
		if len(req.AllowedFileTypes) > 0 {
			isAllowed := false
			for _, allowedExt := range req.AllowedFileTypes {
				if matchExtension(path, allowedExt) {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				return nil
			}
		}

		// Get file information to check size before reading
		fileInfo, err := d.Info()
		if err != nil {
			return nil // Skip if we can't get file info
		}

		// Skip very large files to prevent memory issues
		if fileInfo.Size() > req.MaxFileSize {
			return nil
		}

		// Skip very small files based on min file size
		if fileInfo.Size() < req.MinFileSize {
			return nil
		}

		// Check exclude patterns
		for _, patternStr := range req.ExcludePatterns {
			if patternStr != "" && a.matchesPattern(path, patternStr) {
				return nil
			}
		}

		// If not including binary files, check if this file is binary and skip if it is
		// Read only the first portion of the file for binary detection to avoid memory issues
		if !req.IncludeBinary {
			file, err := os.Open(path)

			if err == nil {
				defer file.Close()
				// Read only the first 512 bytes to check for binary content
				buffer := make([]byte, 512)
				n, _ := file.Read(buffer)
				if n > 0 && a.isBinary(buffer[:n]) {
					return nil // Skip binary files
				}
			}
		}

		filesToProcess = append(filesToProcess, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return filesToProcess, nil
}

// createSearchContext creates a context for the search operation with associated cancellation
func (a *App) createSearchContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	// Store the cancel function so it can be called externally to cancel the search
	a.searchCancel = cancel
	return ctx, cancel
}

// SearchState holds the atomic counters for the search process
type SearchState struct {
	processedFiles int32
	resultsCount   int32
}

// processFilesWithWorkers processes files using a worker pool and returns a channel of results
func (a *App) processFilesWithWorkers(ctx context.Context, filesToProcess []string, req SearchRequest, pattern *regexp.Regexp, baseDir string, totalFiles int) (chan SearchResult, *SearchState) {
	// Use a worker pool to process files in parallel
	numWorkers := numCPU()
	if len(filesToProcess) < numWorkers {
		numWorkers = len(filesToProcess)
	}

	// Create channels
	filesChan := make(chan string, len(filesToProcess))
	resultsChan := make(chan SearchResult, 100)

	// Track progress
	searchState := &SearchState{
		processedFiles: 0,
		resultsCount:   0,
	}

	// Create atomic flag to track if cancellation has been triggered to prevent multiple cancellations
	var searchCancelled int32 = 0

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					// Context cancelled, stop processing and exit
					return
				case filePath, ok := <-filesChan:
					if !ok {
						// Channel closed, exit worker
						return
					}
					// Check if we've already reached the max results
					if int(atomic.LoadInt32(&searchState.resultsCount)) >= req.MaxResults {
						// Only cancel if not already cancelled to prevent race conditions
						if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
							// The context is already stored in a.searchCancel, so we use that
							if a.searchCancel != nil {
								a.searchCancel()
							}
						}
						return
					}

					// Check if context has been cancelled before processing each file
					select {
					case <-ctx.Done():
						// Context cancelled, stop processing
						return
					default:
						// Context is still active, continue processing
					}

					// Get file info to determine if it's a large file that should be processed in streaming mode
					fileInfo, statErr := os.Stat(filePath)
					if statErr != nil {
						continue // Skip if we can't get file info
					}

					// For larger files, use streaming line-by-line processing to avoid memory issues
					// Threshold is set to 1MB (can be adjusted as needed)
					const streamingThreshold = 1024 * 1024 // 1MB
					var fileResults []SearchResult

					// Additional path traversal check for the current file path
					absFilePath, absErr := filepath.Abs(filePath)
					if absErr != nil {
						continue // Skip if we can't get absolute path
					}
					relFilePath, relErr := filepath.Rel(baseDir, absFilePath)
					if relErr != nil || strings.HasPrefix(relFilePath, "..") {
						continue // Skip if file is outside the base directory
					}

					if fileInfo.Size() > streamingThreshold {
						// Use streaming approach for large files
						streamResults, procErr := a.processFileLineByLine(ctx, absFilePath, pattern, req.MaxResults-int(atomic.LoadInt32(&searchState.resultsCount)), req.IncludeBinary)
						if procErr != nil {
							continue // Skip problematic files
						}
						fileResults = streamResults
					} else {
						// Use original approach for smaller files (which is generally faster for small files)
						content, readErr := os.ReadFile(absFilePath)
						if readErr != nil {
							// Skip unreadable files (permissions, etc.)
							continue
						}

						// Check if file is binary if we're not including binary files
						if !req.IncludeBinary && a.isBinary(content) {
							continue
						}

						// Split content into lines for line-by-line searching
						lines := strings.Split(string(content), "\n")
						for i, line := range lines {
							// Check again if we've reached max results before processing more
							if int(atomic.LoadInt32(&searchState.resultsCount)) >= req.MaxResults {
								// Only cancel if not already cancelled to prevent race conditions
								if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
									if a.searchCancel != nil {
										a.searchCancel()
									}
								}
								return
							}

							// Check if context has been cancelled during line processing
							if i%100 == 0 { // Check every 100 lines to avoid performance impact
								select {
								case <-ctx.Done():
									// Context cancelled, stop processing
									return
								default:
									// Context is still active, continue processing
								}
							}

							if pattern.MatchString(line) {
								// Calculate context lines (2 before, 2 after)
								contextBefore := []string{}
								contextAfter := []string{}

								// Get up to 2 lines before the match
								for j := i - 2; j < i; j++ {
									if j >= 0 {
										contextBefore = append(contextBefore, lines[j])
									}
								}

								// Get up to 2 lines after the match
								for j := i + 1; j <= i+2 && j < len(lines); j++ {
									contextAfter = append(contextAfter, lines[j])
								}

								// Found a match, send to results channel
								result := SearchResult{
									FilePath:      absFilePath,             // Use absolute cleaned path
									LineNum:       i + 1,                   // Convert to 1-indexed line numbers
									Content:       strings.TrimSpace(line), // Remove leading/trailing whitespace
									MatchedText:   req.Query,               // Store the original query as matched text
									ContextBefore: contextBefore,
									ContextAfter:  contextAfter,
								}

								fileResults = append(fileResults, result)
							}
						}
					}

					// Send all results from this file to the results channel
					for _, result := range fileResults {
						// Check again if max results reached before sending
						if int(atomic.LoadInt32(&searchState.resultsCount)) >= req.MaxResults {
							// Only cancel if not already cancelled to prevent race conditions
							if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
								if a.searchCancel != nil {
									a.searchCancel()
								}
							}
							return
						}

						// Use a non-blocking send with context check
						select {
						case resultsChan <- result:
							// Increment results count atomically
							newResultsCount := atomic.AddInt32(&searchState.resultsCount, 1)

							// Check if we've reached the result limit after incrementing
							if int(newResultsCount) >= req.MaxResults {
								// Only cancel if not already cancelled to prevent race conditions
								if atomic.CompareAndSwapInt32(&searchCancelled, 0, 1) {
									if a.searchCancel != nil {
										a.searchCancel()
									}
								}
							}
						case <-ctx.Done():
							// Context cancelled, stop processing
							return
						}
					}

					// Increment processed files count atomically
					newCount := atomic.AddInt32(&searchState.processedFiles, 1)

					// Emit progress update periodically
					if newCount%10 == 0 || int(newCount) == len(filesToProcess) {
						a.safeEmitEvent("search-progress", map[string]interface{}{
							"processedFiles": int(newCount),
							"totalFiles":     totalFiles,
							"currentFile":    absFilePath,
							"resultsCount":   int(atomic.LoadInt32(&searchState.resultsCount)),
							"status":         "in-progress",
						})
					}
				}
			}
		}()
	}

	// Send all files to the channel
	go func() {
		defer close(filesChan)
		for _, file := range filesToProcess {
			select {
			case <-ctx.Done():
				// Context cancelled, stop sending files
				return
			case filesChan <- file:
				// Continue sending files
			}
		}
	}()

	// Close resultsChan when all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	return resultsChan, searchState
}

// SelectDirectory opens a native directory selection dialog and returns the selected path.
// This function uses the Wails runtime dialog to provide a native directory selection
// experience across all platforms (Windows, Linux, macOS).
func (a *App) SelectDirectory(title string) (string, error) {
	// Validate input parameters
	if title == "" {
		title = "Select Directory" // Use default title if none provided
	}

	// Check if we have a valid context
	if a.ctx == nil {
		return "", fmt.Errorf("no valid context available for dialog - application may not be fully initialized")
	}

	// Prepare dialog options with the provided title
	dialogOptions := wailsRuntime.OpenDialogOptions{
		Title: title,
	}

	// Use Wails runtime OpenDirectoryDialog to show the native dialog
	selectedPath, err := wailsRuntime.OpenDirectoryDialog(a.ctx, dialogOptions)
	if err != nil {
		// Return any error that occurred during the dialog operation
		// This includes system-level errors but excludes user cancellation
		return "", fmt.Errorf("failed to open directory dialog: %w", err)
	}

	// If selectedPath is empty, the user cancelled the dialog
	// Return empty string with no error to indicate cancellation
	return selectedPath, nil
}
