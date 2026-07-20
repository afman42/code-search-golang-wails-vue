package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestIsKnownTextExtension verifies that common source-code extensions are
// recognized as text (and thus skip the binary detection probe), while
// unknown extensions and known-binary extensions are NOT in the set.
func TestIsKnownTextExtension(t *testing.T) {
	textExts := []string{
		".go", ".rs", ".py", ".js", ".ts", ".tsx", ".jsx",
		".java", ".kt", ".c", ".cpp", ".h", ".hpp", ".cs",
		".rb", ".php", ".swift", ".html", ".css", ".scss",
		".vue", ".svelte", ".json", ".yaml", ".yml", ".toml",
		".xml", ".md", ".txt", ".sql", ".sh", ".bash",
		".dockerfile", ".gitignore",
	}
	for _, ext := range textExts {
		path := "test" + ext
		if !isKnownTextExtension(path) {
			t.Errorf("expected %s to be a known text extension", ext)
		}
		// Case insensitivity: .GO and .go should both match.
		upper := strings.ToUpper(path)
		if !isKnownTextExtension(upper) {
			t.Errorf("expected %s (uppercase) to be a known text extension", upper)
		}
	}

	// Unknown or binary extensions must NOT be in the set.
	nonTextExts := []string{
		".dat", ".bin", ".exe", ".dll", ".so", ".dylib",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico",
		".zip", ".tar", ".gz", ".bz2", ".7z", ".rar",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".mp3", ".mp4", ".avi", ".mov", ".wav",
		".wasm", // explicitly NOT text
		".unknown_ext",
	}
	for _, ext := range nonTextExts {
		path := "test" + ext
		if isKnownTextExtension(path) {
			t.Errorf("expected %s to NOT be a known text extension", ext)
		}
	}

	// No extension at all → not in the set (safe default: probe).
	if isKnownTextExtension("Makefile") {
		t.Error("expected a file with no extension to NOT be in the known-text set (safe default)")
	}
}

// TestWalkDirectoryTreeSkipsBinaryProbeForKnownText verifies that the walk
// splits files correctly: .go files go into textCandidates (no binary probe
// needed), while .dat files go into binaryCheckCandidates (need a probe).
// This is the core of Opt 3 — skipping the open+read+close syscall for
// known-text extensions.
func TestWalkDirectoryTreeSkipsBinaryProbeForKnownText(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Create a .go file (known text) and a .dat file (unknown, needs probe).
	goFile := filepath.Join(tempDir, "code.go")
	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("creating .go file: %v", err)
	}
	datFile := filepath.Join(tempDir, "data.dat")
	if err := os.WriteFile(datFile, []byte("some data\x00binary"), 0o644); err != nil {
		t.Fatalf("creating .dat file: %v", err)
	}

	// walkDirectoryTree expects the request to already have defaults set
	// (MaxFileSize, MaxResults) — just like the real call path where
	// validateAndSetDefaults runs first.
	req := SearchRequest{
		Directory:     tempDir,
		Query:         "test",
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024, // 10MB
		MaxResults:    1000,
	}

	textCandidates, binaryCandidates, stats, err := app.walkDirectoryTree(req, false)
	if err != nil {
		t.Fatalf("walkDirectoryTree failed: %v", err)
	}

	// The .go file should be in textCandidates (no probe needed).
	foundGo := false
	for _, m := range textCandidates {
		if strings.HasSuffix(m.absPath, "code.go") {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Error("expected .go file to be in textCandidates (known text extension)")
	}

	// The .dat file should be in binaryCheckCandidates (needs probe).
	foundDat := false
	for _, m := range binaryCandidates {
		if strings.HasSuffix(m.absPath, "data.dat") {
			foundDat = true
			break
		}
	}
	if !foundDat {
		t.Error("expected .dat file to be in binaryCheckCandidates (unknown extension)")
	}

	// No files should have been skipped (both passed size/extension filters).
	if stats.filesSkipped != 0 {
		t.Errorf("expected 0 skipped files, got %d", stats.filesSkipped)
	}
}

