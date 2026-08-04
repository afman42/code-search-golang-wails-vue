// Package main implements symbol extraction for the code search application.
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GetSymbolType determines the type of symbol based on its declaration pattern.
func GetSymbolType(keyword, signature string) string {
	lowerSig := strings.ToLower(signature)

	if strings.Contains(lowerSig, "func ") || strings.Contains(lowerSig, "function ") ||
		strings.Contains(lowerSig, "def ") || strings.Contains(lowerSig, "method ") {
		return "function"
	}
	if keyword != "" && (keyword == "class" || keyword == "struct" || keyword == "interface") {
		return "class"
	}
	// Empty keyword means it's a var/let/const declaration we detected via regex
	// "const X = 1" should be "variable", not "const"
	if keyword == "" {
		if strings.Contains(lowerSig, "const ") || strings.Contains(lowerSig, "let ") ||
			strings.Contains(lowerSig, "var ") {
			return "variable"
		}
		return "symbol"
	}
	if keyword == "const" {
		return "const"
	}
	if strings.Contains(lowerSig, "var ") || strings.Contains(lowerSig, "let ") {
		return "variable"
	}
	return "symbol"
}

// GetAllSymbols scans all supported source files in a directory and extracts
// symbols. Returns up to maxResults symbols. Supports Go, TypeScript, and Vue.
func GetAllSymbols(directory string, maxResults int) []SymbolInfo {
	return GetAllSymbolsWithProgress(directory, maxResults, nil)
}

// GetAllSymbolsWithProgress is GetAllSymbols with an optional progress callback.
// It collects the supported source files first so `total` is known up front,
// then extracts symbols file by file, invoking progress (when non-nil) after
// each file. Passing a nil callback makes it behave exactly like GetAllSymbols.
func GetAllSymbolsWithProgress(directory string, maxResults int, progress SymbolProgressFunc) []SymbolInfo {
	if maxResults <= 0 {
		maxResults = 1000
	}

	extensions := []string{".go", ".ts", ".tsx", ".js", ".vue"}
	isSupported := func(path string) bool {
		ext := strings.ToLower(filepath.Ext(path))
		for _, e := range extensions {
			if ext == e {
				return true
			}
		}
		return false
	}

	// Pass 1: enumerate the supported files so total is known for progress.
	var files []string
	_ = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			dirName := strings.ToLower(info.Name())
			if dirName == "node_modules" || dirName == ".git" || dirName == "vendor" ||
				dirName == "build" || dirName == "dist" || dirName == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if isSupported(path) {
			files = append(files, path)
		}
		return nil
	})

	total := len(files)
	var symbols []SymbolInfo

	// Pass 2: extract, reporting progress after each file.
	for i, path := range files {
		ext := strings.ToLower(filepath.Ext(path))
		symbols = append(symbols, extractSymbolsFromFile(path, ext)...)
		if progress != nil {
			progress(i+1, total, path)
		}
		if len(symbols) >= maxResults {
			break
		}
	}

	if len(symbols) > maxResults {
		result := make([]SymbolInfo, maxResults)
		copy(result, symbols[:maxResults])
		return result
	}
	result := make([]SymbolInfo, len(symbols))
	copy(result, symbols)
	return result
}

// SearchSymbols searches for symbols matching a name pattern.
// Case-insensitive search across all supported source files.
// Returns up to maxResults matches.
func SearchSymbols(name string, directory string, maxResults int) []SymbolInfo {
	if maxResults <= 0 {
		maxResults = 1000
	}

	if name == "" {
		return GetAllSymbols(directory, maxResults)
	}

	nameLower := strings.ToLower(name)
	matchedSigs := make(map[string]bool)

	// First pass: find signatures containing the search term
	allSymbols := GetAllSymbols(directory, maxResults*2)

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
		return nil
	}
	defer file.Close()

	var symbols []SymbolInfo
	scanner := bufio.NewScanner(file)
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

				// Deduplicate same-name symbols on consecutive lines
				if len(symbols) > 0 && symbols[len(symbols)-1].Name == name &&
					filepath.Base(symbols[len(symbols)-1].File) == filepath.Base(filePath) {
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

	return symbols
}

