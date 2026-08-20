package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// maxSymbolScanFiles bounds how many source files a single directory's symbol
// extraction will scan, mirroring maxCachedFiles in collection_index.go.
// Beyond this the scan is truncated (and logged) to bound memory on huge
// trees; the persistent cache never stores unbounded results.
const maxSymbolScanFiles = 200_000

// skipSymbolScanDirs is the set of directory names skipped during symbol
// extraction and fingerprinting. These are build outputs, dependency caches,
// and VCS metadata that never contain user-authored symbol definitions.
//
// This is the single source of truth for the symbol-scan skip list —
// previously duplicated in symbols.go (extractAllSymbols) and
// symbol_index.go (computeDirectoryFingerprint).
var skipSymbolScanDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	"build":        true,
	"dist":         true,
	"bin":          true,
}

// symbolSupportedExtensions is the set of file extensions that the symbol
// extractor can parse. Adding a new language requires only one entry here
// (plus the regex patterns in getPatternsForExtension).
//
// This is the single source of truth — previously duplicated in symbols.go
// and symbol_index.go.
var symbolSupportedExtensions = []string{".go", ".ts", ".tsx", ".js", ".vue"}

// isSymbolSupportedExtension reports whether the file at the given path has
// an extension that the symbol extractor can parse. Case-insensitive.
func isSymbolSupportedExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range symbolSupportedExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// shouldSkipDirForSymbolScan reports whether a directory entry should be
// skipped during symbol extraction / fingerprinting. It checks the
// skipSymbolScanDirs set by lowercased name.
func shouldSkipDirForSymbolScan(d fs.DirEntry) bool {
	return skipSymbolScanDirs[strings.ToLower(d.Name())]
}
