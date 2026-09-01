package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// csvSafeCell: leading-space bypass
// ---------------------------------------------------------------------------

func TestCsvSafeCell_LeadingSpaceBypass(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"space before =", " =2+2", "' =2+2"},
		{"two spaces before =", "  =cmd|' /C calc'!A0", "'  =cmd|' /C calc'!A0"},
		{"space then tab before =", " \t=2+2", "' \t=2+2"},
		{"space before +", " +2+2", "' +2+2"},
		{"space before -", " -2+2", "' -2+2"},
		{"space before @", " @malicious", "' @malicious"},
		{"tab alone is formula", "\t=2+2", "'\t=2+2"},
		{"space before normal", " hello", " hello"},
		{"no leading space still blocked", "=2+2", "'=2+2"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := csvSafeCell(tt.in); got != tt.want {
				t.Errorf("csvSafeCell(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MaxResults cap
// ---------------------------------------------------------------------------

func TestValidateAndSetDefaults_MaxResultsCap(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	t.Run("over cap rejected", func(t *testing.T) {
		req := SearchRequest{Directory: dir, Query: "x", MaxResults: 10001}
		_, err := app.validateAndSetDefaults(req)
		if err == nil {
			t.Fatal("expected error for MaxResults > cap, got nil")
		}
		if !strings.Contains(err.Error(), "maxResults") {
			t.Errorf("error should mention maxResults, got %q", err.Error())
		}
	})

	t.Run("at cap allowed", func(t *testing.T) {
		req := SearchRequest{Directory: dir, Query: "x", MaxResults: 10000}
		got, err := app.validateAndSetDefaults(req)
		if err != nil {
			t.Fatalf("unexpected error at cap: %v", err)
		}
		if got.MaxResults != 10000 {
			t.Errorf("MaxResults = %d, want 10000", got.MaxResults)
		}
	})

	t.Run("default still 1000", func(t *testing.T) {
		req := SearchRequest{Directory: dir, Query: "x", MaxResults: 0}
		got, err := app.validateAndSetDefaults(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.MaxResults != 1000 {
			t.Errorf("default MaxResults = %d, want 1000", got.MaxResults)
		}
	})
}

// ---------------------------------------------------------------------------
// Query length cap
// ---------------------------------------------------------------------------

func TestValidateAndSetDefaults_QueryLengthCap(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	t.Run("over cap rejected", func(t *testing.T) {
		req := SearchRequest{Directory: dir, Query: strings.Repeat("x", maxQueryLength+1)}
		if _, err := app.validateAndSetDefaults(req); err == nil {
			t.Fatal("expected error for query longer than cap, got nil")
		}
	})

	t.Run("at cap allowed", func(t *testing.T) {
		req := SearchRequest{Directory: dir, Query: strings.Repeat("x", maxQueryLength)}
		if _, err := app.validateAndSetDefaults(req); err != nil {
			t.Fatalf("unexpected error at cap: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Protected directory subtrees
// ---------------------------------------------------------------------------

func TestValidateAndSetDefaults_ProtectedSubtree(t *testing.T) {
	app := NewApp()
	// Use a temp dir that we can create to pass the os.Stat check, then
	// test the protected-path logic via a real path that is a subtree.
	// We test with a synthetic protected list by checking the error message
	// for a known protected prefix.
	// On linux, /etc is protected, so /etc/ssh should be rejected.
	if _, err := os.Stat("/etc"); err != nil {
		t.Skip("no /etc on this platform")
	}
	t.Run("subtree blocked", func(t *testing.T) {
		// Create a temp dir that mimics a protected subtree by using the
		// actual /etc path (exists, will hit protected check before walk).
		req := SearchRequest{Directory: "/etc/ssh", Query: "x", MaxResults: 10}
		_, err := app.validateAndSetDefaults(req)
		if err == nil {
			t.Fatal("expected error for protected subtree /etc/ssh, got nil")
		}
		if !strings.Contains(err.Error(), "protected") {
			t.Errorf("error should mention protected, got %q", err.Error())
		}
	})

	t.Run("similar prefix not blocked", func(t *testing.T) {
		// /etc-backup should NOT be blocked (separator check).
		dir := filepath.Join(t.TempDir(), "etc-backup")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		// This path is not /etc nor /etc/*, so it should pass the protected check.
		// It will fail only if the directory doesn't exist, which it does.
		req := SearchRequest{Directory: dir, Query: "x", MaxResults: 10}
		_, err := app.validateAndSetDefaults(req)
		if err != nil {
			t.Fatalf("unexpected error for /etc-backup-like path: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Symlink handling in walkDirectoryTree
// ---------------------------------------------------------------------------

func TestWalkDirectoryTree_SkipsSymlink(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	// Create a real file and a symlink to it.
	realFile := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(realFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	linkFile := filepath.Join(dir, "link.txt")
	if err := os.Symlink(realFile, linkFile); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	req := SearchRequest{
		Directory:        dir,
		Query:            "hello",
		MaxResults:       10,
		MaxFileSize:      10 * 1024 * 1024,
		AllowedFileTypes: []string{},
	}
	candidates, binaryCandidates, _, err := app.walkDirectoryTree(context.Background(), req, false)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
	// Symlink should be skipped, so only real.txt should be in candidates.
	for _, m := range candidates {
		if strings.Contains(m.absPath, "link.txt") {
			t.Errorf("symlink %q should have been skipped", m.absPath)
		}
	}
	for _, m := range binaryCandidates {
		if strings.Contains(m.absPath, "link.txt") {
			t.Errorf("symlink %q should have been skipped (binary candidates)", m.absPath)
		}
	}
	// At least real.txt should be found (known text extension).
	if len(candidates) == 0 {
		t.Error("expected at least one candidate (real.txt)")
	}
}

func TestWalkDirectoryTree_SymlinkToDirNotFollowed(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	// Symlink to subdir — WalkDir should not descend into it as a separate tree.
	linkDir := filepath.Join(dir, "linkdir")
	if err := os.Symlink(subdir, linkDir); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	req := SearchRequest{
		Directory:   dir,
		Query:       "hello",
		MaxResults:  10,
		MaxFileSize: 10 * 1024 * 1024,
	}
	candidates, _, _, err := app.walkDirectoryTree(context.Background(), req, false)
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
	// Should have exactly one file (sub/a.txt), not duplicated via symlink.
	if len(candidates) != 1 {
		t.Errorf("expected 1 candidate, got %d", len(candidates))
	}
}