// getPatternsForExtension returns appropriate regex patterns for a given file extension.
func getPatternsForExtension(ext string) []patternConfig {
	switch ext {
	case ".go":
		return []patternConfig{
			// Function/method: func (r Receiver) FuncName(params)
			{
				regex:     regexp.MustCompile(`^\s*func\s+(?:\([^)]+\)\s+)?([A-Z]\w*)\s*\(`),
				nameIndex: 1,
				keyword:   "func",
			},
			// Struct: type Name struct {
			{
				regex:     regexp.MustCompile(`^\s*type\s+([A-Z]\w+)\s+struct\s*\{`),
				nameIndex: 1,
				keyword:   "struct",
			},
			// Interface: type Name interface {
			{
				regex:     regexp.MustCompile(`^\s*type\s+([A-Z]\w+)\s+interface\s*\{`),
				nameIndex: 1,
				keyword:   "interface",
			},
			// Const: const Name Type =
			{
				regex:     regexp.MustCompile(`^[\t ]*const\s+([A-Z][a-zA-Z0-9_]*)\s*=`),
				nameIndex: 1,
				keyword:   "const",
			},
			// Var: var Name Type =
			{
				regex:     regexp.MustCompile(`^[\t ]*var\s+([A-Z]\w+)\s+`),
				nameIndex: 1,
				keyword:   "var",
			},
		}
	case ".ts", ".tsx", ".js":
		return []patternConfig{
			// Export function: export function Foo( or export async function Foo(
			{
				regex:     regexp.MustCompile(`export\s+(?:async\s+)?function\s+([a-zA-Z][a-zA-Z0-9_]*)\s*\(`),
				nameIndex: 1,
				keyword:   "function",
			},
			// Standalone function (after newline): function fooName(params)
			{
				regex: regexp.MustCompile(`^[;
{}	]*\s*function\s+([a-zA-Z][a-zA-Z0-9_]*)\s*\(`),
				nameIndex: 1,
				keyword:   "function",
			},
			// Standalone function at start/after semicolon/newline: ; function Foo(
			{
				regex:     regexp.MustCompile(`^[;\n{}\t]*\s*function\s+([A-Z]\w*)\s*\(`),
				nameIndex: 1,
				keyword:   "function",
			},
			// Export const/let/var: export const NAME =
			{
				regex:     regexp.MustCompile(`export\s+(?:const|let|var)\s+([A-Z]\w*)\s*=`),
				nameIndex: 1,
				keyword:   "const",
			},
			// Standalone const/let/var without export: NAME: Type = or NAME[] = or NAME =
			{
				regex:     regexp.MustCompile(`^[;\n{}\t]*\s*(?:const|let|var)\s+([A-Z]\w*)\s*[:=\[]`),
				nameIndex: 1,
				keyword:   "const",
			},
			// Class: class Name or export class Name
			{
				regex:     regexp.MustCompile(`(?:export\s+)?class\s+([A-Z]\w*)\s*(?:extends|implements|{)`),
				nameIndex: 1,
				keyword:   "class",
			},
			// Interface: interface Name or export interface Name
			{
				regex:     regexp.MustCompile(`(?:export\s+)?interface\s+([A-Z]\w*)\s*(?:extends|{)`),
				nameIndex: 1,
				keyword:   "interface",
			},
			// Type alias: type Name =
			{
				regex:     regexp.MustCompile(`type\s+([A-Z]\w*)\s*=`),
				nameIndex: 1,
				keyword:   "type",
			},
		}
	case ".vue":
		return []patternConfig{
			// Script section function: function fooName( or async function fooName(
			{
				regex:     regexp.MustCompile(`(?:function|async\s+function)\s+([a-zA-Z][a-zA-Z0-9_]*)\s*\(`),
				nameIndex: 1,
				keyword:   "function",
			},
			// Arrow const: const foo = () => or const MyFunc = () =>
			{
				regex:     regexp.MustCompile(`const\s+([a-zA-Z][a-zA-Z0-9_]*)\s*=\s*[^=]*=>`),
				nameIndex: 1,
				keyword:   "const",
			},
			// Vue composables: const foo = ref() or const Bar = reactive()
			{
				regex:     regexp.MustCompile(`const\s+([a-zA-Z][a-zA-Z0-9_]*)\s*=\s*(ref|computed|reactive|toRef|toRefs)\s*\(`),
				nameIndex: 1,
				keyword:   "const",
			},
			// Template component usage: <ComponentName
			{
				regex:     regexp.MustCompile(`<([A-Z][a-zA-Z]*)(?:\s|$|/>|\>)`),
				nameIndex: 1,
				keyword:   "component",
			},
		}
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
