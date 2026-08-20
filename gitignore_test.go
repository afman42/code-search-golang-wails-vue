package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// loadGitignoreMatcher
// ---------------------------------------------------------------------------

func TestLoadGitignoreMatcherNoFiles(t *testing.T) {
	dir := t.TempDir()
	matcher := loadGitignoreMatcher(dir)
	if matcher != nil {
		t.Error("expected nil matcher when no .gitignore exists")
	}
}

func TestLoadGitignoreMatcherRootIgnore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\nnode_modules/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	matcher := loadGitignoreMatcher(dir)
	if matcher == nil {
		t.Fatal("expected non-nil matcher")
	}

	if !matcher.MatchesPath("debug.log") {
		t.Error("*.log should match debug.log")
	}
	if !matcher.MatchesPath("node_modules/foo.js") {
		t.Error("node_modules/ should match nested files")
	}
	if matcher.MatchesPath("main.go") {
		t.Error("main.go should not be matched")
	}
}

func TestLoadGitignoreMatcherNegation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n!keep.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	matcher := loadGitignoreMatcher(dir)
	if matcher == nil {
		t.Fatal("expected non-nil matcher")
	}

	if !matcher.MatchesPath("debug.log") {
		t.Error("*.log should match debug.log")
	}
	if matcher.MatchesPath("keep.log") {
		t.Error("!keep.log should negate the match for keep.log")
	}
}

func TestLoadGitignoreMatcherInfoExclude(t *testing.T) {
	dir := t.TempDir()
	infoDir := filepath.Join(dir, ".git", "info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infoDir, "exclude"), []byte("*.tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}

	matcher := loadGitignoreMatcher(dir)
	if matcher == nil {
		t.Fatal("expected non-nil matcher from .git/info/exclude")
	}
	if !matcher.MatchesPath("scratch.tmp") {
		t.Error(".git/info/exclude *.tmp should match scratch.tmp")
	}
}

func TestLoadGitignoreMatcherBothSources(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitDir := filepath.Join(dir, ".git", "info")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "exclude"), []byte("*.tmp\n"), 0644); err != nil {
		t.Fatal(err)
	}

	matcher := loadGitignoreMatcher(dir)
	if matcher == nil {
		t.Fatal("expected non-nil matcher from both sources")
	}
	if !matcher.MatchesPath("debug.log") {
		t.Error("should match .gitignore pattern")
	}
	if !matcher.MatchesPath("scratch.tmp") {
		t.Error("should match .git/info/exclude pattern")
	}
}

// ---------------------------------------------------------------------------
// filterByGitignore
// ---------------------------------------------------------------------------

func TestFilterByGitignoreNilMatcher(t *testing.T) {
	files := []fileMeta{{absPath: "/tmp/foo.go", size: 10}}
	got := filterByGitignore(files, "/tmp", nil)
	if len(got) != 1 {
		t.Error("nil matcher should not drop files")
	}
}

func TestFilterByGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}
	matcher := loadGitignoreMatcher(dir)

	files := []fileMeta{
		{absPath: filepath.Join(dir, "main.go"), size: 100},
		{absPath: filepath.Join(dir, "debug.log"), size: 50},
	}

	filtered := filterByGitignore(files, dir, matcher)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 file after gitignore filter, got %d", len(filtered))
	}
	if !endsWith(filtered[0].absPath, "main.go") {
		t.Error("expected main.go to survive filtering")
	}
}

// ---------------------------------------------------------------------------
// Integration: RespectGitignore in collectFilesToProcess
// ---------------------------------------------------------------------------

func TestCollectFilesToProcessRespectGitignore(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "debug.log"), []byte("error\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	req := SearchRequest{
		Directory:        dir,
		Query:            "main",
		MaxFileSize:      10485760,
		RespectGitignore: true,
	}

	files, err := app.collectFilesToProcess(req, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	// debug.log filtered out; .gitignore itself is a text file not matched
	// by the ignore rules, so it survives.
	if len(files) != 2 {
		t.Fatalf("expected 2 files with gitignore, got %d", len(files))
	}
	hasMain := false
	for _, f := range files {
		if endsWith(f.absPath, "main.go") {
			hasMain = true
		}
	}
	if !hasMain {
		t.Error("expected main.go to survive gitignore filter")
	}
}

func TestCollectFilesToProcessRespectGitignoreOff(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "debug.log"), []byte("error\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	req := SearchRequest{
		Directory:        dir,
		Query:            "main",
		MaxFileSize:      10485760,
		RespectGitignore: false,
	}

	files, err := app.collectFilesToProcess(req, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	// All three files survive — same as pre-feature behavior.
	if len(files) != 3 {
		t.Fatalf("expected 3 files with gitignore off, got %d", len(files))
	}
}

func endsWith(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
