package main

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// processFileLineByLine processes a file line by line to avoid loading large files into memory.
// Binary detection is already performed upstream in collectFilesToProcess.
//
// Context lines (up to contextLines before and after each match) are captured
// the same way as the small-file path: a rolling buffer holds recent lines for
// ContextBefore, and matches stay "pending" until enough following lines are read
// to fill ContextAfter.
func (a *App) processFileLineByLine(ctx context.Context, filePath string, pattern *regexp.Regexp, maxResults int, contextLines int) ([]SearchResult, error) {
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

	st := newScanState(contextLines)
	scanner := bufio.NewScanner(file)
	// Shared 16MB token cap (maxScanLineSize in search_context.go): the 64KB
	// default aborts the whole file on longer lines with ErrTooLong.
	scanner.Buffer(nil, maxScanLineSize)

	lineNum := 1
	linesProcessed := 0
	for scanner.Scan() {
		line := scanner.Text()

		// Fill ContextAfter for matches found on earlier lines.
		st.fillAfter(line)

		// Record a new match (unless we've already hit the result limit).
		if len(st.results) < maxResults && pattern.MatchString(line) {
			st.record(SearchResult{
				FilePath:      filePath,
				LineNum:       lineNum,
				Content:       strings.TrimSpace(line),
				MatchedText:   pattern.FindString(line),
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
		a.logError("Error during line-by-line scanning", err, logrus.Fields{
			"filePath": filePath,
		})
		return nil, err
	}

	a.logDebug("Completed line-by-line file processing", logrus.Fields{
		"filePath":       filePath,
		"resultsFound":   len(st.results),
		"linesProcessed": linesProcessed,
	})
	return st.results, nil
}

// streamingThreshold is the file size (in bytes) above which files are processed
// line-by-line instead of being read entirely into memory.
const streamingThreshold = 1024 * 1024 // 1MB
