// Package main implements symbol extraction for the code search application.
package main

import (
	"bufio"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GetSymbolType determines the type of symbol based on its declaration pattern.
func GetSymbolType(keyword, signature string) string {
	// Keyword takes precedence: `var Foo func() int` contains "func " but is
	// a variable, not a function. Only fall back to signature sniffing when
	// the keyword is empty/unknown.
	switch keyword {
	case "func", "function":
		return "function"
	case "class", "struct", "interface":
		return "class"
	case "const":
		return "const"
	case "var", "let":
		return "variable"
	}

	lowerSig := strings.ToLower(signature)
	if strings.Contains(lowerSig, "func ") || strings.Contains(lowerSig, "function ") ||
		strings.Contains(lowerSig, "def ") || strings.Contains(lowerSig, "method ") {
		return "function"
	}
	if strings.Contains(lowerSig, "const ") || strings.Contains(lowerSig, "let ") ||
		strings.Contains(lowerSig, "var ") {
		return "variable"
	}
	return "symbol"
}

// GetAllSymbols scans all supported source files in a directory and extracts
// symbols. Returns up to maxResults symbols. Supports Go, TypeScript, and Vue.
func GetAllSymbols(directory string, maxResults int) []SymbolInfo {
	return GetAllSymbolsWithProgress(directory, maxResults, nil)
}

// getAllSymbolsUnbounded returns the FULL extracted symbol set for a
// directory, using the persistent index when available (the cache stores the
// complete set regardless of the first caller's maxResults). Callers that
// filter or truncate themselves (SearchSymbols) must use this — asking for
// maxResults up front would truncate before filtering and silently miss
// matches beyond the window.
func getAllSymbolsUnbounded(directory string, progress SymbolProgressFunc) []SymbolInfo {
	if globalSymbolIndex != nil {
		fp := computeDirectoryFingerprint(directory)
		if cached, ok := globalSymbolIndex.get(directory, fp); ok {
			return cached
		}
		// Cache miss: extract the FULL set (maxResults<=0 = unbounded) and
		// cache it, so a later larger request reads complete data instead of
		// a slice truncated to whatever the first caller asked for.
		full := extractAllSymbols(directory, 0, progress)
		globalSymbolIndex.set(directory, fp, full)
		return full
	}
	return extractAllSymbols(directory, 0, progress)
}

// GetAllSymbolsWithProgress is GetAllSymbols with an optional progress callback.
// It collects the supported source files first so `total` is known up front,
// then extracts symbols file by file, invoking progress (when non-nil) after
// each file. Passing a nil callback makes it behave exactly like GetAllSymbols.
func GetAllSymbolsWithProgress(directory string, maxResults int, progress SymbolProgressFunc) []SymbolInfo {
	if maxResults <= 0 {
		maxResults = 1000
	}

	// Truncate on read from the full set; never extract a truncated set into
	// the cache (a later larger request would miss symbols).
	full := getAllSymbolsUnbounded(directory, progress)
	if len(full) > maxResults {
		result := make([]SymbolInfo, maxResults)
		copy(result, full[:maxResults])
		return result
	}
	result := make([]SymbolInfo, len(full))
	copy(result, full)
	return result
}

// extractAllSymbols does the actual two-pass scan + extraction. Split out so
// the cache wrapper above stays readable.
func extractAllSymbols(directory string, maxResults int, progress SymbolProgressFunc) []SymbolInfo {

	// Pass 1: enumerate the supported files so total is known for progress.
	// Uses filepath.WalkDir (not filepath.Walk) for consistency with the rest
	// of the codebase — WalkDir does one Lstat per file instead of two, and
	// avoids allocating an os.FileInfo.
	var files []string
	_ = filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDirForSymbolScan(d) {
				return filepath.SkipDir
			}
			return nil
		}
		if isSymbolSupportedExtension(path) {
			files = append(files, path)
		}
		return nil
	})

	total := len(files)
	var symbols []SymbolInfo

	// Pass 2: extract, reporting progress after each file. Bounded by
	// maxSymbolScanFiles so an unbounded request (maxResults<=0, used by the
	// cache-miss path) cannot balloon memory on a huge tree.
	for i, path := range files {
		if i >= maxSymbolScanFiles {
			log.Printf("symbol scan truncated at %d files (directory too large)", maxSymbolScanFiles)
			break
		}
		ext := strings.ToLower(filepath.Ext(path))
		symbols = append(symbols, extractSymbolsFromFile(path, ext)...)
		if progress != nil {
			progress(i+1, total, path)
		}
		if maxResults > 0 && len(symbols) >= maxResults {
			break
		}
	}

	if maxResults > 0 && len(symbols) > maxResults {
		result := make([]SymbolInfo, maxResults)
		copy(result, symbols[:maxResults])
		return result
	}
	result := make([]SymbolInfo, len(symbols))
	copy(result, symbols)
	return result
}

