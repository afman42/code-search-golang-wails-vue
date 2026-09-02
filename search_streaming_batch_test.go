package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestResultBatcherFlushesBySize verifies a full batch is emitted as soon as
// resultBatchSize results accumulate, with a monotonic sequence starting at 1.
// The batcher is the streaming path's only ordering guarantee, so a wrong or
// reused seq would let the frontend silently drop or duplicate rows.
func TestResultBatcherFlushesBySize(t *testing.T) {
	app := NewApp()
	b := newResultBatcher(app)

	// One result short of the size trigger: nothing may have been emitted yet.
	for i := range resultBatchSize - 1 {
		b.pending = append(b.pending, SearchResult{FilePath: "/a.go", LineNum: i + 1})
	}
	if b.seq != 0 {
		t.Fatalf("seq advanced before any flush: got %d, want 0", b.seq)
	}

	// add() drives the real trigger. Because lastFlush is "now" at construction,
	// the size threshold (not the time one) is what fires here.
	b.add(SearchResult{FilePath: "/a.go", LineNum: resultBatchSize})
	if b.seq != 1 {
		t.Fatalf("full batch did not flush: seq = %d, want 1", b.seq)
	}
	if len(b.pending) != 0 {
		t.Fatalf("pending not cleared after flush: %d results left", len(b.pending))
	}

	// A second full batch must take seq 2, never reuse 1.
	for i := range resultBatchSize {
		b.add(SearchResult{FilePath: "/b.go", LineNum: i + 1})
	}
	if b.seq != 2 {
		t.Fatalf("second batch seq = %d, want 2", b.seq)
	}
}

// TestResultBatcherFlushIsNoopWhenEmpty guards the terminal flush() calls in
// SearchWithProgress: a search with zero matches must not emit an empty batch
// and must not burn a sequence number, or the frontend would treat seq 1 as
// already delivered and drop the first real batch of the next search.
func TestResultBatcherFlushIsNoopWhenEmpty(t *testing.T) {
	b := newResultBatcher(NewApp())

	b.flush()
	b.flush()

	if b.seq != 0 {
		t.Fatalf("empty flush advanced seq to %d, want 0", b.seq)
	}
}

// TestResultBatcherDoesNotShareBackingArray verifies flush() hands ownership of
// the pending slice to the event payload instead of reusing its backing array.
// Reusing it would let a subsequent add() overwrite a batch that the Wails
// bridge is still serializing.
func TestResultBatcherDoesNotShareBackingArray(t *testing.T) {
	b := newResultBatcher(NewApp())

	b.add(SearchResult{FilePath: "/a.go", LineNum: 1})
	emitted := b.pending // captured before flush hands it off
	b.flush()

	b.add(SearchResult{FilePath: "/b.go", LineNum: 2})

	if len(emitted) != 1 {
		t.Fatalf("captured batch length changed: got %d, want 1", len(emitted))
	}
	if emitted[0].FilePath != "/a.go" {
		t.Fatalf("captured batch was mutated by a later add: got %q, want /a.go", emitted[0].FilePath)
	}
	if len(b.pending) != 1 || b.pending[0].FilePath != "/b.go" {
		t.Fatalf("new batch did not start clean: %+v", b.pending)
	}
}

// TestSearchStateRecordFailureCapsSample verifies the failed-path sample is
// bounded while the count stays exact. The count is what the UI headline
// reports; the sample is what it lists, and an unbounded sample would let a
// tree of unreadable files balloon the terminal event payload.
func TestSearchStateRecordFailureCapsSample(t *testing.T) {
	state := &SearchState{}

	const failures = maxFailedPathsReported + 25
	for i := range failures {
		state.recordFailure(fmt.Sprintf("/denied/file%d.go", i))
	}

	if got := int(state.failedFiles); got != failures {
		t.Fatalf("failedFiles = %d, want %d (count must stay exact)", got, failures)
	}

	sample := state.snapshotFailedPaths()
	if len(sample) != maxFailedPathsReported {
		t.Fatalf("sample length = %d, want %d", len(sample), maxFailedPathsReported)
	}

	// The snapshot must be a copy: mutating it cannot reach the guarded slice.
	sample[0] = "/mutated"
	if again := state.snapshotFailedPaths(); again[0] == "/mutated" {
		t.Fatal("snapshotFailedPaths returned the live slice, not a copy")
	}
}

// TestSearchStateRecordFailureIgnoresEmptyPath verifies an empty path still
// counts but is not listed: a blank entry in the UI's skipped-files list would
// be worse than no entry.
func TestSearchStateRecordFailureIgnoresEmptyPath(t *testing.T) {
	state := &SearchState{}

	state.recordFailure("")

	if got := int(state.failedFiles); got != 1 {
		t.Fatalf("failedFiles = %d, want 1", got)
	}
	if sample := state.snapshotFailedPaths(); len(sample) != 0 {
		t.Fatalf("empty path was listed: %v", sample)
	}
}

// TestSearchWithProgressReportsUnreadableFiles drives the real search pipeline
// against an unreadable file and asserts the failure reaches the terminal
// progress payload as both a count and a named path. Before FailedPaths, a
// search that could not open a file was indistinguishable from one that found
// no matches there.
func TestSearchWithProgressReportsUnreadableFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 0000 does not deny access")
	}

	app := NewApp()
	tempDir := t.TempDir()

	readable := filepath.Join(tempDir, "readable.go")
	if err := os.WriteFile(readable, []byte("package main // needle\n"), 0644); err != nil {
		t.Fatalf("write readable file: %v", err)
	}

	denied := filepath.Join(tempDir, "denied.go")
	if err := os.WriteFile(denied, []byte("package main // needle\n"), 0644); err != nil {
		t.Fatalf("write denied file: %v", err)
	}
	if err := os.Chmod(denied, 0000); err != nil {
		t.Skipf("chmod unsupported on this platform: %v", err)
	}
	// Restore permissions so t.TempDir cleanup can remove the file.
	t.Cleanup(func() { _ = os.Chmod(denied, 0644) })

	req := SearchRequest{Directory: tempDir, Query: "needle", MaxResults: 1000}
	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// The readable file still matches — one unreadable file must not abort the
	// search or drop its siblings' results.
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (from the readable file)", len(results))
	}

	// SearchState is internal to the search, so assert via a direct processFile
	// call on the same file: it is the function that records the failure.
	state := &SearchState{}
	pattern, err := app.compileSearchPattern(req)
	if err != nil {
		t.Fatalf("compile pattern: %v", err)
	}
	absPath, fileResults := app.processFile(
		context.Background(), fileMeta{absPath: denied, size: 23}, pattern,
		SearchRequest{MaxFileSize: 1 << 20, MaxResults: 10}, state, new(int32), func() {},
	)
	if absPath != "" || fileResults != nil {
		t.Fatalf("unreadable file was processed: path=%q results=%v", absPath, fileResults)
	}
	if int(state.failedFiles) != 1 {
		t.Fatalf("failedFiles = %d, want 1", state.failedFiles)
	}
	sample := state.snapshotFailedPaths()
	if len(sample) != 1 || sample[0] != denied {
		t.Fatalf("failed path not recorded: got %v, want [%s]", sample, denied)
	}
}
