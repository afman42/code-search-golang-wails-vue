package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestIsBinaryNoAllocationOnNullByteCheck is a regression test for the
// isBinary optimization (#7). The previous implementation called
// string(content[:min(512,len(content))]) just to run strings.Contains for
// "\x00", which allocated a string on every file probe. The new
// implementation uses bytes.Contains with a reusable []byte{0} so no string
// allocation happens. We verify the BEHAVIOR is unchanged (null-byte
// content is still detected as binary) and the allocation count is zero.
func TestIsBinaryNoAllocationOnNullByteCheck(t *testing.T) {
	app := NewApp()

	// Content with a null byte in the first 512 bytes must be detected
	// as binary. Put the null byte near the start so it's within the
	// checked range.
	content := []byte("hello\x00world this is binary")
	if !app.isBinary(content) {
		t.Error("expected content with null byte to be detected as binary")
	}

	// Content with a null byte AFTER byte 512 must NOT be flagged by the
	// null-byte check (the check only scans the first 512 bytes). The
	// printable ratio should still mark it as text.
	textStart := bytes.Repeat([]byte("a"), 600)
	textWithLateNull := append(textStart, []byte("\x00")...)
	if app.isBinary(textWithLateNull) {
		t.Error("expected content with null byte after byte 512 to be treated as text (null byte out of scanned range)")
	}

	// Verify the allocation count is zero for a typical small file. The
	// previous string(content[:n]) path allocated; bytes.Contains with a
	// pre-declared nullByte slice does not.
	plainText := []byte("just some plain text content without any null bytes here")
	allocs := testing.AllocsPerRun(100, func() {
		_ = app.isBinary(plainText)
	})
	if allocs != 0 {
		t.Errorf("expected 0 allocations in isBinary for plain text, got %v (#7 regression)", allocs)
	}
}

// TestIsBinaryEmptyAndShort verifies edge cases that the allocation
// optimization could have broken: empty content and content shorter than
// 512 bytes.
func TestIsBinaryEmptyAndShort(t *testing.T) {
	app := NewApp()

	if app.isBinary(nil) {
		t.Error("expected nil content to not be binary")
	}
	if app.isBinary([]byte{}) {
		t.Error("expected empty content to not be binary")
	}
	short := []byte("hi")
	if app.isBinary(short) {
		t.Error("expected 2-byte text content to not be binary")
	}
	// Short binary content (null byte) must still be caught.
	shortBin := []byte("h\x00i")
	if !app.isBinary(shortBin) {
		t.Error("expected 3-byte content with null byte to be binary")
	}
}

// TestCompileSearchPatternLiteralModeNoExtraCompile verifies that the
// literal-mode pattern compilation (#11) no longer runs a redundant
// regexp.Compile(req.Query) on the raw query. The previous code compiled
// the raw query once (just to check if it WOULD be a valid regex) and then
// compiled the escaped version — two compiles per literal search. The fix
// removes the raw compile, so literal mode is one compile, not two.
//
// We verify the behavioral invariant: literal mode escapes and compiles
// successfully even when the raw query would be an invalid regex.
func TestCompileSearchPatternLiteralModeNoExtraCompile(t *testing.T) {
	app := NewApp()

	falseValue := false
	req := SearchRequest{
		Query:    "[invalid", // would be invalid regex, valid literal
		UseRegex: &falseValue,
	}

	pattern, err := app.compileSearchPattern(req)
	if err != nil {
		t.Fatalf("literal mode must compile '[invalid' without error, got: %v", err)
	}
	if pattern == nil {
		t.Fatal("expected a compiled pattern, got nil")
	}
	// The compiled pattern should match the literal "[invalid".
	if !pattern.MatchString("this has [invalid brackets") {
		t.Error("expected literal-mode pattern to match the literal '[invalid' string")
	}
	if pattern.MatchString("this does not have the string") {
		t.Error("literal-mode pattern matched a string that doesn't contain the literal")
	}
}

// TestCompileSearchPatternRegexModeRejectsInvalid verifies that regex mode
// still rejects an invalid regex (this is the mode where invalid regex
// SHOULD be an error — the literal-mode rejection was the dead-code bug #11).
func TestCompileSearchPatternRegexModeRejectsInvalid(t *testing.T) {
	app := NewApp()

	trueValue := true
	req := SearchRequest{
		Query:    "[unclosed", // invalid regex
		UseRegex: &trueValue,
	}

	if _, err := app.compileSearchPattern(req); err == nil {
		t.Error("expected regex mode to reject invalid regex '[unclosed', got nil error")
	}
}

// TestBinaryCheckBufferPoolReused verifies that the binary-detection buffer
// pool (#8) returns a 512-byte buffer. The previous implementation
// allocated make([]byte, 512) per file; the pool returns a *[]byte so the
// backing array can be reused across files.
func TestBinaryCheckBufferPoolReused(t *testing.T) {
	// Get a buffer, put it back, get again — the pool should typically
	// return the same pointer (sync.Pool preserves the most recently Put
	// item for the next Get on the same P).
	buf1 := binaryCheckBufPool.Get().(*[]byte)
	binaryCheckBufPool.Put(buf1)
	buf2 := binaryCheckBufPool.Get().(*[]byte)
	defer binaryCheckBufPool.Put(buf2)

	if buf1 != buf2 {
		// Not a hard failure — sync.Pool doesn't GUARANTEE reuse. Log
		// so we notice if the pool wiring is broken, but don't fail.
		t.Logf("note: pool did not return the same buffer on immediate reuse (buf1=%p buf2=%p) — sync.Pool doesn't guarantee reuse", buf1, buf2)
	}

	// Verify the buffer is the right size (512 bytes).
	buf := *buf2
	if cap(buf) != 512 {
		t.Errorf("expected buffer capacity 512, got %d", cap(buf))
	}
}

