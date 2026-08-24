package main

import (
	"sync"
)

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

// defaultContextLines is the number of lines captured before and after each
// match when SearchRequest.ContextLines is left unset (0). It is used by both
// the streaming (line-by-line) path and the small-file path so results are
// consistent regardless of file size.
const defaultContextLines = 2

// maxContextLines caps the context window a request may ask for, so a single
// request cannot balloon result payloads with arbitrarily large context.
const maxContextLines = 10

// searchContextLines resolves and clamps a request's desired context window.
// 0 means "unset" and falls back to defaultContextLines, keeping the historical
// behavior for callers that construct a SearchRequest without the field.
func searchContextLines(n int) int {
	if n <= 0 {
		return defaultContextLines
	}
	if n > maxContextLines {
		return maxContextLines
	}
	return n
}

// safeContextLinesBytes returns lines[start:end] safe against out-of-bounds
// indices, for the small-file processing path that splits file content with
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

// maxScanLineSize is the bufio.Scanner token cap shared by both streaming
// file scanners. The default 64KB max aborts the whole file on any longer
// line (ErrTooLong), silently yielding zero results; minified JS/CSS/data
// files routinely contain single lines far beyond that. 16MB covers
// realistic minified files while still bounding memory per line.
const maxScanLineSize = 16 * 1024 * 1024 // 16MB

// pendingMatch tracks a recorded result (by index into scanState.results)
// that still needs contextLines more trailing lines of ContextAfter.
type pendingMatch struct {
	idx       int
	remaining int
}

// scanState holds the rolling context machinery shared by the two streaming
// file scanners (processFileLineByLine and processFileFuzzy): the results
// collected so far, a buffer of the last contextLines lines for
// ContextBefore, and the queue of matches still awaiting ContextAfter lines.
type scanState struct {
	results []SearchResult
	prev    []string
	pending []pendingMatch
}

func newScanState(contextLines int) *scanState {
	return &scanState{prev: make([]string, 0, contextLines)}
}

// before returns a copy of the preceding-lines buffer, so stored results
// don't alias the rolling window.
func (s *scanState) before() []string {
	out := make([]string, len(s.prev))
	copy(out, s.prev)
	return out
}

// fillAfter appends line as trailing context to every match still pending.
func (s *scanState) fillAfter(line string) {
	if len(s.pending) == 0 {
		return
	}
	stillPending := s.pending[:0]
	for _, p := range s.pending {
		s.results[p.idx].ContextAfter = append(s.results[p.idx].ContextAfter, line)
		p.remaining--
		if p.remaining > 0 {
			stillPending = append(stillPending, p)
		}
	}
	s.pending = stillPending
}

// record appends a result and queues it for contextLines more trailing lines.
func (s *scanState) record(res SearchResult, contextLines int) {
	s.results = append(s.results, res)
	s.pending = append(s.pending, pendingMatch{idx: len(s.results) - 1, remaining: contextLines})
}

// advance pushes line onto the rolling preceding-lines buffer.
func (s *scanState) advance(line string, contextLines int) {
	s.prev = append(s.prev, line)
	if len(s.prev) > contextLines {
		s.prev = s.prev[1:]
	}
}

// done reports whether the scanner can stop early: the limit is reached and
// every recorded match has its full trailing context.
func (s *scanState) done(limit int) bool {
	return len(s.results) >= limit && len(s.pending) == 0
}
