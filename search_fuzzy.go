package main

// Fuzzy search support (phase 2 of SearchWithProgress).
//
// When SearchRequest.FuzzySearch is set (and UseRegex is not), the engine
// first collects exact matches exactly as before, then performs a second pass
// over the same collected files to find "near-miss" candidates: lines that do
// NOT match the exact pattern but align with the query under the same
// sliding-window similarity scoring the frontend applies (see
// frontend/src/utils/fuzzyMatch.ts). The frontend's useSearch post-processing
// re-scores these candidates, flags them with the fuzzy badge, and drops any
// it cannot score — so the backend may safely over-deliver slightly.
//
// Fuzzy matching is case-insensitive regardless of CaseSensitive, mirroring
// the frontend's scoring (findFuzzyMatches always lowercases). With
// CaseSensitive=true this means case-variant occurrences surface as plain
// results (the frontend does not badge lines that contain the query
// case-insensitively) while true near-misses get the badge.

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// fuzzySimilarityThreshold mirrors SLIDING_WINDOW_SIMILARITY_THRESHOLD in
// frontend/src/utils/fuzzyMatch.ts: a window qualifies when at least 60% of
// its characters align positionally with the query.
const fuzzySimilarityThreshold = 0.6

// fuzzyMaxLineLen mirrors MAX_TEXT_LENGTH_FOR_FUZZY_SEARCH in fuzzyMatch.ts:
// the frontend discards texts longer than this, so the backend never emits
// candidates for them.
const fuzzyMaxLineLen = 50000

// fuzzyThreshold returns the minimum number of positionally-aligned character
// matches a len(query)-wide window needs to qualify as a fuzzy candidate.
// Mirrors Math.floor(query.length * 0.6), with a >=1 floor: for a 1-char
// query the floor is 0, which would match every line and flood the result
// set with garbage candidates.
func fuzzyThreshold(queryLen int) int {
	t := int(float64(queryLen) * fuzzySimilarityThreshold)
	if t < 1 {
		t = 1
	}
	return t
}

// fuzzyBestWindow slides a len(queryLower)-wide window over textLower and
// returns the best positional-match count and its window start (earliest on
// ties), or count -1 when no window reaches the threshold. Both inputs must
// already be case-folded. Mirrors the per-window scoring in findFuzzyMatches.
func fuzzyBestWindow(textLower, queryLower []byte, threshold int) (bestCount, bestStart int) {
	if len(queryLower) == 0 || len(textLower) < len(queryLower) || len(textLower) > fuzzyMaxLineLen {
		return -1, 0
	}
	bestCount = -1
	bestStart = 0
	for pos := range len(textLower) - len(queryLower) + 1 {
		count := 0
		for i, q := range queryLower {
			if textLower[pos+i] == q {
				count++
			}
		}
		if count >= threshold && count > bestCount {
			bestCount = count
			bestStart = pos
			if count == len(queryLower) {
				break // a perfect window cannot be beaten
			}
		}
	}
	return bestCount, bestStart
}

