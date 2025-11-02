// Package main implements the backend functionality for the code search application.
// It provides functions for searching through code files, validating directories,
// and interacting with the system's file manager.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// SearchResult represents a single match found in a file during a search operation.
// It contains the file path, line number where the match was found, and the content of that line.
type SearchResult struct {
	FilePath    string `json:"filePath"`    // Full path to the file containing the match
	LineNum     int    `json:"lineNum"`     // Line number where the match was found (1-indexed)
	Content     string `json:"content"`     // Content of the line containing the match
	MatchedText string `json:"matchedText"` // The specific text that matched the query
}

// SearchRequest contains all parameters needed for a search operation.
// It defines what to search for and where to search.
type SearchRequest struct {
	Directory     string `json:"directory"`     // Path to the directory to search in
	Query         string `json:"query"`         // Text to search for
	Extension     string `json:"extension"`     // File extension to filter by (empty means all extensions)
	CaseSensitive bool   `json:"caseSensitive"` // Whether the search should be case sensitive
	IncludeBinary bool   `json:"includeBinary"` // Whether to include binary files in search
	MaxFileSize   int64  `json:"maxFileSize"`   // Maximum file size in bytes (default 10MB if 0)
	MaxResults    int    `json:"maxResults"`    // Maximum number of results to return (default 1000 if 0)
	SearchSubdirs bool   `json:"searchSubdirs"` // Whether to search subdirectories (default true)
}

// App struct holds the application context and provides methods for the frontend to call.
type App struct {
	ctx context.Context
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
}

