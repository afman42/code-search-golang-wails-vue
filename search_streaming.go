package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// streamingThreshold is the file size (in bytes) above which files are processed
// line-by-line instead of being read entirely into memory.
const streamingThreshold = 1024 * 1024 // 1MB

// processFileLineByLine processes a file line by line to avoid loading large files into memory.
// Binary detection is already performed upstream in collectFilesToProcess.
//
// Context lines (up to contextLines before and after each match) are captured
// the same way as the small-file path: a rolling buffer holds recent lines for
// ContextBefore, and matches stay "pending" until enough following lines are read
// to fill ContextAfter.
func (a *App) processFileLineByLine(
	ctx context.Context,
	filePath string,
	pattern *regexp.Regexp,
	maxResults, contextLines int, // grouped search limits
) ([]SearchResult, error) {
	a.logDebug("Starting line-by-line file processing", logrus.Fields{
		"filePath":   filePath,
		"maxResults": maxResults,
	})

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer file.Close()

	st := newScanState(contextLines)
	scanner := bufio.NewScanner(file)
	// Shared 16MB token cap (maxScanLineSize in search_context.go): the 64KB
	// default aborts the whole file on longer lines with ErrTooLong.
	// Initial buffer from the shared pool (see scanBufPool): Scanner grows
	// it only when a line needs it.
	scanBufPtr := scanBufPool.Get().(*[]byte)
	defer scanBufPool.Put(scanBufPtr)
	scanner.Buffer((*scanBufPtr)[:0], maxScanLineSize)

	lineNum := 1
	var linesProcessed int
	for scanner.Scan() {
		// Bytes() + single FindIndex: Text() copies every line AND the old
		// code ran the regexp twice per hit (MatchString then FindString).
		// FindIndex on the raw buffer does match+locate in one engine run;
		// strings are materialized only below.
		raw := scanner.Bytes()
		loc := pattern.FindIndex(raw)
		matched := loc != nil && len(st.results) < maxResults

		// Materialize once per line: fillAfter/advance store the UNTRIMMED
		// line (matches the old scanner.Text() semantics — ContextBefore /
		// After are untrimmed, only Content is trimmed). Skipped entirely
		// when contextLines==0, no match, and nobody awaits trailing ctx.
		var line string
		needLine := contextLines > 0 || matched || len(st.pending) > 0
		if needLine {
			line = string(raw)
		}

		// Fill ContextAfter for matches found on earlier lines.
		if len(st.pending) > 0 {
			st.fillAfter(line)
		}

		// Record a new match (unless we've already hit the result limit).
		if matched {
			trimmed := strings.TrimSpace(line)
			// Locate the match within the TRIMMED content for MatchedText:
			// Find on trimmed, not raw, so offsets line up with Content.
			// (Old code found on untrimmed; trimmed-find is the small-file
			// equivalent and keeps MatchedText ⊆ Content.)
			matchLoc := pattern.FindStringIndex(trimmed)
			matchedText := trimmed
			if matchLoc != nil {
				matchedText = trimmed[matchLoc[0]:matchLoc[1]]
			}
			st.record(SearchResult{
				FilePath:      filePath,
				LineNum:       lineNum,
				Content:       trimmed,
				MatchedText:   matchedText,
				ContextBefore: st.before(),
				ContextAfter:  []string{},
			}, contextLines)
		}

		// Advance the rolling buffer of preceding lines.
		st.advance(line, contextLines)

		lineNum++
		linesProcessed++

		// Stop once the result limit is reached and every match has its trailing context.
		if st.done(maxResults) {
			break
		}

		if linesProcessed%100 == 0 {
			select {
			case <-ctx.Done():
				a.logDebug("Line-by-line processing cancelled due to context", logrus.Fields{
					"filePath":       filePath,
					"linesProcessed": linesProcessed,
					"resultsFound":   len(st.results),
				})
				return st.results, nil
			default:
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	a.logDebug("Completed line-by-line file processing", logrus.Fields{
		"filePath":       filePath,
		"resultsFound":   len(st.results),
		"linesProcessed": linesProcessed,
	})
	return st.results, nil
}
