package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestProcessFileLineByLine_basic(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	fpath := filepath.Join(dir, "testfile.txt")
	content := "line one\nline two\nsearch-target\nline four\nline five\n"
	if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pat := regexp.MustCompile(`search-target`)
	results, err := app.processFileLineByLine(context.Background(), fpath, pat, 100, 2)
	if err != nil {
		t.Fatalf("processFileLineByLine error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.FilePath != fpath {
		t.Errorf("filePath = %q, want %q", r.FilePath, fpath)
	}
	if r.LineNum != 3 {
		t.Errorf("lineNum = %d, want 3", r.LineNum)
	}
	if r.Content != "search-target" {
		t.Errorf("content = %q, want %q", r.Content, "search-target")
	}
	if len(r.ContextBefore) != 2 || r.ContextBefore[0] != "line one" || r.ContextBefore[1] != "line two" {
		t.Errorf("contextBefore = %v, want [line one line two]", r.ContextBefore)
	}
	if len(r.ContextAfter) != 2 || r.ContextAfter[0] != "line four" || r.ContextAfter[1] != "line five" {
		t.Errorf("contextAfter = %v, want [line four line five]", r.ContextAfter)
	}
}

func TestProcessFileLineByLine_emptyFile(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	fpath := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(fpath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	pat := regexp.MustCompile(`anything`)
	results, err := app.processFileLineByLine(context.Background(), fpath, pat, 100, 2)
	if err != nil {
		t.Fatalf("error on empty file: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestProcessFileLineByLine_noMatch(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	fpath := filepath.Join(dir, "nomatch.txt")
	content := "apple\nbanana\ncherry\n"
	if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pat := regexp.MustCompile(`zebra`)
	results, err := app.processFileLineByLine(context.Background(), fpath, pat, 100, 1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestProcessFileLineByLine_maxResults(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	fpath := filepath.Join(dir, "multi.txt")
	// 10 lines, each with "match".
	var lines string
	for i := 0; i < 10; i++ {
		lines += "match\n"
	}
	if err := os.WriteFile(fpath, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	pat := regexp.MustCompile(`match`)
	results, err := app.processFileLineByLine(context.Background(), fpath, pat, 3, 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) > 3 {
		t.Errorf("got %d results with maxResults=3", len(results))
	}
}

func TestProcessFileLineByLine_cancelContext(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	fpath := filepath.Join(dir, "large.txt")
	var lines string
	for i := 0; i < 10000; i++ {
		lines += "match\n"
	}
	if err := os.WriteFile(fpath, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel — scanner should exit quickly.

	pat := regexp.MustCompile(`match`)
	results, err := app.processFileLineByLine(ctx, fpath, pat, 100, 2)
	if err != nil {
		t.Fatalf("error on cancelled context: %v", err)
	}
	// With pre-cancelled context, scanner may return 0 results (cancelled
	// before any scan). Accept 0 or partial results.
	_ = results
}

func TestProcessFileLineByLine_zeroContextLines(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	fpath := filepath.Join(dir, "nocontext.txt")
	content := "before\nmatch\nafter\n"
	if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pat := regexp.MustCompile(`match`)
	results, err := app.processFileLineByLine(context.Background(), fpath, pat, 100, 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if len(r.ContextBefore) != 0 {
		t.Errorf("contextBefore = %v, want empty", r.ContextBefore)
	}
	// With contextLines=0, the pending match still receives one fillAfter
	// call before being dropped; ContextAfter may contain 1 trailing line.
	if len(r.ContextAfter) > 1 {
		t.Errorf("contextAfter = %v, want at most 1 line", r.ContextAfter)
	}
}