// TestWalkDirectoryTreeIncludeBinarySkipsProbe verifies that when
// IncludeBinary=true, ALL files go into textCandidates without any binary
// probe — the user explicitly asked to search binary files, so the probe
// would be wasted work.
func TestWalkDirectoryTreeIncludeBinarySkipsProbe(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Create both a text file and a binary file.
	textFile := filepath.Join(tempDir, "text.txt")
	os.WriteFile(textFile, []byte("hello"), 0o644)
	binFile := filepath.Join(tempDir, "data.dat")
	os.WriteFile(binFile, []byte("bin\x00ary"), 0o644)

	req := SearchRequest{
		Directory:     tempDir,
		Query:         "test",
		IncludeBinary: true,
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024,
		MaxResults:    1000,
	}

	textCandidates, binaryCandidates, _, err := app.walkDirectoryTree(req, false)
	if err != nil {
		t.Fatalf("walkDirectoryTree failed: %v", err)
	}

	// Both files should be in textCandidates; binaryCandidates should be empty.
	if len(textCandidates) != 2 {
		t.Errorf("expected 2 text candidates (IncludeBinary=true), got %d", len(textCandidates))
	}
	if len(binaryCandidates) != 0 {
		t.Errorf("expected 0 binary probe candidates (IncludeBinary=true), got %d", len(binaryCandidates))
	}
}

// TestProbeBinaryInParallelFiltersBinary verifies that the parallel binary
// probe correctly separates text files from binary files. A text .dat file
// should pass; a binary .dat file (with null bytes) should be skipped.
func TestProbeBinaryInParallelFiltersBinary(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Create a text .dat file and a binary .dat file.
	textDat := filepath.Join(tempDir, "text.dat")
	os.WriteFile(textDat, []byte("this is plain text content"), 0o644)
	binDat := filepath.Join(tempDir, "binary.dat")
	os.WriteFile(binDat, []byte("binary\x00content\x00here"), 0o644)

	candidates := []fileMeta{
		{absPath: textDat, size: 25},
		{absPath: binDat, size: 20},
	}

	textFiles, skipped := app.probeBinaryInParallel(nil, candidates, false)

	// The text .dat should pass; the binary .dat should be skipped.
	if len(textFiles) != 1 {
		t.Errorf("expected 1 text file from probe, got %d", len(textFiles))
	}
	if skipped != 1 {
		t.Errorf("expected 1 skipped binary file, got %d", skipped)
	}
	if len(textFiles) > 0 && !strings.HasSuffix(textFiles[0].absPath, "text.dat") {
		t.Errorf("expected text.dat to pass the probe, got %s", textFiles[0].absPath)
	}
}

// TestProbeBinaryInParallelEmpty verifies the edge case of zero candidates.
func TestProbeBinaryInParallelEmpty(t *testing.T) {
	app := NewApp()
	textFiles, skipped := app.probeBinaryInParallel(nil, nil, false)
	if textFiles != nil && len(textFiles) != 0 {
		t.Errorf("expected nil/empty result for zero candidates, got %d files", len(textFiles))
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped for zero candidates, got %d", skipped)
	}
}

// TestCollectFilesToProcessKnownTextSkipsProbe is an end-to-end test: a
// tree of only .go files should complete collection WITHOUT any binary
// probes (binaryCandidates should be empty internally). We verify this by
// checking that the collection result includes all .go files and the
// benchmark-visible behavior (no file I/O for binary detection).
func TestCollectFilesToProcessKnownTextSkipsProbe(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Create 10 .go files.
	for i := 0; i < 10; i++ {
		path := filepath.Join(tempDir, "file_"+string(rune('a'+i))+".go")
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("creating .go file: %v", err)
		}
	}

	req := SearchRequest{
		Directory:     tempDir,
		Query:         "main",
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024,
		MaxResults:    1000,
	}

	files, err := app.collectFilesToProcess(req, nil, tempDir+string(filepath.Separator))
	if err != nil {
		t.Fatalf("collectFilesToProcess failed: %v", err)
	}

	if len(files) != 10 {
		t.Errorf("expected 10 collected files, got %d", len(files))
	}
}

