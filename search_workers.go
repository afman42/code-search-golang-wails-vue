package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// progressEmitInterval is the minimum interval between "in-progress" events.
// The search can process thousands of files; emitting one IPC event per file
// floods the frontend. Throttling to ~50ms keeps the progress bar smooth
// without overwhelming the IPC bridge.
const progressEmitInterval = 50 * time.Millisecond

// maxFailedPathsReported bounds how many unreadable file paths a search
// remembers for the UI. The count (SearchProgress.FailedFiles) stays exact;
// this only caps the listed sample, so a tree with 50k permission-denied
// files cannot balloon the terminal event payload.
const maxFailedPathsReported = 50

// resultBatchSize is how many results accumulate before a "search-results"
// batch is pushed. Paired with progressEmitInterval as the time-based flush,
// it keeps a fast search from emitting one IPC event per match while still
// rendering incrementally on a slow one.
const resultBatchSize = 256

// searchResultsChanBuf bounds the worker/fuzzy results channels: large enough
// that workers rarely block on the drain goroutine, small enough that an
// unbounded run cannot buffer an unclamped flood in memory.
const searchResultsChanBuf = 100

// searchJob bundles the inputs the search phases share per run, mirroring
// the walkCtx/replaceJob precedent. req is a pointer so the worker and fuzzy
// phases don't copy the request per call; the Wails entry
// (SearchWithProgress) keeps its by-value signature.
type searchJob struct {
	files   []fileMeta
	req     *SearchRequest
	pattern *regexp.Regexp
	total   int
}

// searchWorker bundles the shared per-search worker state: the atomic
// counters (resultsCount lives on state, not duplicated here), the
// exactly-once cancellation flag, and the result cap.
type searchWorker struct {
	state      *SearchState
	cancelled  *int32
	cancel     context.CancelFunc
	maxResults int
}

// cancelOnce marks the search cancelled and cancels the context exactly
// once, so racing workers cannot issue duplicate cancellations.
func (w searchWorker) cancelOnce() {
	if atomic.CompareAndSwapInt32(w.cancelled, 0, 1) {
		w.cancel()
	}
}

