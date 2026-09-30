package main

// FileReplacement describes one line change in one file produced by a
// ReplaceInFiles dry-run or apply.
type FileReplacement struct {
	FilePath string `json:"filePath"`
	LineNum  int    `json:"lineNum"`
	OldLine  string `json:"oldLine"`
	NewLine  string `json:"newLine"`
}

// ReplaceRequest carries the search to reuse (its matches define what gets
// replaced) plus the literal replacement string. Apply=false computes a dry-run
// and writes nothing; Apply=true writes the changes atomically.
type ReplaceRequest struct {
	Search      SearchRequest `json:"search"`
	Replacement string        `json:"replacement"`
	Apply       bool          `json:"apply"`
}

// ReplaceResult is the outcome of a dry-run or apply. FilesChanged/LinesChanged
// count only lines whose replacement actually differs from the original.
type ReplaceResult struct {
	Files        []FileReplacement `json:"files"`
	FilesChanged int               `json:"filesChanged"`
	LinesChanged int               `json:"linesChanged"`
}
