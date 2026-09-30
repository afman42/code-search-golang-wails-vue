package main

import "errors"

// Sentinel errors for validation and lookup failures. Messages match the
// historical strings exactly so errors.Is works without breaking existing
// text assertions. Wrap with fmt.Errorf("%w: ...") — never %v on live errors.
var (
	ErrPathTraversal      = errors.New("invalid file path: contains directory traversal")
	ErrDirectoryNotFound  = errors.New("directory does not exist")
	ErrFileNotFound       = errors.New("file does not exist")
	ErrEmptyPath          = errors.New("file path is required")
	ErrInvalidLogFileName = errors.New("invalid log file name")
	ErrProtectedDirectory = errors.New("searching in protected system directory not allowed")
	ErrQueryTooLong       = errors.New("query too long")
	ErrMaxResultsExceeded = errors.New("maxResults too large")
	ErrNoActiveSearch     = errors.New("no active search to cancel")
)