// searchSymbols searches for symbols matching a name pattern.
// Case-insensitive search across all supported source files.
// Returns up to maxResults matches. Lowercase: the Wails binding is the App
// method SearchSymbols; a same-named package func is a footgun.
func searchSymbols(name string, directory string, maxResults int) []SymbolInfo {
	if maxResults <= 0 {
		maxResults = 1000
	}

	if name == "" {
		return GetAllSymbols(directory, maxResults)
	}

	nameLower := strings.ToLower(name)
	matchedSigs := make(map[string]bool)

	// Fetch the FULL symbol set, then filter, then truncate. Fetching a
	// pre-truncated set (e.g. maxResults*2) silently misses matches beyond
	// the window. The persistent index makes this cheap on repeat keystrokes.
	allSymbols := getAllSymbolsUnbounded(directory, nil)

	var results []SymbolInfo
	for _, sym := range allSymbols {
		if strings.Contains(strings.ToLower(sym.Name), nameLower) ||
			strings.Contains(strings.ToLower(sym.Signature), nameLower) {

			// Avoid duplicates
			key := sym.File + ":" + sym.Name
			if !matchedSigs[key] {
				matchedSigs[key] = true
				results = append(results, sym)

				if len(results) >= maxResults {
					break
				}
			}
		}
	}

	if results == nil {
		return []SymbolInfo{}
	}
	return results
}

// extractSymbolsFromFile parses a single source file and extracts symbol definitions.
func extractSymbolsFromFile(filePath string, extension string) []SymbolInfo {
	file, err := os.Open(filePath)
	if err != nil {
		// Silent nil here made unreadable files indistinguishable from empty
		// ones. Surface it (debug-level information only).
		log.Printf("symbol extraction: cannot open %s: %v", filePath, err)
		return nil
	}
	defer file.Close()

	var symbols []SymbolInfo
	scanner := bufio.NewScanner(file)
	// Default Scanner max token is 64KB: a single longer line (minified JS)
	// aborts the whole file and silently drops every symbol in it. 10MB
	// covers realistic minified files while bounding memory.
	const maxSymbolLineLen = 10 * 1024 * 1024 // 10MB
	scanner.Buffer(make([]byte, 64*1024), maxSymbolLineLen)
	lineNum := 0

	// Patterns for different languages
	patterns := getPatternsForExtension(extension)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, pattern := range patterns {
			matches := pattern.regex.FindStringSubmatch(line)
			if matches != nil {
				nameIdx := pattern.nameIndex
				if nameIdx < 0 || nameIdx >= len(matches) {
					nameIdx = 1 // Default to first capturing group
				}

				name := normalizeSymbolName(matches[nameIdx])
				if name == "" || strings.HasPrefix(name, "_") {
					continue // Skip private/internal symbols
				}

				// Deduplicate same-name symbols on the SAME line (pattern
				// overlap). Line is included so legitimate distinct symbols
				// like `type Foo` + `func Foo` on consecutive lines are both
				// kept.
				if len(symbols) > 0 && symbols[len(symbols)-1].Name == name &&
					filepath.Base(symbols[len(symbols)-1].File) == filepath.Base(filePath) &&
					symbols[len(symbols)-1].Line == lineNum {
					continue
				}

				symbol := SymbolInfo{
					Name:      name,
					Type:      GetSymbolType(pattern.keyword, line),
					Line:      lineNum,
					File:      filePath,
					Signature: strings.TrimSpace(line),
				}
				symbols = append(symbols, symbol)
			}
		}
	}

	// Surface a truncated scan (ErrTooLong beyond the 10MB cap, I/O error)
	// instead of returning silently-partial symbols as if the file were
	// complete. The standalone function has no logger; log.Printf is the
	// only sink available.
	if err := scanner.Err(); err != nil {
		log.Printf("symbol extraction truncated for %s: %v", filePath, err)
	}

	return symbols
}