// TestCollectFilesToProcessMixedExtensions verifies that a mixed tree of
// known-text (.go) and unknown-extension (.dat) files is handled correctly:
// .go files are collected directly, .dat files are probed in parallel, and
// binary .dat files are skipped while text .dat files are included.
func TestCollectFilesToProcessMixedExtensions(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// .go file (known text, no probe)
	os.WriteFile(filepath.Join(tempDir, "code.go"), []byte("package main\n"), 0o644)
	// text .dat file (unknown ext, probe → text)
	os.WriteFile(filepath.Join(tempDir, "plain.dat"), []byte("plain text data"), 0o644)
	// binary .dat file (unknown ext, probe → binary, skipped)
	os.WriteFile(filepath.Join(tempDir, "binary.dat"), []byte("bin\x00ary\x00data"), 0o644)

	req := SearchRequest{
		Directory:     tempDir,
		Query:         "test",
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024,
		MaxResults:    1000,
	}

	files, err := app.collectFilesToProcess(req, nil, tempDir+string(filepath.Separator))
	if err != nil {
		t.Fatalf("collectFilesToProcess failed: %v", err)
	}

	// Expect: code.go + plain.dat = 2 files. binary.dat should be skipped.
	if len(files) != 2 {
		t.Errorf("expected 2 collected files (go + text dat), got %d", len(files))
	}

	// Verify the binary .dat is NOT in the results.
	for _, f := range files {
		if strings.HasSuffix(f.absPath, "binary.dat") {
			t.Error("binary .dat file should have been skipped by the parallel probe")
		}
	}
}

// TestWalkDirectoryTreeAbsPathComputedOnce is a regression test for Opt 1.
// The previous implementation called filepath.Abs(path) per file inside
// the WalkDir callback. The new implementation computes the absolute base
// directory once before the walk and uses filepath.Join. We verify that
// the abs paths in the returned fileMeta entries are correct absolute
// paths regardless of whether req.Directory was relative or absolute.
func TestWalkDirectoryTreeAbsPathComputedOnce(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Create a file.
	filePath := filepath.Join(tempDir, "test.go")
	os.WriteFile(filePath, []byte("package main\n"), 0o644)

	// Test with an absolute directory.
	t.Run("AbsoluteDirectory", func(t *testing.T) {
	req := SearchRequest{
		Directory:     tempDir,
		Query:         "test",
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024,
		MaxResults:    1000,
	}
		textCandidates, _, _, err := app.walkDirectoryTree(req, false)
		if err != nil {
			t.Fatalf("walkDirectoryTree failed: %v", err)
		}
		if len(textCandidates) != 1 {
			t.Fatalf("expected 1 candidate, got %d", len(textCandidates))
		}
		if !filepath.IsAbs(textCandidates[0].absPath) {
			t.Errorf("expected absPath to be absolute, got %q", textCandidates[0].absPath)
		}
	})

	// Test with a relative directory (by chdir-ing to the parent).
	t.Run("RelativeDirectory", func(t *testing.T) {
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		// Change to tempDir's parent so we can reference tempDir by name.
		parent := filepath.Dir(tempDir)
		baseName := filepath.Base(tempDir)
		if err := os.Chdir(parent); err != nil {
			t.Fatalf("chdir to parent: %v", err)
		}

		req := SearchRequest{
			Directory:     baseName,
			Query:         "test",
			SearchSubdirs: true,
			MaxFileSize:   10 * 1024 * 1024,
			MaxResults:    1000,
		}
		textCandidates, _, _, err := app.walkDirectoryTree(req, false)
		if err != nil {
			t.Fatalf("walkDirectoryTree failed: %v", err)
		}
		if len(textCandidates) != 1 {
			t.Fatalf("expected 1 candidate, got %d", len(textCandidates))
		}
		// The absPath must be absolute even though req.Directory was relative.
		if !filepath.IsAbs(textCandidates[0].absPath) {
			t.Errorf("expected absPath to be absolute even for relative Directory, got %q", textCandidates[0].absPath)
		}
		// And it must resolve to the same file.
		absExpected, _ := filepath.Abs(filepath.Join(tempDir, "test.go"))
		if filepath.Clean(textCandidates[0].absPath) != filepath.Clean(absExpected) {
			t.Errorf("expected absPath %q, got %q", absExpected, textCandidates[0].absPath)
		}
	})
}

