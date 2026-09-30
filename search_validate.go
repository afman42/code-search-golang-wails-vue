package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func (a *App) validateAndSetDefaults(req SearchRequest) (SearchRequest, error) {
	if len(req.Query) > maxQueryLength {
		return req, fmt.Errorf("%w: %d chars (max %d)", ErrQueryTooLong, len(req.Query), maxQueryLength)
	}
	// Set default values for optional parameters
	modifiedReq := req
	if modifiedReq.MaxFileSize <= 0 {
		modifiedReq.MaxFileSize = 10 * 1024 * 1024 // 10MB default
	}
	if modifiedReq.MaxResults <= 0 {
		modifiedReq.MaxResults = 1000 // 1000 results default
	}
	if modifiedReq.MaxResults > maxAllowedResults {
		return req, fmt.Errorf("%w: %d (max %d)", ErrMaxResultsExceeded, modifiedReq.MaxResults, maxAllowedResults)
	}

	// Validate directory is not empty
	if modifiedReq.Directory == "" {
		return req, fmt.Errorf("%w: empty directory path provided", ErrDirectoryNotFound)
	}

	// Before proceeding with file operations, validate that the final resolved directory is not a result of
	// dangerous path traversal that could cause access to unintended scopes
	cleanPath := filepath.Clean(modifiedReq.Directory)

	// Validate directory exists before starting the search
	if _, err := os.Stat(cleanPath); errors.Is(err, fs.ErrNotExist) {
		return req, fmt.Errorf("%w: %s", ErrDirectoryNotFound, cleanPath)
	}

	// Get absolute path for internal processing
	absDir, err := filepath.Abs(cleanPath)
	if err != nil {
		return req, fmt.Errorf("failed to get absolute path for directory: %w", err)
	}

	// Prevent searching system-critical directories and their subtrees.
	// Exact match blocks the directory itself; prefix+separator blocks
	// subtrees like /etc/ssh when /etc is protected, without blocking
	// unrelated paths like /etc-backup.
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
		if cleanBaseDir == protected || strings.HasPrefix(cleanBaseDir, protected+string(filepath.Separator)) {
			return req, fmt.Errorf("%w: %s", ErrProtectedDirectory, cleanBaseDir)
		}
	}

	return modifiedReq, nil
}

// compileSearchPattern prepares the search pattern based on case sensitivity and regex requirements
// Uses LRU cache to avoid recompiling frequently-used patterns
func (a *App) compileSearchPattern(req SearchRequest) (*regexp.Regexp, error) {
	var pattern *regexp.Regexp
	var err error

	// Determine whether to treat the query as a regex or as a literal.
	// UseRegex is a plain bool (zero value = false = literal search), matching
	// what the frontend always sends.
	useRegex := req.UseRegex

	// Build cache key including fuzziness flag
	cacheKey := getPatternCacheKey(useRegex, req.CaseSensitive, req.Query)

	// Try to get from LRU cache first
	if a.patternCache != nil {
		if cached, ok := a.patternCache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	searchPattern := req.Query
	if useRegex {
		// If using regex, use the query as-is (with case sensitivity flag)
		if !req.CaseSensitive {
			// Use the (?i) flag for case insensitive matching
			searchPattern = "(?i)" + req.Query
		}
	} else {
		// For literal search, escape special regex characters. An "invalid
		// regex" query (e.g. "[unclosed") is a perfectly valid literal
		// string once QuoteMeta has escaped it, so we do NOT separately
		// compile the raw query here — that was dead work that ran an extra
		// regex compile on every literal search just to satisfy a single
		// test that was itself testing the wrong mode (#11).
		searchPattern = regexp.QuoteMeta(req.Query)
		if !req.CaseSensitive {
			searchPattern = "(?i)" + searchPattern
		}
	}
	pattern, err = regexp.Compile(searchPattern)

	if err != nil {
		return nil, fmt.Errorf("invalid search pattern: %w", err)
	}

	// Cache the compiled pattern
	if a.patternCache != nil {
		a.patternCache.Set(cacheKey, pattern)
	}

	return pattern, nil
}
