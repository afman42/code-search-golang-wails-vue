package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Nested .gitignore coverage. The root-level helpers (loadGitignoreMatcher,
// filterByGitignore, splitIgnoreLines) are covered by gitignore_test.go; these
// tests drive ignoreStack, which is what collection actually uses.

// writeIgnoreTree materializes a map of relative path -> content under a temp
// dir, creating parent directories as needed. Empty content still creates the
// file, which matters for a .gitignore whose rules are all comments.
func writeIgnoreTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

// collectedNames runs the real collection and returns the base names it kept,
// which is what a caller can actually observe.
func collectedNames(t *testing.T, root string, respectGitignore bool) map[string]bool {
	t.Helper()
	app := NewApp()
	req := SearchRequest{
		Directory:        root,
		Query:            "x",
		MaxFileSize:      1 << 20,
		MaxResults:       1000,
		RespectGitignore: respectGitignore,
	}
	files, err := app.collectFilesToProcess(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	names := make(map[string]bool, len(files))
	for _, f := range files {
		rel, relErr := filepath.Rel(root, f.absPath)
		if relErr != nil {
			t.Fatalf("rel %s: %v", f.absPath, relErr)
		}
		names[filepath.ToSlash(rel)] = true
	}
	return names
}

// TestNestedGitignoreDeeperFileOverridesAncestor is the feature's core claim: a
// nested .gitignore's rules win over a shallower one. Before nested support the
// root rule applied everywhere and sub/keep.log was dropped.
func TestNestedGitignoreDeeperFileOverridesAncestor(t *testing.T) {
	root := writeIgnoreTree(t, map[string]string{
		".gitignore":     "*.log\n",
		"main.go":        "package main\n",
		"drop.log":       "noise\n",
		"sub/.gitignore": "!keep.log\n",
		"sub/keep.log":   "wanted\n",
		"sub/other.log":  "noise\n",
	})

	got := collectedNames(t, root, true)

	if !got["sub/keep.log"] {
		t.Error("nested negation did not re-include sub/keep.log")
	}
	if got["drop.log"] {
		t.Error("root *.log rule failed to drop drop.log")
	}
	if got["sub/other.log"] {
		t.Error("root *.log rule failed to drop sub/other.log (nested file only negated keep.log)")
	}
	if !got["main.go"] {
		t.Error("main.go was dropped by an unrelated rule")
	}
}

// TestNestedGitignorePatternsAreRelativeToOwnDirectory guards the bug the old
// implementation had: patterns were matched against the path relative to the
// SEARCH ROOT, so a nested "build/" only worked at the root. Here the nested
// rule names a path that exists at both levels, and only the nested one may
// take effect.
func TestNestedGitignorePatternsAreRelativeToOwnDirectory(t *testing.T) {
	root := writeIgnoreTree(t, map[string]string{
		"pkg/.gitignore":     "generated.go\n",
		"generated.go":       "root level, not covered by pkg rule\n",
		"pkg/generated.go":   "covered\n",
		"pkg/real.go":        "package pkg\n",
		"other/generated.go": "different subtree, not covered\n",
	})

	got := collectedNames(t, root, true)

	if got["pkg/generated.go"] {
		t.Error("pkg/.gitignore did not apply to its own directory")
	}
	if !got["generated.go"] {
		t.Error("pkg/.gitignore leaked upward and dropped the root generated.go")
	}
	if !got["other/generated.go"] {
		t.Error("pkg/.gitignore leaked sideways and dropped other/generated.go")
	}
	if !got["pkg/real.go"] {
		t.Error("pkg/real.go was dropped")
	}
}

// TestNestedGitignorePrunesIgnoredDirectory verifies the directory-pruning path
// (ignoresDir): a plain directory rule drops the whole subtree, including a
// nested .gitignore inside it, mirroring git's rule that nothing under an
// excluded directory can be re-included.
func TestNestedGitignorePrunesIgnoredDirectory(t *testing.T) {
	root := writeIgnoreTree(t, map[string]string{
		".gitignore":              "node_modules/\n",
		"app.go":                  "package main\n",
		"node_modules/dep.go":     "vendored\n",
		"node_modules/.gitignore": "!dep.go\n",
	})

	got := collectedNames(t, root, true)

	if got["node_modules/dep.go"] {
		t.Error("ignored directory was descended; nested negation must not re-include inside it")
	}
	if !got["app.go"] {
		t.Error("app.go was dropped")
	}
}

// TestNestedGitignoreContentsRuleKeepsNegatedFile covers the case where pruning
// must be ABANDONED: "build/*" plus "!build/keep.txt" excludes the contents but
// keeps one file, and go-gitignore's regex for "build/*" also matches the bare
// directory. Pruning there would swallow the re-included file.
func TestNestedGitignoreContentsRuleKeepsNegatedFile(t *testing.T) {
	root := writeIgnoreTree(t, map[string]string{
		".gitignore":      "build/*\n!build/keep.txt\n",
		"build/keep.txt":  "kept\n",
		"build/junk.txt":  "dropped\n",
		"build/out/bin.s": "dropped\n",
	})

	got := collectedNames(t, root, true)

	if !got["build/keep.txt"] {
		t.Error("negated file inside a contents-excluded directory was dropped")
	}
	if got["build/junk.txt"] {
		t.Error("build/* failed to drop build/junk.txt")
	}
}

// TestGitignoreOffReadsNoIgnoreFiles verifies the gate: with RespectGitignore
// false, ignore files are not consulted at all, so a rule that would otherwise
// drop a file has no effect. This is the byte-identical-to-before guarantee.
func TestGitignoreOffReadsNoIgnoreFiles(t *testing.T) {
	root := writeIgnoreTree(t, map[string]string{
		".gitignore":     "*.log\n",
		"sub/.gitignore": "*.go\n",
		"keep.log":       "noise\n",
		"sub/keep.go":    "package sub\n",
	})

	got := collectedNames(t, root, false)

	if !got["keep.log"] || !got["sub/keep.go"] {
		t.Errorf("ignore rules applied while RespectGitignore=false: %v", got)
	}
}

// TestIgnoreStackReadsEachIgnoreFileOnce is the cost-model guard. The stack
// memoizes per directory, so a directory holding many files must not re-read
// its .gitignore per file. Asserted through the memo map: after testing several
// files in one directory, exactly the directories on their chains are cached.
func TestIgnoreStackReadsEachIgnoreFileOnce(t *testing.T) {
	root := writeIgnoreTree(t, map[string]string{
		".gitignore":     "*.log\n",
		"sub/.gitignore": "!keep.log\n",
	})

	stack := newIgnoreStack(root)
	if len(stack.chains) != 0 {
		t.Fatalf("newIgnoreStack read %d chains eagerly; construction must read nothing", len(stack.chains))
	}

	sub := filepath.Join(root, "sub")
	for _, name := range []string{"a.log", "b.log", "c.go"} {
		stack.ignoresFile(filepath.Join(sub, name))
	}

	// Only root and sub are on the chain, regardless of how many files were
	// tested under sub.
	if len(stack.chains) != 2 {
		t.Errorf("chains = %d entries after 3 files in one directory, want 2 (root + sub): %v",
			len(stack.chains), stack.chains)
	}
}

// TestCollectionFingerprintCoversNestedGitignore verifies cache invalidation:
// editing a nested .gitignore must change the fingerprint, or a stale cached
// collection would keep serving files the new rule excludes.
func TestCollectionFingerprintCoversNestedGitignore(t *testing.T) {
	root := writeIgnoreTree(t, map[string]string{
		"sub/.gitignore": "*.log\n",
		"sub/app.go":     "package sub\n",
	})

	before := computeCollectionFingerprint(root)

	nested := filepath.Join(root, "sub", ".gitignore")
	if err := os.WriteFile(nested, []byte("*.log\n*.tmp\n"), 0644); err != nil {
		t.Fatalf("rewrite nested ignore: %v", err)
	}

	if after := computeCollectionFingerprint(root); after == before {
		t.Error("editing sub/.gitignore did not change the fingerprint; a stale cache entry would survive")
	}
}