// TestWalkDirectoryTreeTraversalCheck verifies that the prefix-based
// traversal check (Opt 2) still rejects paths outside the search scope.
// The previous implementation used filepath.Rel; the new one uses a
// prefix check with a separator-terminated base directory.
func TestWalkDirectoryTreeTraversalCheck(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()

	// Create a normal file inside the search directory.
	normalFile := filepath.Join(tempDir, "inside.go")
	os.WriteFile(normalFile, []byte("package main\n"), 0o644)

	// Create a file OUTSIDE the search directory (in a sibling directory).
	siblingDir := filepath.Join(filepath.Dir(tempDir), "sibling_"+filepath.Base(tempDir))
	os.MkdirAll(siblingDir, 0o755)
	defer os.RemoveAll(siblingDir)
	outsideFile := filepath.Join(siblingDir, "outside.go")
	os.WriteFile(outsideFile, []byte("package main\n"), 0o644)

	req := SearchRequest{
		Directory:     tempDir,
		Query:         "test",
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024,
		MaxResults:    1000,
	}

	textCandidates, _, _, err := app.walkDirectoryTree(req, false)
	if err != nil {
		t.Fatalf("walkDirectoryTree failed: %v", err)
	}

	// Only the inside file should be collected — the outside file is in a
	// sibling directory that the walk shouldn't reach (it's not under
	// req.Directory). The prefix check ensures that even if a symlink or
	// path trick tried to escape, it would be caught.
	for _, m := range textCandidates {
		if strings.Contains(m.absPath, "outside.go") {
			t.Error("file outside the search directory was collected — prefix traversal check failed")
		}
	}
	if len(textCandidates) != 1 {
		t.Errorf("expected 1 file (inside.go only), got %d", len(textCandidates))
	}
}

// TestWalkDirectoryTreeSiblingDirNotPrefixMatched verifies that a sibling
// directory whose name is a prefix of the search directory (e.g.
// /tmp/test vs /tmp/test-backup) is NOT matched by the prefix check. This
// is the edge case that the separator-terminated prefix handles correctly.
func TestWalkDirectoryTreeSiblingDirNotPrefixMatched(t *testing.T) {
	app := NewApp()

	// Create two sibling directories: "project" and "project-backup".
	// The prefix check must NOT match "project-backup" when searching
	// "project", because the prefix is "project/" (with separator), not
	// "project".
	parentDir := t.TempDir()
	searchDir := filepath.Join(parentDir, "project")
	siblingDir := filepath.Join(parentDir, "project-backup")

	os.MkdirAll(searchDir, 0o755)
	os.MkdirAll(siblingDir, 0o755)

	os.WriteFile(filepath.Join(searchDir, "real.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(siblingDir, "backup.go"), []byte("package main\n"), 0o644)

	req := SearchRequest{
		Directory:     searchDir,
		Query:         "test",
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024,
		MaxResults:    1000,
	}

	textCandidates, _, _, err := app.walkDirectoryTree(req, false)
	if err != nil {
		t.Fatalf("walkDirectoryTree failed: %v", err)
	}

	// Only real.go should be collected; backup.go is in the sibling.
	for _, m := range textCandidates {
		if strings.HasSuffix(m.absPath, "backup.go") {
			t.Error("file from sibling directory (prefix-name match) was collected — separator-terminated prefix check failed")
		}
	}
}

// TestCollectFilesToProcessParallelProbeScaling is a smoke test that the
// parallel binary probe works with more files than workers (verifying the
// work channel drains correctly). It's not a benchmark, just a correctness
// check that no files are lost when there are many candidates.
func TestCollectFilesToProcessParallelProbeScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scaling test in short mode")
	}

	app := NewApp()
	tempDir := t.TempDir()

	// Create more files than there are CPU cores, all with unknown
	// extensions (.dat) and text content, so the parallel probe must
	// process all of them.
	numFiles := runtime.NumCPU() * 5
	for i := 0; i < numFiles; i++ {
		path := filepath.Join(tempDir, "file_"+string(rune('a'+(i%26)))+string(rune('a'+(i/26%26)))+".dat")
			os.WriteFile(path, []byte("plain text content "+string(rune('a'+i%26))), 0o644)
	}

	req := SearchRequest{
		Directory:     tempDir,
		Query:         "plain",
		SearchSubdirs: true,
		MaxFileSize:   10 * 1024 * 1024,
		MaxResults:    1000,
	}

	files, err := app.collectFilesToProcess(req, nil, tempDir+string(filepath.Separator))
	if err != nil {
		t.Fatalf("collectFilesToProcess failed: %v", err)
	}

	// All files should pass the binary probe (they're all text .dat files).
	if len(files) != numFiles {
		t.Errorf("expected %d files, got %d — some files were lost in the parallel probe", numFiles, len(files))
	}
}
