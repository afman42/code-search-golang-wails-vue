package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// TestSearchResultsSorted verifies SearchWithProgress returns results ordered
// by file path then line number, regardless of worker completion order.
func TestSearchResultsSorted(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	files := map[string]string{
		"zeta.txt":  "apple second\nfirst\napple third\napple first\n",
		"alpha.txt": "apple first\napple second\napple third\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	req := SearchRequest{Directory: tempDir, Query: "apple", MaxResults: 1000}

	for i := 0; i < 3; i++ {
		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress error: %v", err)
		}
		if len(results) != 6 {
			t.Fatalf("expected 6 results, got %d", len(results))
		}
		if !sort.SliceIsSorted(results, func(i, j int) bool {
			if results[i].FilePath != results[j].FilePath {
				return results[i].FilePath < results[j].FilePath
			}
			return results[i].LineNum < results[j].LineNum
		}) {
			t.Fatalf("results not sorted by path/line: %+v", results)
		}
	}
}

// TestSearchWithProgressCancelled drives the real cancel flow: start a search
// in a goroutine, cancel it once the backend has registered the active search
// context (i.e. collection finished and the worker pool is running), then
// assert the search returns EMPTY results rather than a partial "completed"
// set that the frontend would otherwise repopulate.
func TestSearchWithProgressCancelled(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Large enough fixture that processing outlives our cancel poll loop.
	for i := 0; i < 120; i++ {
		var sb []byte
		for j := 0; j < 200; j++ {
			sb = append(sb, []byte(fmt.Sprintf("needle-%d segment-%d preamble text here\n", i, j))...)
		}
		if err := os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("f%03d.txt", i)), sb, 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	req := SearchRequest{Directory: tempDir, Query: "needle", MaxResults: 1000}

	resultsCh := make(chan []SearchResult, 1)
	errCh := make(chan error, 1)
	go func() {
		r, err := app.SearchWithProgress(req)
		resultsCh <- r
		errCh <- err
	}()

	// Wait (bounded) for the backend to register the active search context,
	// then cancel. cancelActiveSearch only returns true while the search is
	// in flight, so this guarantees the cancel lands mid-processing.
	cancelled := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if app.cancelActiveSearch() {
			cancelled = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !cancelled {
		t.Fatal("search finished before cancel could land — fixture too small")
	}

	result := <-resultsCh
	if err := <-errCh; err != nil {
		t.Fatalf("cancelled search error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("cancelled search returned %d non-empty results; want empty", len(result))
	}
}
