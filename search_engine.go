package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
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
		"searchSubdirs": req.SearchSubdirs,
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

	// Get the base directory for path traversal check
	absDir, err := filepath.Abs(req.Directory)
	if err != nil {
		a.logError("Failed to get absolute path for directory", err, logrus.Fields{
			"directory": req.Directory,
		})
		return nil, fmt.Errorf("failed to get absolute path for directory: %v", err)
	}
	baseDir := filepath.Clean(absDir) + string(filepath.Separator)

	// Collect all files to process based on search criteria
	a.logDebug("Collecting files to process", logrus.Fields{
		"directory": req.Directory,
	})
	filesToProcess, err := a.collectFilesToProcess(req, pattern, baseDir)
	if err != nil {
		a.logError("Failed to collect files to process", err, logrus.Fields{
			"directory": req.Directory,
			"query":     req.Query,
		})
		return nil, err
	}

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

	// Create search context with cancellation
	ctx, cancel := a.createSearchContext()
	defer func() {
		// Clear the cancel function when the search completes
		a.clearSearchCancel()
		cancel()
	}()

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

	// Emit final progress using the SearchProgress struct
	finalProgress := &SearchProgress{
		ProcessedFiles: int(atomic.LoadInt32(&searchState.processedFiles)),
		TotalFiles:     totalFiles,
		CurrentFile:    "",
		ResultsCount:   len(results),
		Status:         "completed",
	}

	a.logInfo("Sending final search progress", logrus.Fields{
		"status":         "completed",
		"processedFiles": int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":     totalFiles,
		"resultsCount":   len(results),
	})

	a.safeEmitEvent("search-progress", finalProgress)

	// Log search completion
	duration := time.Since(searchStart)
	a.logInfo("Search operation completed", logrus.Fields{
		"resultsCount":    len(results),
		"processedFiles":  int(atomic.LoadInt32(&searchState.processedFiles)),
		"totalFiles":      totalFiles,
		"durationSeconds": duration.Seconds(),
		"directory":       req.Directory,
		"query":           req.Query,
	})

	return results, nil
}

// fileMeta carries the per-file metadata gathered during collection so the
// worker pool can process a file without repeating syscalls. The absolute path
// and size are computed once in collectFilesToProcess (file_collection.go);
// reusing them avoids a second os.Stat and filepath.Abs per file.
type fileMeta struct {
	absPath string
	size    int64
}

// binaryCheckBufPool reuses the 512-byte scratch buffer used by the binary
// detection probe. The pool returns *[]byte so the slice header isn't pinned
// and the backing array can be reused across files. Used by the parallel
// binary probe in file_collection.go.
var binaryCheckBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 512)
		return &buf
	},
}

// streamContextLines is the number of lines captured before and after each match
// during streaming (line-by-line) processing. It mirrors the context window used
// for small files so results are consistent regardless of file size.
const streamContextLines = 2

// processFileLineByLine processes a file line by line to avoid loading large files into memory.
// Binary detection is already performed upstream in collectFilesToProcess.
//
// Context lines (up to streamContextLines before and after each match) are captured
// the same way as the small-file path: a rolling buffer holds recent lines for
// ContextBefore, and matches stay "pending" until enough following lines are read
// to fill ContextAfter.
func (a *App) processFileLineByLine(ctx context.Context, filePath string, pattern *regexp.Regexp, maxResults int) ([]SearchResult, error) {
	a.logDebug("Starting line-by-line file processing", logrus.Fields{
		"filePath":   filePath,
		"maxResults": maxResults,
	})

	file, err := os.Open(filePath)
	if err != nil {
		a.logError("Failed to open file for line-by-line processing", err, logrus.Fields{
			"filePath": filePath,
		})
		return nil, err
	}
	defer file.Close()

	var results []SearchResult
	scanner := bufio.NewScanner(file)

	// Set a larger buffer for very long lines (1MB)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	// prev holds up to streamContextLines preceding lines for ContextBefore.
	prev := make([]string, 0, streamContextLines)
	// pending tracks matches (by index into results) still awaiting ContextAfter lines.
	type pendingMatch struct {
		idx       int
		remaining int
	}
	var pending []pendingMatch

	lineNum := 1
	linesProcessed := 0
	for scanner.Scan() {
		line := scanner.Text()

		// Fill ContextAfter for matches found on earlier lines.
		if len(pending) > 0 {
			stillPending := pending[:0]
			for _, p := range pending {
				results[p.idx].ContextAfter = append(results[p.idx].ContextAfter, line)
				p.remaining--
				if p.remaining > 0 {
					stillPending = append(stillPending, p)
				}
			}
			pending = stillPending
		}

		// Record a new match (unless we've already hit the result limit).
		if len(results) < maxResults && pattern.MatchString(line) {
			contextBefore := make([]string, len(prev))
			copy(contextBefore, prev)
			results = append(results, SearchResult{
				FilePath:      filePath,
				LineNum:       lineNum,
				Content:       strings.TrimSpace(line),
				MatchedText:   pattern.FindString(line),
				ContextBefore: contextBefore,
				ContextAfter:  []string{},
			})
			pending = append(pending, pendingMatch{idx: len(results) - 1, remaining: streamContextLines})
		}

		// Advance the rolling buffer of preceding lines.
		prev = append(prev, line)
		if len(prev) > streamContextLines {
			prev = prev[1:]
		}

		lineNum++
		linesProcessed++

		// Stop once the result limit is reached and every match has its trailing context.
		if len(results) >= maxResults && len(pending) == 0 {
			break
		}

		if linesProcessed%100 == 0 {
			select {
			case <-ctx.Done():
				a.logDebug("Line-by-line processing cancelled due to context", logrus.Fields{
					"filePath":       filePath,
					"linesProcessed": linesProcessed,
					"resultsFound":   len(results),
				})
				return results, nil
			default:
			}
		}
	}

	if err := scanner.Err(); err != nil {
		a.logError("Error during line-by-line scanning", err, logrus.Fields{
			"filePath": filePath,
		})
		return nil, err
	}

	a.logDebug("Completed line-by-line file processing", logrus.Fields{
		"filePath":       filePath,
		"resultsFound":   len(results),
		"linesProcessed": linesProcessed,
	})
	return results, nil
}

