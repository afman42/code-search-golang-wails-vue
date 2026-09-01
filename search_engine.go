package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// SearchWithProgress performs a search and emits progress updates to the frontend
func (a *App) SearchWithProgress(req SearchRequest) ([]SearchResult, error) {
	// Log the start of the search operation
	searchStart := time.Now()
	a.logInfo("Starting search operation", logrus.Fields{
		"directory":     req.Directory,
		"query":         req.Query,
		"extension":     req.Extension,
		"caseSensitive": req.CaseSensitive,
		"useRegex":      req.UseRegex,
		"maxFileSize":   req.MaxFileSize,
		"maxResults":    req.MaxResults,
		"includeBinary": req.IncludeBinary,
		"excludeCount":  len(req.ExcludePatterns),
		"allowedTypes":  req.AllowedFileTypes,
	})

	// Validate and set defaults for parameters
	validatedReq, err := a.validateAndSetDefaults(req)
	if err != nil {
		a.logError("Search request validation failed", err, logrus.Fields{
			"directory": req.Directory,
			"query":     req.Query,
		})
		return nil, err
	}
	req = validatedReq

	// If query is empty, return empty results instead of error to maintain compatibility
	if req.Query == "" {
		a.logWarn("Empty query provided, returning empty results", logrus.Fields{
			"directory": req.Directory,
		})
		return []SearchResult{}, nil
	}

	// Prepare search pattern based on case sensitivity and regex requirements
	pattern, err := a.compileSearchPattern(req)
	if err != nil {
		a.logError("Failed to compile search pattern", err, logrus.Fields{
			"query":         req.Query,
			"useRegex":      req.UseRegex,
			"caseSensitive": req.CaseSensitive,
		})
		return nil, err
	}

	// Build the list of directories to search. The primary Directory is always
	// included; any AdditionalDirectories are appended (deduplicated).
	searchDirs := []string{req.Directory}
	seen := map[string]bool{filepath.Clean(req.Directory): true}
	for _, d := range req.Directories {
		if d == "" {
			continue
		}
		cleaned := filepath.Clean(d)
		if !seen[cleaned] {
			seen[cleaned] = true
			searchDirs = append(searchDirs, d)
		}
	}

	// Create search context with cancellation. This must happen BEFORE file
	// collection so a user cancel aborts the walk/probe too (not just the
	// worker phase). The stored cancel is cleared only if it is still this
	// search's — an overlapping search may have replaced it (M9).
	ctx, cancel, cancelHandle := a.createSearchContext()
	defer func() {
		a.clearSearchCancel(cancelHandle)
		cancel()
	}()

	// Collect all files to process across all search directories.
	a.logDebug("Collecting files to process", logrus.Fields{
		"directories": searchDirs,
	})
	var filesToProcess []fileMeta
	for _, dir := range searchDirs {
		singleReq := req
		singleReq.Directory = dir
		singleReq.Directories = nil // avoid recursion
		dirFiles, err := a.collectFilesToProcess(ctx, singleReq, pattern)
		if err != nil {
			a.logError("Failed to collect files for directory", err, logrus.Fields{
				"directory": dir,
				"query":     req.Query,
			})
			return nil, err
		}
		filesToProcess = append(filesToProcess, dirFiles...)
	}

	// Dedupe by absolute path: nested search directories (Directory=/a plus
	// Directories=[/a/sub]) walk the same files twice, which would duplicate
	// every result under the nested dir.
	seenFiles := make(map[string]bool, len(filesToProcess))
	deduped := filesToProcess[:0]
	for _, f := range filesToProcess {
		if seenFiles[f.absPath] {
			continue
		}
		seenFiles[f.absPath] = true
		deduped = append(deduped, f)
	}
	filesToProcess = deduped

	totalFiles := len(filesToProcess)
	a.logInfo("File collection completed", logrus.Fields{
		"totalFiles": totalFiles,
		"directory":  req.Directory,
	})

	// Emit initial progress using the SearchProgress struct
	initialProgress := &SearchProgress{
		ProcessedFiles: 0,
		TotalFiles:     totalFiles,
		CurrentFile:    "",
		ResultsCount:   0,
		Status:         "started",
	}

	a.logInfo("Sending initial search progress", logrus.Fields{
		"status":       "started",
		"totalFiles":   totalFiles,
		"currentFile":  "",
		"resultsCount": 0,
	})

	a.safeEmitEvent("search-progress", initialProgress)

	// Log search start
	a.logInfo("Starting file processing with worker pool", logrus.Fields{
		"totalFiles": totalFiles,
		"workers":    numCPU(),
		"maxResults": req.MaxResults,
	})

	// Process files using worker pool
	resultsChan, searchState := a.processFilesWithWorkers(ctx, cancel, filesToProcess, req, pattern, totalFiles)

	// Collect results
	var results []SearchResult
	for result := range resultsChan {
		results = append(results, result)

		// Check if we've reached the result limit
		if len(results) >= req.MaxResults {
			a.logInfo("Reached maximum results limit, stopping search", logrus.Fields{
				"resultsCount": len(results),
				"maxResults":   req.MaxResults,
			})
			// The context is already cancelled by the workers, but we'll do it again just in case
			cancel()
			// Trim results to max results if somehow we got more
			if len(results) > req.MaxResults {
				results = results[:req.MaxResults]
			}
			break
		}
	}

	// Check if the search was cancelled (by the user or another frame) before
	// emitting a misleading "completed" event. The CancelSearch binding already
	// emitted a "cancelled" event for user cancels, and returning empty results
	// keeps the frontend from repopulating the list with whatever partial
	// matches raced into the channel before cancellation.
	if ctx.Err() != nil && len(results) < req.MaxResults {
		a.logInfo("Search operation was cancelled", logrus.Fields{
			"directory":       req.Directory,
			"query":           req.Query,
			"durationSeconds": time.Since(searchStart).Seconds(),
		})
		return []SearchResult{}, nil
	}

	// Phase 2: fuzzy near-miss candidates fill any remaining quota. The exact pass above is untouched by this; fuzzy only appends, and only lines the exact pattern did not match, so enabling it never changes exact results. With CaseSensitive=true this means case-variant occurrences surface as plain results (the frontend does not badge lines that contain the query case-insensitively) while true near-misses get flagged with a fuzzy badge.
	if req.FuzzySearch && !req.UseRegex && len(results) < req.MaxResults {
		fuzzyQuota := req.MaxResults - len(results)
		fuzzyCtx, fuzzyCancel := context.WithCancel(ctx)
		fuzzyResults := a.searchFuzzyCandidates(fuzzyCtx, filesToProcess, req, pattern, fuzzyQuota)
		fuzzyCancel() // abort any in-flight fuzzy scans when done
		results = append(results, fuzzyResults...)
		a.logInfo("Fuzzy candidate pass completed", logrus.Fields{
			"exactMatches":    len(results) - len(fuzzyResults),
			"fuzzyCandidates": len(fuzzyResults),
		})
	}
	// Sort results by file path then line number so output is deterministic
	// regardless of worker completion order.
	sort.Slice(results, func(i, j int) bool {
		if results[i].FilePath != results[j].FilePath {
			return results[i].FilePath < results[j].FilePath
		}
		return results[i].LineNum < results[j].LineNum
	})

	// Re-check cancellation right before the "completed" emit: a cancel that
	// lands after the check above but before this emit would otherwise
	// produce BOTH a "cancelled" event (from CancelSearch) and a "completed"
	// one, and the UI would repopulate results the user just cancelled.
	// The len < MaxResults guard distinguishes a USER cancel from the
	// limit-triggered cancel() that fires when MaxResults is reached.
	if ctx.Err() != nil && len(results) < req.MaxResults {
		a.logInfo("Search operation was cancelled", logrus.Fields{
			"directory":       req.Directory,
			"query":           req.Query,
			"durationSeconds": time.Since(searchStart).Seconds(),
		})
		return []SearchResult{}, nil
	}

	// Emit final progress using the SearchProgress struct
	finalProgress := &SearchProgress{
		ProcessedFiles: int(atomic.LoadInt32(&searchState.processedFiles)),
		TotalFiles:     totalFiles,
		CurrentFile:    "",
		ResultsCount:   len(results),
		FailedFiles:    int(atomic.LoadInt32(&searchState.failedFiles)),
		Status:         "completed",
	}

	a.logInfo("Sending final search progress", logrus.Fields{
		"status":         "completed",
		"processedFiles": int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":     totalFiles,
		"resultsCount":   len(results),
		"failedFiles":    int(atomic.LoadInt32(&searchState.failedFiles)),
	})

	a.safeEmitEvent("search-progress", finalProgress)

	// Log search completion
	duration := time.Since(searchStart)
	a.logInfo("Search operation completed", logrus.Fields{
		"resultsCount":    len(results),
		"processedFiles":  int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":      totalFiles,
		"failedFiles":     int(atomic.LoadInt32(&searchState.failedFiles)),
		"durationSeconds": duration.Seconds(),
		"directory":       req.Directory,
		"query":           req.Query,
	})

	return results, nil
}

