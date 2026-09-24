package main

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// SearchWithProgress performs a search and emits progress updates to the frontend
func (a *App) SearchWithProgress(req SearchRequest) ([]SearchResult, error) {
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
	req, pattern, err := a.prepareSearch(req)
	if err != nil {
		return nil, err
	}
	if pattern == nil {
		// Empty query — prepareSearch logged the warning; preserve the
		// original early-return of empty results instead of proceeding with
		// a zero-value regexp (which would panic on .Match).
		return []SearchResult{}, nil
	}

	ctx, cancel, cancelHandle := a.createSearchContext()
	defer func() {
		a.clearSearchCancel(cancelHandle)
		cancel()
	}()

	filesToProcess, totalFiles := a.collectSearchFiles(ctx, req, pattern)
	a.emitSearchProgress(searchProgressParams{Processed: 0, Total: totalFiles, Current: "", ResultsCount: 0, FailedFiles: 0, FailedPaths: nil, Status: "started"})

	resultsChan, searchState := a.processFilesWithWorkers(ctx, cancel, filesToProcess, req, pattern, totalFiles)
	batcher := newResultBatcher(a)
	results := a.drainResults(resultsChan, batcher, req.MaxResults, cancel)
	batcher.flush()

	if a.searchCancelled(ctx, results, req.MaxResults, searchStart) {
		return []SearchResult{}, nil
	}

	results = a.appendFuzzy(ctx, results, filesToProcess, req, pattern, batcher)
	sortSearchResults(results)

	if a.searchCancelled(ctx, results, req.MaxResults, searchStart) {
		return []SearchResult{}, nil
	}

	failedPaths := searchState.snapshotFailedPaths()
	a.emitSearchProgress(searchProgressParams{
		Processed:    int(atomic.LoadInt32(&searchState.processedFiles)),
		Total:        totalFiles,
		Current:      "",
		ResultsCount: len(results),
		FailedFiles:  int(atomic.LoadInt32(&searchState.failedFiles)),
		FailedPaths:  failedPaths,
		Status:       "completed",
	})
	a.logInfo("Search operation completed", logrus.Fields{
		"resultsCount":    len(results),
		"processedFiles":  int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":      totalFiles,
		"failedFiles":     int(atomic.LoadInt32(&searchState.failedFiles)),
		"durationSeconds": time.Since(searchStart).Seconds(),
		"directory":       req.Directory,
		"query":           req.Query,
	})
	return results, nil
}

// prepareSearch validates the request, applies defaults, and compiles the
// search pattern. Returns the validated request and compiled pattern.
func (a *App) prepareSearch(req SearchRequest) (SearchRequest, *regexp.Regexp, error) {
	validatedReq, err := a.validateAndSetDefaults(req)
	if err != nil {
		a.logError("Search request validation failed", err, logrus.Fields{
			"directory": req.Directory,
			"query":     req.Query,
		})
		return req, nil, err
	}
	req = validatedReq

	// Empty query returns empty results instead of error to maintain compatibility.
	// Return a nil pattern so SearchWithProgress takes the early-return path
	// instead of proceeding with a zero-value regexp (which panics on .Match).
	if req.Query == "" {
		a.logWarn("Empty query provided, returning empty results", logrus.Fields{
			"directory": req.Directory,
		})
		return req, nil, nil
	}

	pattern, err := a.compileSearchPattern(req)
	if err != nil {
		a.logError("Failed to compile search pattern", err, logrus.Fields{
			"query":         req.Query,
			"useRegex":      req.UseRegex,
			"caseSensitive": req.CaseSensitive,
		})
		return req, nil, err
	}
	return req, pattern, nil
}

