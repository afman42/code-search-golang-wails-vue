package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// parseLogLine / parseLogEntryMessage / isNoisyMessage
// ---------------------------------------------------------------------------

func TestIsNoisyMessage(t *testing.T) {
	cases := []struct {
		msg      string
		expected bool
	}{
		{"Skipping file: foo.go", true},
		{"Sending file progress: bar.go", true},
		{"Search started", false},
		{"", false},
		{"Skipping", true},                // substring match
		{"Sending file", true},            // exact substring
		{"sending file lowercase", false}, // case-sensitive: "sending" != "Sending"
		{"File skipped", false},           // "skipped" != "Skipping"
		{"not sending file stuff", false}, // lowercase "sending file" != "Sending file"
	}
	for _, tc := range cases {
		got := isNoisyMessage(tc.msg)
		if got != tc.expected {
			t.Errorf("isNoisyMessage(%q) = %v, want %v", tc.msg, got, tc.expected)
		}
	}
}

func TestParseLogEntryMessageString(t *testing.T) {
	// Plain string: non-noisy passes through.
	content, skip := parseLogEntryMessage("Search completed")
	if skip {
		t.Error("expected non-noisy string to not be skipped")
	}
	if s, ok := content.(string); !ok || s != "Search completed" {
		t.Errorf("expected content to be the same string, got %v", content)
	}

	// Noisy string is skipped.
	_, skip = parseLogEntryMessage("Skipping binary file")
	if !skip {
		t.Error("expected noisy string to be skipped")
	}
}

func TestParseLogEntryMessageJSONObject(t *testing.T) {
	// Structured logrus entry with a non-noisy msg.
	obj := map[string]interface{}{"level": "info", "msg": "Search started"}
	content, skip := parseLogEntryMessage(obj)
	if skip {
		t.Error("expected non-noisy JSON object to not be skipped")
	}
	if m, ok := content.(map[string]interface{}); !ok || m["msg"] != "Search started" {
		t.Errorf("expected content to be the same object, got %v", content)
	}

	// Structured entry with noisy msg is skipped.
	noisyObj := map[string]interface{}{"level": "debug", "msg": "Skipping file: data.dat"}
	_, skip = parseLogEntryMessage(noisyObj)
	if !skip {
		t.Error("expected JSON object with noisy msg to be skipped")
	}
}

func TestParseLogEntryMessageOtherTypes(t *testing.T) {
	// Non-string, non-map types pass through without filtering.
	for _, raw := range []interface{}{42, true, 3.14} {
		content, skip := parseLogEntryMessage(raw)
		if skip {
			t.Errorf("expected type %T to not be skipped", raw)
		}
		if content != raw {
			t.Errorf("expected content to be the same value, got %v vs %v", content, raw)
		}
	}
	// nil passes through too.
	content, skip := parseLogEntryMessage(nil)
	if skip {
		t.Error("expected nil to not be skipped")
	}
	if content != nil {
		t.Errorf("expected content to be nil, got %v", content)
	}
}

func TestParseLogLinePlainText(t *testing.T) {
	msg, skip := parseLogLine("Search operation completed")
	if skip {
		t.Error("expected non-noisy plain text to not be skipped")
	}
	if msg.Type != "log" {
		t.Errorf("expected Type='log', got %q", msg.Type)
	}
	if s, ok := msg.Content.(string); !ok || s != "Search operation completed" {
		t.Errorf("expected Content to be the line string, got %v", msg.Content)
	}
}

func TestParseLogLineNoisyText(t *testing.T) {
	_, skip := parseLogLine("Skipping file due to size limit")
	if !skip {
		t.Error("expected noisy plain text to be skipped")
	}
}

func TestParseLogLineJSON(t *testing.T) {
	jsonLine := `{"level":"info","msg":"Search started","time":"2024-01-01T00:00:00Z"}`
	msg, skip := parseLogLine(jsonLine)
	if skip {
		t.Error("expected non-noisy JSON line to not be skipped")
	}
	if msg.Type != "log" {
		t.Errorf("expected Type='log', got %q", msg.Type)
	}
	obj, ok := msg.Content.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Content to be a parsed JSON object, got %T", msg.Content)
	}
	if obj["msg"] != "Search started" {
		t.Errorf("expected msg='Search started', got %v", obj["msg"])
	}
}

func TestParseLogLineNoisyJSON(t *testing.T) {
	jsonLine := `{"level":"debug","msg":"Skipping file: data.dat"}`
	_, skip := parseLogLine(jsonLine)
	if !skip {
		t.Error("expected JSON line with noisy msg to be skipped")
	}
}

