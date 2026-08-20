package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ReplaceInFiles
// ---------------------------------------------------------------------------

func TestReplaceInFilesDryRunWritesNothing(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	files := map[string]string{
		"a.txt": "hello world\nfoo bar\nhello again\n",
		"b.txt": "nope\nhello world\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Capture modtime before dry-run
	before := mustStat(t, filepath.Join(dir, "a.txt")).ModTime()

	result, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
		},
		Replacement: "goodbye",
		Apply:       false,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Dry-run reports correct diffs
	if result.FilesChanged != 2 {
		t.Errorf("expected FilesChanged=2, got %d", result.FilesChanged)
	}
	if result.LinesChanged != 3 {
		t.Errorf("expected LinesChanged=3, got %d", result.LinesChanged)
	}

	// No file was modified
	after := mustStat(t, filepath.Join(dir, "a.txt")).ModTime()
	if !after.Equal(before) {
		t.Error("dry-run should not modify files")
	}

	content, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(content) != "hello world\nfoo bar\nhello again\n" {
		t.Error("dry-run should not change file content")
	}
}

func TestReplaceInFilesApply(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello world\nfoo bar\nhello again\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
		},
		Replacement: "goodbye",
		Apply:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.FilesChanged != 1 {
		t.Errorf("expected FilesChanged=1, got %d", result.FilesChanged)
	}
	if result.LinesChanged != 2 {
		t.Errorf("expected LinesChanged=2, got %d", result.LinesChanged)
	}

	content, _ := os.ReadFile(path)
	expected := "goodbye world\nfoo bar\ngoodbye again\n"
	if string(content) != expected {
		t.Errorf("apply: expected %q, got %q", expected, string(content))
	}

	// Unmatched line untouched
	if !strings.Contains(string(content), "foo bar") {
		t.Error("apply: unmatched line should be preserved")
	}
}

func TestReplaceInFilesPreservesFileMode(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0755); err != nil {
		t.Fatal(err)
	}

	modeBefore := mustStat(t, path).Mode().Perm()

	_, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
		},
		Replacement: "goodbye",
		Apply:       true,
	})
	if err != nil {
		t.Fatal(err)
	}

	modeAfter := mustStat(t, path).Mode().Perm()
	if modeAfter != modeBefore {
		t.Errorf("expected file mode %o, got %o", modeBefore, modeAfter)
	}
}

func TestReplaceInFilesRejectsRegexMode(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
			UseRegex:  true,
		},
		Replacement: "goodbye",
		Apply:       true,
	})
	if err == nil {
		t.Fatal("expected error for regex mode")
	}
	if !strings.Contains(err.Error(), "literal-only") {
		t.Errorf("expected 'literal-only' error, got: %v", err)
	}
}

func TestReplaceInFilesEmptyQuery(t *testing.T) {
	app := NewApp()
	_, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Query: "",
		},
		Replacement: "x",
	})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestReplaceInFilesNoOpSkip(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Replace "hello" with "hello" — no-op.
	result, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
		},
		Replacement: "hello",
		Apply:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesChanged != 0 {
		t.Errorf("expected FilesChanged=0 for no-op, got %d", result.FilesChanged)
	}
	if result.LinesChanged != 0 {
		t.Errorf("expected LinesChanged=0 for no-op, got %d", result.LinesChanged)
	}

	content, _ := os.ReadFile(path)
	if string(content) != "hello world\n" {
		t.Error("no-op replace should not change file content")
	}
}

func TestReplaceInFilesCaseSensitivity(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("Hello world\nhello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Case-sensitive: only "Hello" matches.
	result, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory:     dir,
			Query:         "Hello",
			CaseSensitive: true,
		},
		Replacement: "Hi",
		Apply:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.LinesChanged != 1 {
		t.Errorf("case-sensitive: expected LinesChanged=1, got %d", result.LinesChanged)
	}

	// Restore the original content so the case-insensitive pass starts clean.
	if err := os.WriteFile(path, []byte("Hello world\nhello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Case-insensitive: both lines match (search default).
	result2, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
		},
		Replacement: "Hi",
		Apply:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result2.LinesChanged != 2 {
		t.Errorf("case-insensitive: expected LinesChanged=2, got %d", result2.LinesChanged)
	}
}

func TestReplaceInFilesMultiFile(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("hello\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
		},
		Replacement: "goodbye",
		Apply:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesChanged != 3 {
		t.Errorf("expected FilesChanged=3, got %d", result.FilesChanged)
	}
	if result.LinesChanged != 3 {
		t.Errorf("expected LinesChanged=3, got %d", result.LinesChanged)
	}

	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		content, _ := os.ReadFile(filepath.Join(dir, name))
		if string(content) != "goodbye\n" {
			t.Errorf("%s: expected 'goodbye\\n', got %q", name, string(content))
		}
	}
}

func TestReplaceInFilesResultDeterministic(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := app.ReplaceInFiles(ReplaceRequest{
		Search: SearchRequest{
			Directory: dir,
			Query:     "hello",
		},
		Replacement: "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Results sorted by file path then line: a.txt:1, b.txt:1
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 file replacements, got %d", len(result.Files))
	}
	if !strings.HasSuffix(result.Files[0].FilePath, "a.txt") {
		t.Errorf("expected first result to be a.txt, got %s", result.Files[0].FilePath)
	}
	if !strings.HasSuffix(result.Files[1].FilePath, "b.txt") {
		t.Errorf("expected second result to be b.txt, got %s", result.Files[1].FilePath)
	}
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}