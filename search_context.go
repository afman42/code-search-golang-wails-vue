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