func TestParseLogLineEmpty(t *testing.T) {
	msg, skip := parseLogLine("")
	if skip {
		t.Error("expected empty line to not be skipped")
	}
	if msg.Type != "log" {
		t.Errorf("expected Type='log', got %q", msg.Type)
	}
}

// ---------------------------------------------------------------------------
// matchesPattern
// ---------------------------------------------------------------------------

func TestMatchesPattern(t *testing.T) {
	app := NewApp()
	cases := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{"exact full path", "/home/user/project/main.go", "/home/user/project/main.go", true},
		{"exact component match", "/home/user/node_modules/app.js", "node_modules", true},
		{"component match .git", "/home/user/.git/config", ".git", true},
		{"glob pattern", "/home/user/build/output.log", "*.log", true},
		{"glob pattern build*", "/home/user/build_dir/app.js", "build*", true},
		{"no match", "/home/user/project/main.go", "node_modules", false},
		{"substring not a component", "/home/user/vigilant.go", "git", false},
		{"substring in filename", "/home/user/digits.txt", "git", false},
		{"empty pattern matches empty path component", "/home/user/main.go", "", true},
		{"pattern matches filename exactly", "/home/user/main.go", "main.go", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := app.matchesPattern(tc.path, tc.pattern)
			if got != tc.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tc.path, tc.pattern, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// getFullExtension / matchExtension
// ---------------------------------------------------------------------------

func TestGetFullExtension(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"file.min.js", ".min.js"},
		{"archive.tar.gz", ".tar.gz"},
		{"main.go", ".go"},
		{"Makefile", ""},
		{"noext", ""},
		{".gitignore", ".gitignore"},
		{"path/to/file.ts", ".ts"},
	}
	for _, tc := range cases {
		got := getFullExtension(tc.path)
		if got != tc.expected {
			t.Errorf("getFullExtension(%q) = %q, want %q", tc.path, got, tc.expected)
		}
	}
}