// streamingThreshold is the file size (in bytes) above which files are processed
// line-by-line instead of being read entirely into memory.
const streamingThreshold = 1024 * 1024 // 1MB

// Helper function to get number of CPUs
func numCPU() int {
	n := runtime.NumCPU()
	if n < 2 {
		return 2 // Use at least 2 workers for parallelism
	}
	return n
}

// createSearchContext creates a context for the search operation with associated cancellation
func (a *App) createSearchContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	// Store the cancel function so it can be called externally to cancel the search
	a.setSearchCancel(cancel)
	return ctx, cancel
}

// processFilesWithWorkers processes files using a worker pool and returns a channel of results
func (a *App) processFilesWithWorkers(ctx context.Context, cancel context.CancelFunc, filesToProcess []fileMeta, req SearchRequest, pattern *regexp.Regexp, totalFiles int) (chan SearchResult, *SearchState) {
	numWorkers := numCPU()
	if len(filesToProcess) < numWorkers {
		numWorkers = len(filesToProcess)
	}

	a.logDebug("Initializing worker pool", logrus.Fields{
		"numWorkers":         numWorkers,
		"totalFiles":         totalFiles,
		"maxResults":         req.MaxResults,
		"streamingThreshold": int64(streamingThreshold),
	})

	filesChan := make(chan fileMeta, len(filesToProcess))
	resultsChan := make(chan SearchResult, 100)

	searchState := &SearchState{}
	var searchCancelled int32

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case meta, ok := <-filesChan:
					if !ok {
						return
					}

					if !a.workerShouldContinue(ctx, &searchCancelled, cancel, &searchState.resultsCount, req.MaxResults, workerID) {
						return
					}

					absFilePath, fileResults := a.processFile(ctx, meta, pattern, req, searchState, &searchCancelled, cancel)
					if absFilePath == "" {
						continue
					}

					// Send results and emit progress
					a.emitFileResults(ctx, fileResults, resultsChan, searchState, &searchCancelled, cancel, req.MaxResults)
					a.emitFileProgress(searchState, totalFiles, absFilePath)
				}
			}
		}()
	}

	// Send files to channel
	go func() {
		defer close(filesChan)
		for _, file := range filesToProcess {
			select {
			case <-ctx.Done():
				return
			case filesChan <- file:
			}
		}
	}()

	// Close results when all workers finish
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	return resultsChan, searchState
}