// Precompiled regex patterns for symbol extraction per language. Compiled once
// at package init instead of every call to getPatternsForExtension (which was
// MustCompile on every file — 700k redundant compiles for 100k files).
var (
	goPatterns = []patternConfig{
		// Function/method: func (r Receiver) FuncName(params)
		{regex: regexp.MustCompile(`^\s*func\s+(?:\([^)]+\)\s+)?([A-Z]\w*)\s*\(`), nameIndex: 1, keyword: "func"},
		// Struct: type Name struct {
		{regex: regexp.MustCompile(`^\s*type\s+([A-Z]\w+)\s+struct\s*\{`), nameIndex: 1, keyword: "struct"},
		// Interface: type Name interface {
		{regex: regexp.MustCompile(`^\s*type\s+([A-Z]\w+)\s+interface\s*\{`), nameIndex: 1, keyword: "interface"},
		// Const: const Name [Type] = / const A, B = ... (typed consts, multi-name)
		{regex: regexp.MustCompile(`^[\t ]*const\s+([A-Z][a-zA-Z0-9_]*)\b`), nameIndex: 1, keyword: "const"},
		// Var: var Name [Type] = / var X = 5 (typed, multi-name, no-space)
		{regex: regexp.MustCompile(`^[\t ]*var\s+([A-Z]\w*)\b`), nameIndex: 1, keyword: "var"},
	}
	tsPatterns = []patternConfig{
		// Export function: export function Foo( or export async function Foo(
		{regex: regexp.MustCompile(`export\s+(?:async\s+)?function\s+([a-zA-Z][a-zA-Z0-9_]*)\s*\(`), nameIndex: 1, keyword: "function"},
		// Standalone function (after newline/semicolon): ; function Foo(
		{regex: regexp.MustCompile(`^[;\n{}\t]*\s*function\s+([a-zA-Z][a-zA-Z0-9_]*)\s*\(`), nameIndex: 1, keyword: "function"},
		// Export const/let/var: export const NAME =
		{regex: regexp.MustCompile(`export\s+(?:const|let|var)\s+([A-Z]\w*)\s*=`), nameIndex: 1, keyword: "const"},
		// Standalone const/let/var without export: NAME: Type = or NAME[] = or NAME =
		{regex: regexp.MustCompile(`^[;\n{}\t]*\s*(?:const|let|var)\s+([A-Z]\w*)\s*[:=\[]`), nameIndex: 1, keyword: "const"},
		// Class: class Name or export class Name
		{regex: regexp.MustCompile(`(?:export\s+)?class\s+([A-Z]\w*)\s*(?:extends|implements|{)`), nameIndex: 1, keyword: "class"},
		// Interface: interface Name or export interface Name
		{regex: regexp.MustCompile(`(?:export\s+)?interface\s+([A-Z]\w*)\s*(?:extends|{)`), nameIndex: 1, keyword: "interface"},
		// Type alias: type Name =
		{regex: regexp.MustCompile(`type\s+([A-Z]\w*)\s*=`), nameIndex: 1, keyword: "type"},
	}
	vuePatterns = []patternConfig{
		// Script section function: function fooName( or async function fooName(
		{regex: regexp.MustCompile(`(?:function|async\s+function)\s+([a-zA-Z][a-zA-Z0-9_]*)\s*\(`), nameIndex: 1, keyword: "function"},
		// Arrow const: const foo = () => or const MyFunc = () =>
		{regex: regexp.MustCompile(`const\s+([a-zA-Z][a-zA-Z0-9_]*)\s*=\s*[^=]*=>`), nameIndex: 1, keyword: "const"},
		// Vue composables: const foo = ref() or const Bar = reactive()
		{regex: regexp.MustCompile(`const\s+([a-zA-Z][a-zA-Z0-9_]*)\s*=\s*(ref|computed|reactive|toRef|toRefs)\s*\(`), nameIndex: 1, keyword: "const"},
		// Template component usage: <ComponentName
		{regex: regexp.MustCompile(`<([A-Z][a-zA-Z]*)(?:\s|$|/>|\>)`), nameIndex: 1, keyword: "component"},
	}
)

// getPatternsForExtension returns appropriate regex patterns for a given file extension.
func getPatternsForExtension(ext string) []patternConfig {
	switch ext {
	case ".go":
		return goPatterns
	case ".ts", ".tsx", ".js":
		return tsPatterns
	case ".vue":
		return vuePatterns
	default:
		return nil
	}
}

// normalizeSymbolName cleans up extracted symbol names.
func normalizeSymbolName(name string) string {
	// Remove leading/trailing whitespace
	name = strings.TrimSpace(name)

	// Handle edge cases like destructuring patterns
	if idx := strings.Index(name, "["); idx != -1 {
		name = name[:idx]
	}
	if idx := strings.Index(name, "("); idx != -1 {
		name = name[:idx]
	}

	return name
}
