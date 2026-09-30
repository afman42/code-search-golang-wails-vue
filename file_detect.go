package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"sync"
)

// excludePatternCache memoizes the compiled form of user exclude patterns so
// the walk doesn't re-split / re-glob per file. Keyed by the raw pattern
// string: exact==true means a plain component/exact-path match, otherwise
// glob is the compiled filepath.Match pattern (valid==false when the pattern
// doesn't compile — then it only matches exactly).
type excludePattern struct {
	raw    string
	exact  string
	glob   string
	valid  bool
	isAbs  bool
	hasSep bool
}

var excludePatternCache sync.Map // map[string]excludePattern

func cachedExcludePattern(pattern string) excludePattern {
	if v, ok := excludePatternCache.Load(pattern); ok {
		return v.(excludePattern)
	}
	ep := excludePattern{raw: pattern, exact: pattern, hasSep: strings.ContainsRune(pattern, filepath.Separator)}
	if _, err := filepath.Match(pattern, ""); err == nil {
		ep.glob = pattern
		ep.valid = true
	}
	ep.isAbs = filepath.IsAbs(pattern)
	excludePatternCache.Store(pattern, ep)
	return ep
}

// matchExcludePattern reports whether one pre-split path component matches a
// compiled exclude pattern. Split out so rejectByExcludes can split once per
// file instead of once per pattern (matchesPattern did the split inside the
// per-pattern loop).
func matchExcludePattern(component string, ep excludePattern) bool {
	if component == ep.exact {
		return true
	}
	if ep.valid {
		if matched, err := filepath.Match(ep.glob, component); err == nil && matched {
			return true
		}
	}
	return false
}

// isTextByte reports whether a byte is considered printable text.
// Extracted from isBinary to name the complex conditional (Decompose Conditional)
// and allow isolated testing of the printable-character rule.
func isTextByte(b byte) bool {
	return (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' || b >= 127
}

// isBinary checks if content appears to be binary by looking for null bytes
// and a high proportion of non-text characters
func (a *App) isBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check for null bytes in the first 512 bytes using bytes.Contains so we
	// don't allocate a string just to scan for a single byte (#7). The
	// previous implementation did string(content[:min(512,len(content))])
	// on every file, which is a measurable cost on large corpora.
	checkLen := 512
	if len(content) < checkLen {
		checkLen = len(content)
	}
	if bytes.Contains(content[:checkLen], nullByte) {
		return true
	}

	// Count printable vs non-printable characters in first part of file
	// For UTF-8 text, we need to be more lenient as many Unicode characters have high bytes
	var printableCount int
	for i, b := range content {
		if i >= checkLen { // Only check first 512 bytes for performance
			break
		}
		if isTextByte(b) {
			printableCount++
		}
	}

	// If less than 50% of characters are printable, consider it binary.
	// The 0.5 threshold (not 0.7) is intentional: high-byte UTF-8 sequences
	// are counted as printable, so legitimate Unicode text stays above 0.5
	// while genuinely binary content (control bytes, structured data) falls
	// below it. See TestBinaryFileFiltering for the calibration anchor.
	return float64(printableCount)/float64(checkLen) < 0.5
}

// matchesPattern checks if a path matches an exclude pattern.
// It matches against individual path components so that patterns like "git"
// match ".git" but not "digits.txt" or "vigilant.go".
//
// Perf: kept for tests/callers, but the walk path (rejectByExcludes) uses
// matchComponents which splits once per file. This wrapper preserves the
// exact legacy semantics, including "" matching every path (Split always
// yields an empty component compare) — see TestMatchesPattern.
func (a *App) matchesPattern(path string, pattern string) bool {
	// First try exact match on the full path
	if path == pattern {
		return true
	}
	components := strings.Split(path, string(filepath.Separator))
	return matchComponents(components, cachedExcludePattern(pattern))
}