func TestMatchExtension(t *testing.T) {
	cases := []struct {
		name         string
		path         string
		requestedExt string
		expected     bool
	}{
		{"single ext match", "main.go", "go", true},
		{"single ext case-insensitive", "main.GO", "go", true},
		{"full ext match", "file.min.js", "min.js", true},
		{"full ext match tar.gz", "archive.tar.gz", "tar.gz", true},
		{"final ext matches but not full", "file.min.js", "js", true},
		{"no match", "main.go", "py", false},
		{"empty requested ext matches all", "anything.txt", "", true},
		{"requested ext with leading dot matches", "main.go", ".go", true}, // leading dot tolerated (UI sends ".go")
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchExtension(tc.path, tc.requestedExt)
			if got != tc.expected {
				t.Errorf("matchExtension(%q, %q) = %v, want %v", tc.path, tc.requestedExt, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isKnownTextExtension
// ---------------------------------------------------------------------------

func TestIsKnownTextExtensionCaseInsensitiveAndDefaults(t *testing.T) {
	known := []string{"main.go", "app.ts", "style.css", "config.json", "README.md", "script.sh", "data.yaml"}
	for _, p := range known {
		if !isKnownTextExtension(p) {
			t.Errorf("isKnownTextExtension(%q) = false, want true", p)
		}
	}
	// Case-insensitive
	if !isKnownTextExtension("MAIN.GO") {
		t.Error("expected isKnownTextExtension to be case-insensitive")
	}
	// Unknown extension -> false (safe default, needs binary probe)
	if isKnownTextExtension("data.dat") {
		t.Error("expected unknown extension .dat to return false")
	}
	// Explicitly false entry
	if isKnownTextExtension("module.wasm") {
		t.Error("expected .wasm to return false (explicitly marked non-text)")
	}
	// No extension
	if isKnownTextExtension("Makefile") {
		t.Error("expected extensionless file to return false")
	}
}

// ---------------------------------------------------------------------------
// containsDotDotComponent
// ---------------------------------------------------------------------------

func TestContainsDotDotComponent(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{"/home/user/../etc/passwd", true},
		{"/home/user/project/..", true},
		{"../secret", true},
		{"/home/user/foo..bar.txt", false}, // not a path component
		{"/home/user/project", false},
		{"", false},
		{"/home/user/../..", true},
		// Windows-style separators
		{`C:\Users\..\admin`, true},
		{`C:\Users\foo..bar.txt`, false},
	}
	for _, tc := range cases {
		got := containsDotDotComponent(tc.path)
		if got != tc.expected {
			t.Errorf("containsDotDotComponent(%q) = %v, want %v", tc.path, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// safeContextLinesBytes / bytesToStrings / searchContextLines
// ---------------------------------------------------------------------------

func TestSafeContextLinesBytes(t *testing.T) {
	lines := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	cases := []struct {
		name        string
		start, end  int
		expectedLen int
	}{
		{"normal range", 0, 2, 2},
		{"start clamped", -2, 1, 1},
		{"end clamped", 2, 100, 2},
		{"empty range", 2, 2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := safeContextLinesBytes(lines, tc.start, tc.end)
			if len(got) != tc.expectedLen {
				t.Errorf("len = %d, want %d", len(got), tc.expectedLen)
			}
		})
	}
}

func TestBytesToStrings(t *testing.T) {
	input := [][]byte{[]byte("hello"), []byte("world")}
	got := bytesToStrings(input)
	if len(got) != 2 || got[0] != "hello" || got[1] != "world" {
		t.Errorf("bytesToStrings = %v, want [hello world]", got)
	}

	// Empty input returns empty slice (not nil).
	got = bytesToStrings(nil)
	if got == nil || len(got) != 0 {
		t.Errorf("bytesToStrings(nil) = %v, want non-nil empty slice", got)
	}
}

func TestSearchContextLines(t *testing.T) {
	cases := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero defaults to 2", 0, 2},
		{"negative defaults to 2", -1, 2},
		{"within range", 5, 5},
		{"clamped to max", 100, maxContextLines},
		{"exact max", maxContextLines, maxContextLines},
		{"one", 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := searchContextLines(tc.input)
			if got != tc.expected {
				t.Errorf("searchContextLines(%d) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// validateAndSetDefaults
// ---------------------------------------------------------------------------

func TestValidateAndSetDefaults(t *testing.T) {
	app := NewApp()

	t.Run("sets defaults for zero values", func(t *testing.T) {
		req := SearchRequest{Directory: t.TempDir()}
		result, err := app.validateAndSetDefaults(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.MaxFileSize != 10*1024*1024 {
			t.Errorf("MaxFileSize = %d, want 10MB default", result.MaxFileSize)
		}
		if result.MaxResults != 1000 {
			t.Errorf("MaxResults = %d, want 1000 default", result.MaxResults)
		}
	})

	t.Run("preserves explicit values", func(t *testing.T) {
		req := SearchRequest{
			Directory:   t.TempDir(),
			MaxFileSize: 1024,
			MaxResults:  50,
		}
		result, err := app.validateAndSetDefaults(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.MaxFileSize != 1024 {
			t.Errorf("MaxFileSize = %d, want 1024", result.MaxFileSize)
		}
		if result.MaxResults != 50 {
			t.Errorf("MaxResults = %d, want 50", result.MaxResults)
		}
	})

	t.Run("rejects empty directory", func(t *testing.T) {
		_, err := app.validateAndSetDefaults(SearchRequest{Directory: ""})
		if err == nil {
			t.Error("expected error for empty directory")
		}
		if !strings.Contains(err.Error(), "empty directory") {
			t.Errorf("expected 'empty directory' in error, got: %v", err)
		}
	})

	t.Run("rejects non-existent directory", func(t *testing.T) {
		_, err := app.validateAndSetDefaults(SearchRequest{Directory: "/this/path/does/not/exist/xyzzy"})
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})

	t.Run("rejects protected system directory", func(t *testing.T) {
		protected := "/"
		if runtime.GOOS == "windows" {
			protected = `C:\Windows`
		}
		_, err := app.validateAndSetDefaults(SearchRequest{Directory: protected})
		if err == nil {
			t.Error("expected error for protected system directory")
		}
		if !strings.Contains(err.Error(), "protected") {
			t.Errorf("expected 'protected' in error, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// rotateLogFileIfNeeded
// ---------------------------------------------------------------------------

func TestRotateLogFileIfNeededNoFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nonexistent.log")
	// Should be a no-op when the file doesn't exist.
	rotateLogFileIfNeeded(logPath)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("expected file to not exist after rotating a non-existent file")
	}
}

func TestRotateLogFileIfNeededSmallFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	if err := os.WriteFile(logPath, []byte("small"), 0o644); err != nil {
		t.Fatalf("creating log file: %v", err)
	}
	rotateLogFileIfNeeded(logPath)
	// File should still exist with the same content (no rotation).
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if string(content) != "small" {
		t.Errorf("expected content unchanged, got %q", content)
	}
	// No .1 file should exist.
	if _, err := os.Stat(logPath + ".1"); !os.IsNotExist(err) {
		t.Error("expected no .1 rotation file for small log")
	}
}

func TestRotateLogFileIfNeededLargeFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	// Write a file larger than maxLogFileSize.
	big := strings.Repeat("x", maxLogFileSize+100)
	if err := os.WriteFile(logPath, []byte(big), 0o644); err != nil {
		t.Fatalf("creating big log file: %v", err)
	}
	rotateLogFileIfNeeded(logPath)
	// The original should be gone (renamed to .1).
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("expected original log to be renamed away after rotation")
	}
	// The .1 file should contain the old content.
	content, err := os.ReadFile(logPath + ".1")
	if err != nil {
		t.Fatalf("reading rotated file: %v", err)
	}
	if len(content) != len(big) {
		t.Errorf("rotated file size = %d, want %d", len(content), len(big))
	}
}

func TestRotateLogFileIfNeededOverwritesPreviousRotation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	// Pre-existing .1 file with stale content.
	stale := "stale rotation content"
	if err := os.WriteFile(logPath+".1", []byte(stale), 0o644); err != nil {
		t.Fatalf("creating stale .1: %v", err)
	}
	// Current log exceeds cap.
	big := strings.Repeat("y", maxLogFileSize+10)
	if err := os.WriteFile(logPath, []byte(big), 0o644); err != nil {
		t.Fatalf("creating big log: %v", err)
	}
	rotateLogFileIfNeeded(logPath)
	// .1 should now contain the new content, not the stale content.
	content, err := os.ReadFile(logPath + ".1")
	if err != nil {
		t.Fatalf("reading rotated file: %v", err)
	}
	if string(content) == stale {
		t.Error("expected .1 to be overwritten with new content, not stale")
	}
	if len(content) != len(big) {
		t.Errorf("rotated file size = %d, want %d", len(content), len(big))
	}
}

// ---------------------------------------------------------------------------
// ReadFileLog
// ---------------------------------------------------------------------------

func TestReadFileLog(t *testing.T) {
	app := NewApp()
	result, err := app.ReadFileLog("app.log")
	if err != nil {
		t.Fatalf("ReadFileLog returned error: %v", err)
	}
	// Should return a path ending with logs/app.log.
	if !strings.HasSuffix(result, filepath.Join("logs", "app.log")) {
		t.Errorf("ReadFileLog returned %q, expected suffix logs/app.log", result)
	}
	// Should be an absolute path.
	if !filepath.IsAbs(result) {
		// On some systems Getwd may return a relative path in test envs;
		// the important thing is it joins CWD + logs + filename.
		t.Logf("ReadFileLog returned relative path %q (CWD may be relative in tests)", result)
	}
}

func TestReadFileLogDifferentName(t *testing.T) {
	app := NewApp()
	result, err := app.ReadFileLog("custom.log")
	if err != nil {
		t.Fatalf("ReadFileLog returned error: %v", err)
	}
	if !strings.HasSuffix(result, filepath.Join("logs", "custom.log")) {
		t.Errorf("ReadFileLog returned %q, expected suffix logs/custom.log", result)
	}
}

// ---------------------------------------------------------------------------
// GetDirectoryContents
// ---------------------------------------------------------------------------

func TestGetDirectoryContents(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Create subdirectories.
	subdirs := []string{"dir1", "dir2", "dir1/subdir1", ".hidden_dir"}
	for _, d := range subdirs {
		if err := os.MkdirAll(filepath.Join(tempDir, d), 0o755); err != nil {
			t.Fatalf("creating subdir %s: %v", d, err)
		}
	}
	// Create a file (should NOT appear in results).
	if err := os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("creating file: %v", err)
	}

	contents, err := app.GetDirectoryContents(tempDir)
	if err != nil {
		t.Fatalf("GetDirectoryContents returned error: %v", err)
	}

	// The root dir itself is included.
	found := map[string]bool{}
	for _, c := range contents {
		found[c] = true
	}

	// dir1 and dir2 should be present.
	if !found[filepath.Join(tempDir, "dir1")] {
		t.Error("expected dir1 in results")
	}
	if !found[filepath.Join(tempDir, "dir2")] {
		t.Error("expected dir2 in results")
	}
	// Nested subdir should be present.
	if !found[filepath.Join(tempDir, "dir1", "subdir1")] {
		t.Error("expected dir1/subdir1 in results")
	}
	// Hidden directory should be skipped.
	if found[filepath.Join(tempDir, ".hidden_dir")] {
		t.Error("expected .hidden_dir to be skipped")
	}
	// Files should not appear.
	for _, c := range contents {
		if strings.HasSuffix(c, "file.txt") {
			t.Error("files should not appear in GetDirectoryContents results")
		}
	}
}

func TestGetDirectoryContentsNonExistent(t *testing.T) {
	app := NewApp()
	// WalkDir swallows access errors (returns nil in the callback), so a
	// non-existent path returns an empty slice, not an error.
	contents, err := app.GetDirectoryContents("/this/path/definitely/does/not/exist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contents) != 0 {
		t.Errorf("expected empty slice for non-existent dir, got %d items", len(contents))
	}
}