// processFileFuzzy scans one file for fuzzy-only candidates: lines that do NOT
// match the exact pattern (already returned or capped by the primary pass) but
// contain a window similar enough to the query. Context lines are captured the
// same way as processFileLineByLine (rolling prev buffer + pending queue).
func (a *App) processFileFuzzy(ctx context.Context, meta fileMeta, pattern *regexp.Regexp, queryLower []byte, threshold, quota, ctxLines int) []SearchResult {
	file, err := os.Open(meta.absPath)
	if err != nil {
		a.logDebug("Skipping file in fuzzy pass", logrus.Fields{"filePath": meta.absPath, "error": err.Error()})
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Shared 16MB token cap (maxScanLineSize in search_context.go). Passing
	// nil lets Scanner start at its 4KB default and grow only when a line
	// needs it — a 64KB preallocation per file dominated allocation in
	// benchmarks (200 files → 12.8MB/op) while most lines are short.
	scanner.Buffer(nil, maxScanLineSize)

	st := newScanState(ctxLines)

	lineNum := 0
	for scanner.Scan() {
		raw := scanner.Bytes()
		lineNum++
		line := string(raw)

		// Fill ContextAfter for candidates found on earlier lines.
		st.fillAfter(line)

		if len(st.results) < quota && !pattern.Match(raw) {
			// The frontend scores r.content, which is the trimmed line —
			// match against the trimmed form so candidates line up exactly
			// with what useSearch sees.
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 {
				lower := bytes.ToLower([]byte(trimmed))
				count, start := fuzzyBestWindow(lower, queryLower, threshold)
				if count >= threshold {
					// Window offsets are only valid on the original string
					// when case-folding preserved byte length (always true
					// for ASCII; rare Unicode folds can resize). Fall back
					// to the whole line otherwise.
					matchedText := trimmed
					if len(lower) == len(trimmed) && start+len(queryLower) <= len(trimmed) {
						matchedText = trimmed[start : start+len(queryLower)]
					}
					st.record(SearchResult{
						FilePath:      meta.absPath,
						LineNum:       lineNum,
						Content:       trimmed,
						MatchedText:   matchedText,
						ContextBefore: st.before(),
						ContextAfter:  []string{},
					}, ctxLines)
				}
			}
		}

		st.advance(line, ctxLines)

		// Stop once the quota is reached and every candidate has its context.
		if st.done(quota) {
			break
		}

		if lineNum%100 == 0 {
			select {
			case <-ctx.Done():
				return st.results
			default:
			}
		}
	}

	if err := scanner.Err(); err != nil {
		a.logDebug("Fuzzy scan error", logrus.Fields{"filePath": meta.absPath, "error": err.Error()})
	}
	return st.results
}

// searchFuzzyCandidates runs the fuzzy phase-2 pass over the files already
// collected for the search (collection, excludes, size/type filters, and path
// validation were all applied in phase 1). It appends up to quota near-miss
// lines, skipping lines the exact pattern matches. Output is sorted by
// path/line so the merged result list stays deterministic regardless of
// worker completion order.
func (a *App) searchFuzzyCandidates(ctx context.Context, filesToProcess []fileMeta, req SearchRequest, pattern *regexp.Regexp, quota int) []SearchResult {
	if quota <= 0 || len(filesToProcess) == 0 {
		return nil
	}

	a.logInfo("Starting fuzzy candidate pass", logrus.Fields{
		"query":   req.Query,
		"files":   len(filesToProcess),
		"quota":   quota,
		"workers": numCPU(),
	})

	queryLower := bytes.ToLower([]byte(req.Query))
	threshold := fuzzyThreshold(len(queryLower))
	ctxLines := searchContextLines(req.ContextLines)

	numWorkers := numCPU()
	if len(filesToProcess) < numWorkers {
		numWorkers = len(filesToProcess)
	}

	filesChan := make(chan fileMeta, len(filesToProcess))
	resultsChan := make(chan SearchResult, 100)

	var found int32
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for meta := range filesChan {
				if int(atomic.LoadInt32(&found)) >= quota {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
				for _, r := range a.processFileFuzzy(ctx, meta, pattern, queryLower, threshold, quota, ctxLines) {
					if int(atomic.AddInt32(&found, 1)) > quota {
						return
					}
					select {
					case resultsChan <- r:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		defer close(filesChan)
		for _, f := range filesToProcess {
			select {
			case <-ctx.Done():
				return
			case filesChan <- f:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var results []SearchResult
	for r := range resultsChan {
		results = append(results, r)
	}

	// Deterministic order regardless of worker completion order; trim any
	// race overshoot to the strict quota afterwards.
	sort.Slice(results, func(i, j int) bool {
		if results[i].FilePath != results[j].FilePath {
			return results[i].FilePath < results[j].FilePath
		}
		return results[i].LineNum < results[j].LineNum
	})
	if len(results) > quota {
		results = results[:quota]
	}
	return results
}