// collectSearchFiles gathers and dedupes files across all search directories.
// A per-directory collect error aborts the whole search (logged here).
func (a *App) collectSearchFiles(ctx context.Context, req SearchRequest, pattern *regexp.Regexp) ([]fileMeta, int) {
	a.logDebug("Collecting files to process", logrus.Fields{
		"directories": expandSearchDirs(req),
	})
	filesToProcess, err := collectAcrossDirs(ctx, req,
		func(c context.Context, singleReq SearchRequest) ([]fileMeta, error) {
			return a.collectFilesToProcess(c, singleReq, pattern)
		},
		func(dir string, err error) ([]fileMeta, error) {
			a.logError("Failed to collect files for directory", err, logrus.Fields{
				"directory": dir,
				"query":     req.Query,
			})
			return nil, nil
		})
	if err != nil {
		return nil, 0
	}

	totalFiles := len(filesToProcess)
	a.logInfo("File collection completed", logrus.Fields{
		"totalFiles": totalFiles,
		"directory":  req.Directory,
	})
	return filesToProcess, totalFiles
}

// drainResults reads from the worker channel, accumulating results and
// pushing them to the frontend in batches. Stops early when MaxResults is
// reached.
func (a *App) drainResults(resultsChan chan SearchResult, batcher *resultBatcher, maxResults int, cancel context.CancelFunc) []SearchResult {
	var results []SearchResult
	for result := range resultsChan {
		results = append(results, result)
		batcher.add(result)
		if len(results) >= maxResults {
			a.logInfo("Reached maximum results limit, stopping search", logrus.Fields{
				"resultsCount": len(results),
				"maxResults":   maxResults,
			})
			cancel()
			if len(results) > maxResults {
				results = results[:maxResults]
			}
			break
		}
	}
	return results
}

// appendFuzzy runs the fuzzy near-miss pass to fill any remaining quota.
func (a *App) appendFuzzy(ctx context.Context, results []SearchResult, filesToProcess []fileMeta, req SearchRequest, pattern *regexp.Regexp, batcher *resultBatcher) []SearchResult {
	if !req.FuzzySearch || req.UseRegex || len(results) >= req.MaxResults {
		return results
	}
	fuzzyQuota := req.MaxResults - len(results)
	fuzzyCtx, fuzzyCancel := context.WithCancel(ctx)
	fuzzyResults := a.searchFuzzyCandidates(fuzzyCtx, filesToProcess, req, pattern, fuzzyQuota)
	fuzzyCancel()
	results = append(results, fuzzyResults...)
	for _, r := range fuzzyResults {
		batcher.add(r)
	}
	batcher.flush()
	a.logInfo("Fuzzy candidate pass completed", logrus.Fields{
		"exactMatches":    len(results) - len(fuzzyResults),
		"fuzzyCandidates": len(fuzzyResults),
	})
	return results
}

// searchCancelled reports whether the search was cancelled before completion.
func (a *App) searchCancelled(ctx context.Context, results []SearchResult, maxResults int, searchStart time.Time) bool {
	if ctx.Err() != nil && len(results) < maxResults {
		a.logInfo("Search operation was cancelled", logrus.Fields{
			"durationSeconds": time.Since(searchStart).Seconds(),
		})
		return true
	}
	return false
}

// searchProgressParams groups emitSearchProgress arguments (Introduce Parameter Object).
// Replaces 7 loose params with a single value object, making call sites readable
// and future extensions (e.g. adding elapsed) non-breaking.
type searchProgressParams struct {
	Processed    int
	Total        int
	Current      string
	ResultsCount int
	FailedFiles  int
	FailedPaths  []string
	Status       string
}

// emitSearchProgress emits a search-progress event with the given state.
func (a *App) emitSearchProgress(p searchProgressParams) {
	a.safeEmitEvent("search-progress", &SearchProgress{
		ProcessedFiles: p.Processed,
		TotalFiles:     p.Total,
		CurrentFile:    p.Current,
		ResultsCount:   p.ResultsCount,
		FailedFiles:    p.FailedFiles,
		FailedPaths:    p.FailedPaths,
		Status:         p.Status,
	})
	a.logInfo("Sending search progress", logrus.Fields{
		"status":         p.Status,
		"processedFiles": p.Processed,
		"totalFiles":     p.Total,
		"resultsCount":   p.ResultsCount,
		"failedFiles":    p.FailedFiles,
		"failedSampled":  len(p.FailedPaths),
	})
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
	// Store the handle so it can be cancelled externally, and so
	// clearSearchCancel can retire this exact search by pointer identity.
	a.setSearchCancel(handle)
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
