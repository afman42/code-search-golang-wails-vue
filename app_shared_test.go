package main

import (
	"os"
	"path/filepath"
	"testing"
)

// app_shared.go is the trust boundary for every editor launch and folder
// reveal: each platform's ShowInFolder / openInEditor routes its user-supplied
// path through these validators before handing it to an exec. They had no
// direct tests — only coverage through the platform wrappers, which are
// build-tagged and so only ever exercised on one OS at a time.

// TestSanitizePathRejectsTraversal covers the two-stage check: the ORIGINAL
// input and the cleaned path are both tested, because filepath.Clean resolves
// "/tmp/../etc/passwd" to "/etc/passwd" and would otherwise hide the intent.
func TestSanitizePathRejectsTraversal(t *testing.T) {
	app := NewApp()

	rejected := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"bare parent", ".."},
		{"leading parent", "../etc/passwd"},
		{"embedded parent", "/tmp/../etc/passwd"},
		{"trailing parent", "/tmp/foo/.."},
		{"nested parent", "/tmp/a/../../etc/passwd"},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := app.sanitizePath(tc.path); err == nil {
				t.Errorf("sanitizePath(%q) = %q, want error", tc.path, got)
			}
		})
	}
}

// TestSanitizePathAcceptsDotsInNames guards against the obvious over-broad
// fix: rejecting the substring ".." would break legitimate filenames. The
// check is component-based for exactly this reason.
func TestSanitizePathAcceptsDotsInNames(t *testing.T) {
	app := NewApp()

	accepted := []struct {
		path string
		want string
	}{
		{"/tmp/foo..bar.txt", "/tmp/foo..bar.txt"},
		{"/tmp/..hidden", "/tmp/..hidden"},
		{"/tmp/archive..tar.gz", "/tmp/archive..tar.gz"},
		// Clean normalizes redundant separators and single dots; neither is
		// traversal, so both must survive.
		{"/tmp//foo.txt", "/tmp/foo.txt"},
		{"/tmp/./foo.txt", "/tmp/foo.txt"},
	}
	for _, tc := range accepted {
		got, err := app.sanitizePath(tc.path)
		if err != nil {
			t.Errorf("sanitizePath(%q) rejected a legitimate path: %v", tc.path, err)
			continue
		}
		if got != tc.want {
			t.Errorf("sanitizePath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestValidatePathForEditorRequiresExistingFile verifies the existence check
// layered on top of sanitization: an editor must never be launched against a
// path that is not there.
func TestValidatePathForEditorRequiresExistingFile(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	real := filepath.Join(dir, "real.go")
	if err := os.WriteFile(real, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := app.validatePathForEditor(real)
	if err != nil {
		t.Fatalf("validatePathForEditor(%q) failed: %v", real, err)
	}
	if got != real {
		t.Errorf("validatePathForEditor(%q) = %q, want the cleaned input", real, got)
	}

	missing := filepath.Join(dir, "gone.go")
	if _, err := app.validatePathForEditor(missing); err == nil {
		t.Errorf("validatePathForEditor(%q) accepted a nonexistent file", missing)
	}

	// Traversal must be rejected before the stat, so a traversing path that
	// WOULD resolve to an existing file is still refused. Built by string
	// concatenation, not filepath.Join — Join cleans, which would resolve the
	// ".." away before sanitizePath ever sees it.
	traversing := dir + string(filepath.Separator) + ".." + string(filepath.Separator) +
		filepath.Base(dir) + string(filepath.Separator) + "real.go"
	if _, err := app.validatePathForEditor(traversing); err == nil {
		t.Errorf("validatePathForEditor(%q) accepted a traversing path to an existing file", traversing)
	}
}

// TestValidatePathForShowInFolderReturnsParentDir verifies the reveal path
// resolves to the file's PARENT directory (that is what a file manager is
// pointed at) and that the parent must exist.
func TestValidatePathForShowInFolderReturnsParentDir(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()

	target := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(target, []byte("x\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := app.validatePathForShowInFolder(target)
	if err != nil {
		t.Fatalf("validatePathForShowInFolder(%q) failed: %v", target, err)
	}
	if got != dir {
		t.Errorf("validatePathForShowInFolder(%q) = %q, want the parent %q", target, got, dir)
	}

	// The file itself need not exist — revealing a since-deleted file should
	// still open its folder — but the folder must.
	if _, err := app.validatePathForShowInFolder(filepath.Join(dir, "deleted.txt")); err != nil {
		t.Errorf("validatePathForShowInFolder rejected an existing dir with a missing file: %v", err)
	}
	if _, err := app.validatePathForShowInFolder(filepath.Join(dir, "nope", "file.txt")); err == nil {
		t.Error("validatePathForShowInFolder accepted a nonexistent parent directory")
	}
	if _, err := app.validatePathForShowInFolder(""); err == nil {
		t.Error("validatePathForShowInFolder accepted an empty path")
	}
}

// TestLookUpEditorResolvesToAbsolutePath verifies the TOCTOU-closing contract:
// the resolved absolute path is returned so callers exec THAT rather than
// re-resolving the name after the check.
func TestLookUpEditorResolvesToAbsolutePath(t *testing.T) {
	app := NewApp()

	// "go" is on PATH by definition — these tests are run by the go tool.
	path, err := app.lookUpEditor("go")
	if err != nil {
		t.Fatalf("lookUpEditor(\"go\") failed: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("lookUpEditor(\"go\") = %q, want an absolute path", path)
	}

	if _, err := app.lookUpEditor("definitely-not-a-real-editor-9f3a"); err == nil {
		t.Error("lookUpEditor accepted a command that is not on PATH")
	}
}

// TestAppendPathNeverMutatesSharedArgs guards the bug the function exists to
// prevent: the args slices come from the package-level editorCatalog and are
// shared across every launch, so appending in place would corrupt concurrent
// OpenInEditorByName calls whenever the slice had spare capacity.
func TestAppendPathNeverMutatesSharedArgs(t *testing.T) {
	// Spare capacity is the precondition for in-place append to be visible.
	shared := make([]string, 1, 4)
	shared[0] = "--flag"

	first := appendPath(shared, "/tmp/a.go")
	second := appendPath(shared, "/tmp/b.go")

	if len(shared) != 1 || shared[0] != "--flag" {
		t.Errorf("appendPath mutated the shared args slice: %v", shared)
	}
	if first[len(first)-1] != "/tmp/a.go" {
		t.Errorf("first call lost its path: %v", first)
	}
	if second[len(second)-1] != "/tmp/b.go" {
		t.Errorf("second call lost its path: %v", second)
	}
	// The decisive check: two calls off one base must not share a backing
	// array, or the second overwrites the first.
	if first[len(first)-1] == second[len(second)-1] {
		t.Errorf("appendPath returned slices sharing a backing array: %v vs %v", first, second)
	}
}

// TestIsSymbolSupportedExtension covers symbol_scan.go's predicate, including
// the case-insensitivity the comment promises and the five languages added
// alongside the original set.
func TestIsSymbolSupportedExtension(t *testing.T) {
	supported := []string{
		"a.go", "b.ts", "c.tsx", "d.js", "e.vue",
		"f.py", "g.rs", "h.java", "i.cs", "j.rb",
		// Case-insensitive per the doc comment.
		"K.GO", "L.Py", "M.JAVA",
		// Extension is taken from the final segment, so a path prefix and a
		// compound name must not confuse it.
		"/deep/dir/n.py", "min.bundle.js",
	}
	for _, path := range supported {
		if !isSymbolSupportedExtension(path) {
			t.Errorf("isSymbolSupportedExtension(%q) = false, want true", path)
		}
	}

	unsupported := []string{"a.txt", "b.md", "c.json", "d.png", "noextension", "e.pyc", ""}
	for _, path := range unsupported {
		if isSymbolSupportedExtension(path) {
			t.Errorf("isSymbolSupportedExtension(%q) = true, want false", path)
		}
	}
}

// TestSymbolCacheKeyNormalizes covers the fix for duplicate cache entries: two
// spellings of one directory must map to one key, or a single tree burns
// multiple slots of the 8-entry budget and evicts live entries.
func TestSymbolCacheKeyNormalizes(t *testing.T) {
	dir := t.TempDir()

	canonical := symbolCacheKey(dir)
	if !filepath.IsAbs(canonical) {
		t.Fatalf("symbolCacheKey(%q) = %q, want an absolute path", dir, canonical)
	}

	// Redundant separators, a "." segment, and a trailing separator are all
	// the same directory.
	variants := []string{
		dir + string(filepath.Separator),
		filepath.Join(dir, "."),
		dir + string(filepath.Separator) + string(filepath.Separator),
	}
	for _, v := range variants {
		if got := symbolCacheKey(v); got != canonical {
			t.Errorf("symbolCacheKey(%q) = %q, want %q", v, got, canonical)
		}
	}
}