// SearchCode performs a text search in files within the specified directory.
// It looks for the provided query in all files (or filtered by extension) and returns matches.
// The search respects case sensitivity settings and has customizable parameters like
// file size limits, result count limits, and binary file inclusion.
func (a *App) SearchCode(req SearchRequest) ([]SearchResult, error) {
	var results []SearchResult
	
	// Set default values for optional parameters
	if req.MaxFileSize == 0 {
		req.MaxFileSize = 10 * 1024 * 1024 // 10MB default
	}
	if req.MaxResults == 0 {
		req.MaxResults = 1000 // 1000 results default
	}
	
	// Validate directory exists before starting the search
	if _, err := os.Stat(req.Directory); os.IsNotExist(err) {
		return results, fmt.Errorf("directory does not exist: %s", req.Directory)
	}
	
	// If query is empty, return empty results instead of error to maintain compatibility
	if req.Query == "" {
		return results, nil
	}
	
	// Prepare search pattern based on case sensitivity requirement
	searchPattern := req.Query
	if !req.CaseSensitive {
		// Use the (?i) flag for case insensitive matching
		// But ensure it doesn't interfere with regex characters
		searchPattern = "(?i)" + req.Query
	}
	
	// Compile the regex pattern for efficient matching
	pattern, err := regexp.Compile(searchPattern)
	if err != nil {
		return results, fmt.Errorf("invalid search pattern: %v", err)
	}
	
	// Walk through the directory tree
	err = filepath.WalkDir(req.Directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If there's an error accessing a file/directory, skip it and continue
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
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if ext != req.Extension {
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
		
		// Read file content into memory
		content, err := os.ReadFile(path)
		if err != nil {
			// Skip unreadable files (permissions, etc.)
			return nil
		}
		
		// Check if file is binary if we're not including binary files
		if !req.IncludeBinary && a.isBinary(content) {
			return nil
		}
		
		// Split content into lines for line-by-line searching
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if pattern.MatchString(line) {
				// Found a match, add to results
				results = append(results, SearchResult{
					FilePath: path,
					LineNum:  i + 1, // Convert to 1-indexed line numbers
					Content:  strings.TrimSpace(line), // Remove leading/trailing whitespace
					MatchedText: req.Query, // Store the original query as matched text
				})
				
				// Limit results to prevent excessive memory usage
				if len(results) >= req.MaxResults {
					return filepath.SkipAll // stop walking the directory completely
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		return results, err
	}
	
	return results, nil
}

// isBinary checks if content appears to be binary by looking for null bytes
// and a high proportion of non-text characters
func (a *App) isBinary(content []byte) bool {
	// Check for null bytes which usually indicate binary content
	if len(content) > 0 && strings.Contains(string(content[:min(512, len(content))]), "\x00") {
		return true
	}
	
	// Count printable vs non-printable characters in first part of file
	// If more than 30% are non-printable (excluding common whitespace), it's likely binary
	printableCount := 0
	for i, b := range content {
		if i >= 512 { // Only check first 512 bytes for performance
			break
		}
		// Printable ASCII range (space through ~) and common whitespace
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printableCount++
		}
	}
	
	// If less than 70% of characters are printable, consider it binary
	if len(content) > 0 {
		return float64(printableCount)/float64(min(512, len(content))) < 0.7
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

// SelectDirectory opens a native directory selection dialog and returns the selected path.
// This function implements cross-platform directory selection using system dialogs:
// - On macOS: Uses AppleScript to show a native dialog
// - On Linux: Tries multiple options in order of preference (zenity, kdialog, yad)
// - On Windows: Not fully implemented in this version (requires additional Windows API calls)
func (a *App) SelectDirectory(title string) (string, error) {
    var cmd string
    var args []string

    switch runtime.GOOS {
    case "windows":
        // On Windows, showing a proper directory picker requires Windows API calls
        // For a complete implementation, you would need to use Windows syscalls
        // to access the native folder browser dialog
        return "", fmt.Errorf("directory picker not implemented on Windows in this version - implement using Windows API calls")
    case "darwin": // macOS
        // Use AppleScript to show a native open panel dialog
        script := fmt.Sprintf("osascript -e 'POSIX path of (choose folder with prompt \"%s\")'", title)
        cmd = "bash"
        args = []string{"-c", script}
    case "linux":
        // Try multiple options in order of preference
        // 1. Try zenity first (GNOME/Unity)
        if _, err := exec.LookPath("zenity"); err == nil {
            cmd = "zenity"
            args = []string{"--get-existing-directory", "--title=" + title}
        } else if _, err := exec.LookPath("kdialog"); err == nil {
            // 2. Fallback to kdialog for KDE systems
            cmd = "kdialog"
            args = []string{"--getexistingdirectory", "--title", title, "/home"}
        } else if _, err := exec.LookPath("yad"); err == nil {
            // 3. Try yad (Yet Another Dialog) which is available on various distros
            cmd = "yad"
            args = []string{"--file", "--directory", "--title=" + title, "--select-dir"}
        } else {
            // 4. If none of the above are available, provide a clear error message
            return "", fmt.Errorf("no suitable directory picker found. Install one of: zenity (GNOME), kdialog (KDE), or yad (multi-desktop)")
        }
    default:
        return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
    
    // Execute the command to show the directory picker
    command := exec.Command(cmd, args...)
    output, err := command.Output()
    if err != nil {
        // Check if the user cancelled the dialog (exit code 1 for zenity, etc.)
        if exitError, ok := err.(*exec.ExitError); ok {
            if exitError.ExitCode() == 1 {
                // User cancelled the dialog - return empty string but no error
                return "", nil
            }
        }
        return "", fmt.Errorf("failed to show directory picker: %v", err)
    }
    
    // Clean up the output (remove trailing newline)
    path := strings.TrimSpace(string(output))
    if path == "" {
        return "", fmt.Errorf("no directory selected")
    }
    
    return path, nil
}

// ShowInFolder opens the containing folder of the given file path in the system's file manager.
// This function is cross-platform and works on Windows, macOS, and Linux.
// It takes a file path and opens the parent directory containing that file.
func (a *App) ShowInFolder(filePath string) error {
	// Get the directory containing the file by taking the parent directory of the file path
	dir := filepath.Dir(filePath)
	
	// Check if directory exists before attempting to open it
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}
	
	// Determine the OS and run appropriate command to open the file manager
	var cmd string
	var args []string
	
	switch runtime.GOOS {
	case "windows":
		// On Windows, use 'cmd /c start' to open the directory
		cmd = "cmd"
		args = []string{"/c", "start", dir}
	case "darwin": // macOS
		// On macOS, use 'open' command to open the directory
		cmd = "open"
		args = []string{dir}
	case "linux":
		// On Linux, use 'xdg-open' command to open the directory (works with most desktop environments)
		cmd = "xdg-open"
		args = []string{dir}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	// Execute the command to open the file manager
	// Use Start() instead of Run() to avoid blocking the application
	command := exec.Command(cmd, args...)
	err := command.Start()
	if err != nil {
		return fmt.Errorf("failed to open folder: %v", err)
	}
	
	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
