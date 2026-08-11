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

	var results []SearchResult
	scanner := bufio.NewScanner(file)

	// Set a larger buffer for very long lines (1MB)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	// prev holds up to contextLines preceding lines for ContextBefore.
	prev := make([]string, 0, contextLines)
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
			pending = append(pending, pendingMatch{idx: len(results) - 1, remaining: contextLines})
		}

		// Advance the rolling buffer of preceding lines.
		prev = append(prev, line)
		if len(prev) > contextLines {
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
