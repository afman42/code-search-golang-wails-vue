package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

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
					// emitFileProgress self-detects the last file from the
					// post-increment count, so no racy caller-side isLast here.
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

	// Respect the request's context window (0 = unset -> defaultContextLines),
	// clamped to maxContextLines so request payloads stay bounded.
	ctxLines := searchContextLines(req.ContextLines)

	if meta.size > int64(streamingThreshold) {
		results, procErr := a.processFileLineByLine(ctx, absFilePath, pattern, req.MaxResults-int(atomic.LoadInt32(&searchState.resultsCount)), ctxLines)
		if procErr != nil {
			atomic.AddInt32(&searchState.failedFiles, 1)
			a.logWarn("Error processing file with streaming", logrus.Fields{"filePath": absFilePath, "error": procErr.Error()})
			return "", nil
		}
		return absFilePath, results
	}

	// Re-stat the file before reading: it could have been replaced or grown
	// past MaxFileSize since collection (TOCTOU). Using f.Stat() also
	// catches the case where the file was swapped for a symlink to a larger
	// file — we open the file (which follows symlinks) and check the real
	// size, bounded by the request's MaxFileSize.
	content, err := func() ([]byte, error) {
		f, err := os.Open(absFilePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if info.Size() > req.MaxFileSize {
			return nil, fmt.Errorf("file size %d exceeds max %d", info.Size(), req.MaxFileSize)
		}
		// Exact-size read: one allocation instead of io.ReadAll's doubling
		// growth from a 512B start (up to ~log2(size) intermediate buffers).
		// If the file grew between Stat and Read (TOCTOU), ReadFull returns
		// ErrUnexpectedEOF — read the remainder to preserve prior behavior.
		content := make([]byte, info.Size())
		if _, err := io.ReadFull(f, content); err != nil {
			if err != io.ErrUnexpectedEOF {
				return nil, err
			}
			rest, rerr := io.ReadAll(f)
			if rerr != nil {
				return nil, rerr
			}
			content = append(content, rest...)
		}
		return content, nil
	}()
	if err != nil {
		atomic.AddInt32(&searchState.failedFiles, 1)
		a.logWarn("Error reading file", logrus.Fields{"filePath": absFilePath, "error": err.Error()})
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
			contextBefore := safeContextLinesBytes(lines, i-ctxLines, i)
			contextAfter := safeContextLinesBytes(lines, i+1, i+1+ctxLines)
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

// progressEmitInterval is the minimum interval between "in-progress" events.
// The search can process thousands of files; emitting one IPC event per file
// floods the frontend. Throttling to ~50ms keeps the progress bar smooth
// without overwhelming the IPC bridge.
const progressEmitInterval = 50 * time.Millisecond

// emitFileProgress increments the processed file counter and sends a progress
// event, throttled to progressEmitInterval. The last file always emits so the
// final count is exact. "Last" is determined from the post-increment count
// inside this function (not a caller pre-computation): two workers racing
// between Load and Add would both think they're last otherwise.
func (a *App) emitFileProgress(searchState *SearchState, totalFiles int, absFilePath string) {
	newCount := atomic.AddInt32(&searchState.processedFiles, 1)

	// The single worker whose increment lands on totalFiles is the last one.
	// Computing this from the post-increment value is race-free: exactly one
	// call observes newCount == totalFiles.
	isLast := int(newCount) >= totalFiles

	if !isLast {
		// Throttle via CAS, not Load-then-Store: the load-and-store pair
		// let two workers both pass the window and emit out-of-order
		// progress (processedFiles 6 then 5).
		now := time.Now().UnixNano()
		last := atomic.LoadInt64(&searchState.lastProgressNano)
		if now-last <= int64(progressEmitInterval) {
			return
		}
		if !atomic.CompareAndSwapInt64(&searchState.lastProgressNano, last, now) {
			return // another worker won this interval's slot
		}
	}

	progressData := &SearchProgress{
		ProcessedFiles: int(newCount),
		TotalFiles:     totalFiles,
		CurrentFile:    absFilePath,
		ResultsCount:   int(atomic.LoadInt32(&searchState.resultsCount)),
		FailedFiles:    int(atomic.LoadInt32(&searchState.failedFiles)),
		Status:         "in-progress",
	}
	a.safeEmitEvent("search-progress", progressData)
}