// TestProcessFileUsesBytesSplitNotStringsSplit is a behavioral regression
// test for #10. The previous implementation did
// strings.Split(string(content), "\n") which allocated a string copy of
// the entire file content plus a []string of all lines. The new
// implementation uses bytes.Split and only converts individual matched
// lines to strings. We verify the functional invariant: matches are found
// correctly with the same ContextBefore/ContextAfter as before.
func TestProcessFileUsesBytesSplitNotStringsSplit(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()

	// Create a file WITH a match and verify the match is found correctly,
	// including context lines (the bytes.Split path must produce the same
	// ContextBefore/ContextAfter as the old strings.Split path).
	matchContent := []byte("alpha\nbeta\nMATCH\ngamma\ndelta\n")
	matchFile := filepath.Join(tempDir, "match.txt")
	if err := os.WriteFile(matchFile, matchContent, 0o644); err != nil {
		t.Fatalf("writing match file: %v", err)
	}

	trueValue := true
	pattern := compilePatternOrFatal(t, "MATCH", &trueValue)
	req := SearchRequest{
		Directory:   tempDir,
		Query:       "MATCH",
		UseRegex:    &trueValue,
		MaxResults:  1000,
		MaxFileSize: 10 * 1024 * 1024,
	}

	searchState := &SearchState{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, results := app.processFile(ctx, fileMeta{absPath: matchFile, size: int64(len(matchContent))}, pattern, req, searchState, new(int32), cancel)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.LineNum != 3 {
		t.Errorf("expected match on line 3, got line %d", r.LineNum)
	}
	if r.Content != "MATCH" {
		t.Errorf("expected Content 'MATCH', got %q", r.Content)
	}
	if len(r.ContextBefore) != 2 || r.ContextBefore[0] != "alpha" || r.ContextBefore[1] != "beta" {
		t.Errorf("expected ContextBefore [alpha, beta], got %v", r.ContextBefore)
	}
	if len(r.ContextAfter) != 2 || r.ContextAfter[0] != "gamma" || r.ContextAfter[1] != "delta" {
		t.Errorf("expected ContextAfter [gamma, delta], got %v", r.ContextAfter)
	}
}

// TestProcessFileSkipsRedundantBinaryCheck verifies that processFile no
// longer re-runs isBinary on small files when !req.IncludeBinary (#4). The
// previous code did a full os.ReadFile then isBinary(content) check even
// though collectFilesToProcess had already filtered binary files. The fix
// removes the redundant check, so a small binary file that IS passed to
// processFile (when IncludeBinary=true) still gets searched.
//
// We verify the behavior by searching a binary file with IncludeBinary=true
// — the search must proceed (and find the pattern if present) rather than
// skip the file as binary.
func TestProcessFileSkipsRedundantBinaryCheck(t *testing.T) {
	app := NewApp()

	tempDir := t.TempDir()
	// A file with a null byte (binary) that also contains a search term.
	binaryWithTerm := []byte("the password is hunter2\x00more binary data")
	binaryFile := filepath.Join(tempDir, "binary.dat")
	if err := os.WriteFile(binaryFile, binaryWithTerm, 0o644); err != nil {
		t.Fatalf("writing binary file: %v", err)
	}

	trueValue := true
	pattern := compilePatternOrFatal(t, "hunter2", &trueValue)

	// With IncludeBinary=true, processFile should search the file (the
	// redundant isBinary check would have skipped it even when the user
	// explicitly asked to include binaries — that was the bug).
	req := SearchRequest{
		Directory:    tempDir,
		Query:        "hunter2",
		UseRegex:     &trueValue,
		IncludeBinary: true,
		MaxResults:   1000,
		MaxFileSize:  10 * 1024 * 1024,
	}

	searchState := &SearchState{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, results := app.processFile(ctx, fileMeta{absPath: binaryFile, size: int64(len(binaryWithTerm))}, pattern, req, searchState, new(int32), cancel)
	if len(results) == 0 {
		t.Error("expected processFile to find 'hunter2' in a binary file when IncludeBinary=true, got 0 results (redundant isBinary check may have skipped it #4)")
	}
}

// compilePatternOrFatal compiles a search pattern via the App's
// compileSearchPattern and fails the test if compilation errors. Used by
// tests that need a *regexp.Regexp to pass directly to processFile.
func compilePatternOrFatal(t *testing.T, query string, useRegex *bool) *regexp.Regexp {
	t.Helper()
	app := NewApp()
	p, err := app.compileSearchPattern(SearchRequest{Query: query, UseRegex: useRegex})
	if err != nil {
		t.Fatalf("compiling pattern %q: %v", query, err)
	}
	return p
}