// processFilesWithWorkers processes files using a worker pool and returns a channel of results.
//
// Best-effort quota: workers check the shared resultsCount and cancel on
// reaching maxResults, but in-flight workers can each emit one more batch,
// so the drained total may overshoot slightly; drainResults trims at the cap.
func (a *App) processFilesWithWorkers(
	ctx context.Context,
	cancel context.CancelFunc,
	job searchJob,
) (<-chan SearchResult, *SearchState) {
	numWorkers := numCPU()
	if len(job.files) < numWorkers {
		numWorkers = len(job.files)
	}

	a.logDebug("Initializing worker pool", logrus.Fields{
		"numWorkers":         numWorkers,
		"totalFiles":         job.total,
		"maxResults":         job.req.MaxResults,
		"streamingThreshold": int64(streamingThreshold),
	})

	// Bounded to min(n, 2*workers): a full-file buffer lets the feeder run
	// ahead unbounded on huge trees; 2*workers keeps workers fed without
	// hoarding memory. Results use the shared bounded const.
	filesChan := make(chan fileMeta, min(len(job.files), 2*numWorkers))
	resultsChan := make(chan SearchResult, searchResultsChanBuf)

	searchState := &SearchState{}
	var searchCancelled int32
	w := searchWorker{
		state:      searchState,
		cancelled:  &searchCancelled,
		cancel:     cancel,
		maxResults: job.req.MaxResults,
	}

	var wg sync.WaitGroup
	for i := range numWorkers {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					a.logError("Search worker panicked", nil, logrus.Fields{"workerID": workerID, "panic": r})
					cancel()
				}
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case meta, ok := <-filesChan:
					if !ok {
						return
					}

					if !a.workerShouldContinue(ctx, w) {
						return
					}

					absFilePath, fileResults := a.processFile(ctx, meta, job.pattern, *job.req, searchState, &searchCancelled, cancel)
					if absFilePath == "" {
						continue
					}

					// Send results and emit progress
					a.emitFileResults(ctx, fileResults, resultsChan, w)
					// emitFileProgress self-detects the last file from the
					// post-increment count, so no racy caller-side isLast here.
					a.emitFileProgress(searchState, job.total, absFilePath)
				}
			}
		}()
	}

	// Send files to channel
	go func() {
		// A panicking send must not wedge workers on an unclosed channel.
		defer func() {
			if r := recover(); r != nil {
				a.logError("Search feeder panicked", nil, logrus.Fields{"panic": r})
			}
		}()
		defer close(filesChan)
		for _, file := range job.files {
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
func (a *App) workerShouldContinue(
	ctx context.Context,
	w searchWorker,
) bool {
	if int(atomic.LoadInt32(&w.state.resultsCount)) >= w.maxResults {
		w.cancelOnce()
		return false
	}
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// readFileBounded opens path, re-stats it, and reads it bounded by maxSize.
//
// Re-statting before reading matters: the file could have been replaced or
// grown past MaxFileSize since collection (TOCTOU). Opening first (which
// follows symlinks) and checking the real size also catches a file swapped
// for a symlink to a larger file. The exact-size read is one allocation
// instead of io.ReadAll's doubling growth from a 512B start; if the file
// grew between Stat and Read, ReadFull returns ErrUnexpectedEOF and the
// remainder is read to preserve prior behavior.
func readFileBounded(absFilePath string, maxSize int64) ([]byte, error) {
	f, err := os.Open(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", absFilePath, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", absFilePath, err)
	}
	if info.Size() > maxSize {
		return nil, fmt.Errorf("file size %d exceeds max %d", info.Size(), maxSize)
	}
	content := make([]byte, info.Size())
	if _, err := io.ReadFull(f, content); err != nil {
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("read %s: %w", absFilePath, err)
		}
		rest, rerr := io.ReadAll(f)
		if rerr != nil {
			return nil, fmt.Errorf("read remainder %s: %w", absFilePath, rerr)
		}
		content = append(content, rest...)
	}
	return content, nil
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
//
// req stays by-value (not *SearchRequest): tests call processFile
// positionally with a SearchRequest literal.
func (a *App) processFile(
	ctx context.Context,
	meta fileMeta,
	pattern *regexp.Regexp,
	req SearchRequest,
	searchState *SearchState,
	searchCancelled *int32,
	cancel context.CancelFunc,
) (string, []SearchResult) {
	absFilePath := meta.absPath

	// Respect the request's context window (0 = unset -> defaultContextLines),
	// clamped to maxContextLines so request payloads stay bounded.
	ctxLines := searchContextLines(req.ContextLines)

	w := searchWorker{
		state:      searchState,
		cancelled:  searchCancelled,
		cancel:     cancel,
		maxResults: req.MaxResults,
	}

	if meta.size > int64(streamingThreshold) {
		remaining := req.MaxResults - int(atomic.LoadInt32(&searchState.resultsCount))
		results, procErr := a.processFileLineByLine(ctx, absFilePath, pattern, remaining, ctxLines)
		if procErr != nil {
			searchState.recordFailure(absFilePath)
			a.logWarn("Error processing file with streaming", logrus.Fields{"filePath": absFilePath, "error": procErr.Error()})
			return "", nil
		}
		return absFilePath, results
	}

	content, err := readFileBounded(absFilePath, req.MaxFileSize)
	if err != nil {
		searchState.recordFailure(absFilePath)
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
	// No pre-size: benchmarks show the common file has 0-1 matches, so
	// make(0, min(8,len)) wastes ~1KB (8×~128B SearchResult) on every
	// non-matching file. Append grows only on actual hits.
	var fileResults []SearchResult

	for i, line := range lines {
		if !a.workerShouldContinue(ctx, w) {
			break
		}

		// Single Find (not Match+Find): the old code ran the regexp engine
		// twice per HIT. The trimmed re-find keeps MatchedText ⊆ Content
		// (offsets on trimmed, not raw); non-matches run one Find and stop.
		loc := pattern.FindIndex(line)
		if loc != nil {
			trimmed := bytes.TrimSpace(line)
			matchedText := pattern.Find(trimmed)
			if matchedText == nil {
				matchedText = trimmed
			}
			contextBefore := safeContextLinesBytes(lines, i-ctxLines, i)
			contextAfter := safeContextLinesBytes(lines, i+1, i+1+ctxLines)

			fileResults = append(fileResults, SearchResult{
				FilePath:      absFilePath,
				LineNum:       i + 1,
				Content:       string(trimmed),
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
func (a *App) emitFileResults(
	ctx context.Context,
	fileResults []SearchResult,
	resultsChan chan<- SearchResult,
	w searchWorker,
) {
	for _, result := range fileResults {
		if int(atomic.LoadInt32(&w.state.resultsCount)) >= w.maxResults {
			w.cancelOnce()
			return
		}

		select {
		case resultsChan <- result:
			// Ownership of result transfers to the drain goroutine on send.
			newCount := atomic.AddInt32(&w.state.resultsCount, 1)
			if int(newCount) >= w.maxResults {
				w.cancelOnce()
			}
		case <-ctx.Done():
			return
		}
	}
}

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

// resultBatcher accumulates results drained from the worker channel and
// flushes them to the frontend as "search-results" events so the UI renders
// progressively instead of waiting for the whole search.
//
// Batches carry worker-completion order, NOT the final sorted order —
// SearchWithProgress's return value remains the authoritative, deterministically
// sorted result set. The frontend appends batches for immediate feedback and
// then replaces the list with the returned slice when the search completes.
//
// Used from the single drain goroutine only, so it needs no locking.
type resultBatcher struct {
	app       *App
	seq       int
	pending   []SearchResult
	lastFlush time.Time
}

func newResultBatcher(a *App) *resultBatcher {
	return &resultBatcher{app: a, lastFlush: time.Now()}
}

// add queues one result, flushing when the batch fills or the flush interval
// has elapsed.
func (b *resultBatcher) add(result SearchResult) {
	b.pending = append(b.pending, result)
	if len(b.pending) >= resultBatchSize || time.Since(b.lastFlush) >= progressEmitInterval {
		b.flush()
	}
}

// flush emits the queued results and starts a new batch. No-op when empty.
//
// pending is set to nil rather than truncated: the slice is handed to the
// event payload, so reusing its backing array could mutate a batch that is
// still being serialized.
func (b *resultBatcher) flush() {
	if len(b.pending) == 0 {
		return
	}
	b.seq++
	b.app.safeEmitEvent("search-results", &SearchResultBatch{Seq: b.seq, Results: b.pending})
	b.pending = nil
	b.lastFlush = time.Now()
}