// Helper function to get number of CPUs
func numCPU() int {
	n := runtime.NumCPU()
	if n < 2 {
		return 2 // Use at least 2 workers for parallelism
	}
	return n
}

// createSearchContext creates a context for the search operation with
// associated cancellation. Returns the handle so the caller can later clear
// its own cancel without clobbering an overlapping search's.
func (a *App) createSearchContext() (context.Context, context.CancelFunc, *searchCancelHandle) {
	ctx, cancel := context.WithCancel(context.Background())
	handle := &searchCancelHandle{cancel: cancel}
	// Store the handle so it can be called externally to cancel the search
	a.setSearchCancel(cancel)
	return ctx, cancel, handle
}

// CancelSearch cancels any active search operation by calling the cancel function
func (a *App) CancelSearch() error {
	if a.cancelActiveSearch() {
		a.logInfo("Cancelling active search", logrus.Fields{})
		// Emit cancellation progress event
		cancelData := &SearchProgress{
			ProcessedFiles: 0,
			TotalFiles:     0,
			CurrentFile:    "",
			ResultsCount:   0,
			Status:         "cancelled",
		}

		a.logInfo("Sending cancellation progress event", logrus.Fields{
			"status":         "cancelled",
			"processedFiles": 0,
			"totalFiles":     0,
			"resultsCount":   0,
		})
		a.safeEmitEvent("search-progress", cancelData)

		return nil
	}
	// If there's no active search to cancel, return an appropriate message
	a.logDebug("No active search to cancel", logrus.Fields{})
	return fmt.Errorf("no active search to cancel")
}