// matchComponents matches pre-split path components against a compiled
// exclude pattern. Splitting once per file instead of once per pattern turns
// rejectByExcludes from O(patterns * components) string work into one split
// + O(patterns * components) cheap compares.
func matchComponents(components []string, ep excludePattern) bool {
	for _, component := range components {
		if matchExcludePattern(component, ep) {
			return true
		}
	}
	return false
}

// compileExcludePatterns precompiles the walk's exclude list once: it drops
// empty patterns (matchesPattern only matched "" via the empty-component
// quirk, never intentionally) and caches the compiled glob validity.
func compileExcludePatterns(patterns []string) []excludePattern {
	if len(patterns) == 0 {
		return nil
	}
	out := make([]excludePattern, 0, len(patterns))
	for _, p := range patterns {
		if p == "" {
			continue
		}
		out = append(out, cachedExcludePattern(p))
	}
	return out
}

// buildAllowedSet normalizes the allow-list once per walk into a lowercased
// set (leading dot stripped), so the per-file check is O(1) instead of
// O(len(allowList)) matchExtension calls.
func buildAllowedSet(allowed []string) map[string]struct{} {
	if len(allowed) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(allowed))
	for _, e := range allowed {
		set[strings.ToLower(strings.TrimPrefix(e, "."))] = struct{}{}
	}
	return set
}

// matchExtensionNormalized checks base/path against an already-normalized
// (dot-stripped, lowercased) extension. Both the final extension and the
// compound extension (min.js, tar.gz) are accepted, case-insensitively.
func matchExtensionNormalized(base, path, normalizedExt string) bool {
	if normalizedExt == "" {
		return true
	}
	if strings.EqualFold(strings.TrimPrefix(filepath.Ext(base), "."), normalizedExt) {
		return true
	}
	return strings.EqualFold(strings.TrimPrefix(getFullExtension(base), "."), normalizedExt)
}

// matchExtensionSet checks base/path against a precomputed allow-list set.
func matchExtensionSet(base, path string, set map[string]struct{}) bool {
	if len(set) == 0 {
		return true
	}
	finalExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(base), "."))
	if _, ok := set[finalExt]; ok {
		return true
	}
	fullExt := strings.ToLower(strings.TrimPrefix(getFullExtension(base), "."))
	if _, ok := set[fullExt]; ok {
		return true
	}
	return false
}

// getFullExtension extracts the full extension from a file path
// For example: "file.min.js" returns ".min.js", "archive.tar.gz" returns ".tar.gz"
func getFullExtension(path string) string {
	base := filepath.Base(path)
	// If there's no dot, return empty string
	if !strings.Contains(base, ".") {
		return ""
	}

	// Find the first dot and return everything after it
	firstDotIndex := strings.Index(base, ".")
	if firstDotIndex == -1 {
		return ""
	}

	return base[firstDotIndex:]
}

// matchExtension checks if a file path matches an extension requirement
// This handles both single extensions (like "js") and full extensions (like "min.js", "tar.gz").
//
// The requestedExt may be written with or without a leading dot: "go" and
// ".go" both match main.go. The leading-dot form is what the UI's
// PatternSelector dropdown sends (and what a user naturally types), while
// GetKnownTextExtensions returns the dot-less form; accepting both keeps the
// allow-list filter working regardless of which convention the caller uses.
func matchExtension(path string, requestedExt string) bool {
	if requestedExt == "" {
		return true
	}

	// Normalize: a single leading dot is stripped so ".go" == "go".
	// (Only one dot, so compound extensions like ".tar.gz" are preserved
	// after stripping just the first.)
	requestedExt = strings.TrimPrefix(requestedExt, ".")

	// First try to match the final extension (current behavior for backward compatibility)
	finalExt := strings.TrimPrefix(filepath.Ext(path), ".")
	if strings.EqualFold(finalExt, requestedExt) {
		return true
	}

	// Then try to match the full extension sequence
	fullExt := strings.TrimPrefix(getFullExtension(path), ".")
	return strings.EqualFold(fullExt, requestedExt)
}
