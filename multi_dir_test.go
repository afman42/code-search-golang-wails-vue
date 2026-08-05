package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMultiDirectorySearch verifies that SearchWithProgress collects files
// from the primary Directory AND any AdditionalDirectories.
func TestMultiDirectorySearch(t *testing.T) {
	app := NewApp()
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	os.WriteFile(filepath.Join(dir1, "a.go"), []byte("package main\nfunc foo() { needle }\n"), 0o644)
	os.WriteFile(filepath.Join(dir2, "b.go"), []byte("package main\nfunc bar() { needle }\n"), 0o644)

	req := SearchRequest{
		Directory:   dir1,
		Directories: []string{dir2},
		Query:       "needle",
	}

	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Fatalf("SearchWithProgress error: %v", err)
	}

	// Should find matches in both directories.
	if len(results) != 2 {
		t.Fatalf("expected 2 results (one per dir), got %d", len(results))
	}

	// Verify results came from both directories.
	seenDirs := map[string]bool{}
	for _, r := range results {
		seenDirs[filepath.Dir(r.FilePath)] = true
	}
	if len(seenDirs) != 2 {
		t.Fatalf("expected results from 2 directories, got %d: %v", len(seenDirs), seenDirs)
	}
}

// TestMultiDirectorySearchDedup verifies that duplicate directories are
// deduplicated (a directory listed in both Directory and Directories is
// only searched once).
func TestMultiDirectorySearchDedup(t *testing.T) {
	app := NewApp()
	dir1 := t.TempDir()

	os.WriteFile(filepath.Join(dir1, "a.go"), []byte("package main\nfunc foo() { needle }\n"), 0o644)

	req := SearchRequest{
		Directory:   dir1,
		Directories: []string{dir1}, // same as primary
		Query:       "needle",
	}

	results, err := app.SearchWithProgress(req)
	if err != nil {
		t.Fatalf("SearchWithProgress error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (deduped), got %d", len(results))
	}
}
