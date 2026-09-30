package main

import "regexp"

// SymbolInfo represents a symbol found in a source file.
type SymbolInfo struct {
	Name      string `json:"name"`      // Name of the symbol
	Type      string `json:"type"`      // Type: function, class, variable, const, etc.
	Line      int    `json:"line"`      // Line number where symbol is defined (1-indexed)
	File      string `json:"file"`      // File path containing the symbol
	Signature string `json:"signature"` // Full signature/declaration line
}

// SymbolProgressFunc receives incremental scan progress: how many source files
// have been processed, the total to process, and the file currently scanned.
type SymbolProgressFunc func(processed, total int, currentFile string)

// patternConfig holds regex patterns for symbol extraction per language
type patternConfig struct {
	regex     *regexp.Regexp
	nameIndex int
	keyword   string
}