// workerShouldContinue checks whether the worker should stop (context cancelled
// or max results reached). If max results is reached, it cancels the context
// atomically to prevent duplicate cancellations.
func (a *App) workerShouldContinue(ctx context.Context, searchCancelled *int32, cancel context.CancelFunc, resultsCount *int32, maxResults int, workerID int) bool {
	if int(atomic.LoadInt32(resultsCount)) >= maxResults {
		if atomic.CompareAndSwapInt32(searchCancelled, 0, 1) {
			cancel()
		}
		return false
	}
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// processFile attempts to process a single file and return its search results.
// Returns the absolute path (or "" if the file was skipped) and any results found.
//
// The file's absolute path and size come from collectFilesToProcess (via meta),
// so this function does not re-stat the file or re-validate path traversal —
// both were already done during collection.
//
// Binary detection is ALSO already done in collectFilesToProcess when
// !req.IncludeBinary, so this function does not re-check binary status for
// small files. Re-checking would waste a full-file read on every small file
// (#4). The only exception is when req.IncludeBinary is true (the user
// explicitly asked to search binaries), in which case we read the file and
// search it regardless.
func (a *App) processFile(ctx context.Context, meta fileMeta, pattern *regexp.Regexp, req SearchRequest, searchState *SearchState, searchCancelled *int32, cancel context.CancelFunc) (string, []SearchResult) {
	absFilePath := meta.absPath

	if meta.size > int64(streamingThreshold) {
		results, procErr := a.processFileLineByLine(ctx, absFilePath, pattern, req.MaxResults-int(atomic.LoadInt32(&searchState.resultsCount)))
		if procErr != nil {
			a.logDebug("Error processing file with streaming", logrus.Fields{"filePath": absFilePath, "error": procErr.Error()})
			return "", nil
		}
		return absFilePath, results
	}

	content, err := os.ReadFile(absFilePath)
	if err != nil {
		a.logDebug("Skipping file due to read error", logrus.Fields{"filePath": absFilePath, "error": err.Error()})
		return "", nil
	}

	// Binary re-check is intentionally omitted here: when !req.IncludeBinary,
	// collectFilesToProcess already filtered binary files out, so re-checking
	// would just waste a pass over every small file's content (#4). When
	// req.IncludeBinary is true, the user wants binary files searched.

	// Use bytes.Split instead of strings.Split to avoid the string(content)
	// copy for sub-1MB files (#10). The previous strings.Split path allocated
	// a string (full-file copy) plus a []string slice of line count; for a
	// 900KB file with 15k lines that's ~16k allocations. bytes.Split keeps
	// the line slices as views into the original []byte, and we only convert
	// a line to string when we need to put it on a SearchResult field.
	lines := bytes.Split(content, []byte("\n"))
	var fileResults []SearchResult

	for i, line := range lines {
		if !a.workerShouldContinue(ctx, searchCancelled, cancel, &searchState.resultsCount, req.MaxResults, -1) {
			break
		}

		if pattern.Match(line) {
			contextBefore := safeContextLinesBytes(lines, i-2, i)
			contextAfter := safeContextLinesBytes(lines, i+1, i+3)
			matchedText := pattern.Find(line)

			fileResults = append(fileResults, SearchResult{
				FilePath:      absFilePath,
				LineNum:       i + 1,
				Content:       strings.TrimSpace(string(line)),
				MatchedText:   string(matchedText),
				ContextBefore: bytesToStrings(contextBefore),
				ContextAfter:  bytesToStrings(contextAfter),
			})
		}
	}

	return absFilePath, fileResults
}

// emitFileResults sends each result from processing a file to the results channel,
// respecting context cancellation and max results limits.
func (a *App) emitFileResults(ctx context.Context, fileResults []SearchResult, resultsChan chan<- SearchResult, searchState *SearchState, searchCancelled *int32, cancel context.CancelFunc, maxResults int) {
	for _, result := range fileResults {
		if int(atomic.LoadInt32(&searchState.resultsCount)) >= maxResults {
			if atomic.CompareAndSwapInt32(searchCancelled, 0, 1) {
				cancel()
			}
			return
		}

		select {
		case resultsChan <- result:
			newCount := atomic.AddInt32(&searchState.resultsCount, 1)
			if int(newCount) >= maxResults {
				if atomic.CompareAndSwapInt32(searchCancelled, 0, 1) {
					cancel()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// emitFileProgress increments the processed file counter and sends a progress event.
func (a *App) emitFileProgress(searchState *SearchState, totalFiles int, absFilePath string) {
	newCount := atomic.AddInt32(&searchState.processedFiles, 1)
	progressData := &SearchProgress{
		ProcessedFiles: int(newCount),
		TotalFiles:     totalFiles,
		CurrentFile:    absFilePath,
		ResultsCount:   int(atomic.LoadInt32(&searchState.resultsCount)),
		Status:         "in-progress",
	}
	a.safeEmitEvent("search-progress", progressData)
}

// safeContextLines returns a slice of lines[start:end] that is safe even when
// start or end are out of bounds.
func safeContextLines(lines []string, start, end int) []string {
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start >= end {
		return []string{}
	}
	return lines[start:end]
}

// safeContextLinesBytes is the byte-slice counterpart to safeContextLines,
// used by the small-file processing path that splits file content with
// bytes.Split instead of strings.Split (#10). It returns sub-slices that
// share the original byte buffer; callers that need to keep a line beyond
// the lifetime of the content buffer should copy it (bytesToStrings below
// does that conversion).
func safeContextLinesBytes(lines [][]byte, start, end int) [][]byte {
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start >= end {
		return nil
	}
	return lines[start:end]
}

// bytesToStrings converts a slice of byte slices to a slice of strings. Used
// when ContextBefore/ContextAfter need to be stored on a SearchResult (which
// holds []string). The conversion copies each line so the result doesn't
// keep the (potentially large) content buffer alive after processFile returns.
func bytesToStrings(lines [][]byte) []string {
	if len(lines) == 0 {
		return []string{}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = string(l)
	}
	return out
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
