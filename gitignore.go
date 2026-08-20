package main

import (
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// ---------------------------------------------------------------------------
// .gitignore support
//
// When SearchRequest.RespectGitignore is true, files matched by the search
// directory's root .gitignore and .git/info/exclude are dropped from
// collection. Matching delegates to go-gitignore, which implements the real
// gitignore rules (negation "!", "**", anchoring, dir-only patterns).
//
// ponytail: root-level only — nested per-directory .gitignore files are NOT
// honored. Upgrade: collect the applicable ignore stack per directory during
// the walk, only if nested-monorepo fidelity is actually needed.
// ---------------------------------------------------------------------------

// loadGitignoreMatcher builds a matcher from the directory's root .gitignore
// and .git/info/exclude. Missing files contribute no lines. Returns nil when
// no ignore source exists, so callers can skip matching entirely.
func loadGitignoreMatcher(directory string) *gitignore.GitIgnore {
	var lines []string

	root := filepath.Join(directory, ".gitignore")
	if bs, err := os.ReadFile(root); err == nil {
		lines = append(lines, splitIgnoreLines(bs)...)
	}

	exclude := filepath.Join(directory, ".git", "info", "exclude")
	if bs, err := os.ReadFile(exclude); err == nil {
		lines = append(lines, splitIgnoreLines(bs)...)
	}

	if len(lines) == 0 {
		return nil
	}
	return gitignore.CompileIgnoreLines(lines...)
}

// splitIgnoreLines splits ignore-file content on "\n", trimming trailing "\r"
// so CRLF ignore files parse identically to LF ones.
func splitIgnoreLines(bs []byte) []string {
	raw := string(bs)
	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

// filterByGitignore drops files whose path relative to the search root is
// matched by the ignore matcher. Returns the input slice if nothing is
// dropped (no allocation).
func filterByGitignore(files []fileMeta, directory string, matcher *gitignore.GitIgnore) []fileMeta {
	if matcher == nil {
		return files
	}
	out := make([]fileMeta, 0, len(files))
	for _, f := range files {
		rel, err := filepath.Rel(directory, f.absPath)
		if err != nil {
			continue // can't resolve — drop, same safe default as the walk
		}
		if matcher.MatchesPath(rel) {
			continue
		}
		out = append(out, f)
	}
	return out
}
