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
	// Every keyword introduced by a patternConfig must appear here — an
	// unmapped keyword silently falls through to signature sniffing, which
	// guesses from substrings. "type" (TS/Rust aliases) and "component"
	// (Vue) are deliberately absent: they already resolve to "symbol" via the
	// fallback and changing that would alter existing output.
	switch keyword {
	case "func", "function", "fn", "def", "method":
		return "function"
	case "class", "struct", "interface", "enum", "trait", "record", "module", "impl":
		return "class"
	case "const":
		return "const"
	case "var", "let", "property", "attr":
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
// symbols. Returns up to maxResults symbols. Supported languages are listed in
// symbolSupportedExtensions (symbol_scan.go).
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
				// Skip the private-by-convention leading underscore (Python
				// `_helper`, Rust `_unused`, C# `_field`). Dunders are the
				// exception: `__init__`/`__str__` are real Python API surface
				// and prime search targets, so a fully underscore-wrapped name
				// is kept. No existing-language pattern can produce a name
				// starting with `_` (every Go/TS/Vue pattern anchors its first
				// character on [A-Za-z] or [A-Z]), so this branch is
				// unreachable for .go/.ts/.tsx/.js/.vue either way.
				isDunder := strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")
				if name == "" || (strings.HasPrefix(name, "_") && !isDunder) {
					continue
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
	// Python. No export marker exists, so constants are matched by the
	// UPPER_CASE convention only — matching every module-level `x = 1` would
	// bury real symbols. Decorators sit on their own line, so decorated defs
	// are covered by the plain def pattern.
	pyPatterns = []patternConfig{
		// Function/method at any indent: def name( / async def name(
		{regex: regexp.MustCompile(`^\s*(?:async\s+)?def\s+([A-Za-z_]\w*)\s*\(`), nameIndex: 1, keyword: "def"},
		// Class: class Name( / class Name:
		{regex: regexp.MustCompile(`^\s*class\s+([A-Za-z_]\w*)\s*[(:]`), nameIndex: 1, keyword: "class"},
		// Module-level constant: NAME = value. `=[^=]` so `NAME ==` (a
		// comparison) is not read as a declaration.
		{regex: regexp.MustCompile(`^([A-Z][A-Z0-9_]*)\s*=[^=]`), nameIndex: 1, keyword: "const"},
		// Module-level annotated constant: NAME: Type [= value]. Same line as
		// the plain form when a value is present; the same-line dedup below
		// drops the duplicate.
		{regex: regexp.MustCompile(`^([A-Z][A-Z0-9_]*)\s*:\s*\S`), nameIndex: 1, keyword: "const"},
	}
	// Rust. Names are not required to be followed by `(` — `fn foo<T>(` puts
	// the generic list between name and parameters — so the function pattern
	// stops at the name.
	rustPatterns = []patternConfig{
		// Function: fn name / pub(crate) const async unsafe extern "C" fn name
		{regex: regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(?:const\s+)?(?:async\s+)?(?:unsafe\s+)?(?:extern\s+"[^"]*"\s+)?fn\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "fn"},
		// Struct: struct Name / pub struct Name<T>
		{regex: regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?struct\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "struct"},
		// Enum: enum Name / pub enum Name<T>
		{regex: regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?enum\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "enum"},
		// Trait: trait Name / pub unsafe trait Name
		{regex: regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(?:unsafe\s+)?trait\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "trait"},
		// Impl block: impl Name / impl<T> Trait for Name — the concrete type
		// is the useful symbol, so the `for` branch skips past the trait.
		{regex: regexp.MustCompile(`^\s*impl(?:<[^>]*>)?\s+(?:[\w:]+(?:<[^>]*>)?\s+for\s+)?([A-Za-z_]\w*)`), nameIndex: 1, keyword: "impl"},
		// Const/static: pub const NAME / static mut NAME. Uppercase-only, so
		// `const fn foo(` falls through to the function pattern above.
		{regex: regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?(?:const|static)\s+(?:mut\s+)?([A-Z][A-Z0-9_]*)\b`), nameIndex: 1, keyword: "const"},
		// Type alias: type Name = / pub type Name<T> =
		{regex: regexp.MustCompile(`^\s*(?:pub(?:\([^)]*\))?\s+)?type\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "type"},
	}
	// Java. Every pattern requires a leading modifier keyword. That is what
	// keeps the method pattern from matching `if (`, `while (` or `return
	// foo(` — a pattern that matched every call site would be worse than
	// missing package-private methods, which it does miss by design.
	javaPatterns = []patternConfig{
		// Declaration keywords come BEFORE the method pattern: the extraction
		// loop keeps the first match on a line and dedups the rest, and
		// `public record Point(int x)` matches the method shape too — leading
		// with `record` is what makes it a class instead of a function.
		// Class: [modifiers] class Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|protected|private|abstract|final|static|sealed|strictfp)\s+)*class\s+([A-Za-z_$]\w*)`), nameIndex: 1, keyword: "class"},
		// Interface: [modifiers] interface Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|protected|private|abstract|static|sealed)\s+)*interface\s+([A-Za-z_$]\w*)`), nameIndex: 1, keyword: "interface"},
		// Enum: [modifiers] enum Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|protected|private|static|final)\s+)*enum\s+([A-Za-z_$]\w*)`), nameIndex: 1, keyword: "enum"},
		// Record: [modifiers] record Name(
		{regex: regexp.MustCompile(`^\s*(?:(?:public|protected|private|static|final)\s+)*record\s+([A-Za-z_$]\w*)\s*\(`), nameIndex: 1, keyword: "record"},
		// Method or constructor: modifiers [generics] [return type] name(
		// The return-type group repeats so multi-token generic types
		// (`Map<String, List<X>> get(`) match, and is optional so
		// constructors (`public Foo(`) match. Lazy so the name is the token
		// immediately before `(`.
		{regex: regexp.MustCompile(`^\s*(?:public|protected|private|static|final|abstract|synchronized|native|default|strictfp)\s+(?:(?:public|protected|private|static|final|abstract|synchronized|native|default|strictfp)\s+)*(?:<[^>]+>\s+)?(?:[\w$.\[\]<>,]+\s+)*?([A-Za-z_$]\w*)\s*\(`), nameIndex: 1, keyword: "method"},
	}
	// C#. Same modifier-anchored shape as Java. `namespace X` and `using X`
	// are deliberately unmatchable: neither word is in any modifier set, and
	// no pattern here accepts a bare leading identifier.
	csPatterns = []patternConfig{
		// Declaration keywords first, same reason as javaPatterns.
		// Class: [modifiers] class Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|static|sealed|abstract|partial|unsafe|new)\s+)*class\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "class"},
		// Interface: [modifiers] interface Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|partial|new)\s+)*interface\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "interface"},
		// Struct: [modifiers] struct Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|partial|readonly|ref|unsafe|new)\s+)*struct\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "struct"},
		// Enum: [modifiers] enum Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|new)\s+)*enum\s+([A-Za-z_]\w*)`), nameIndex: 1, keyword: "enum"},
		// Record: [modifiers] record Name / record struct Name
		{regex: regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|sealed|abstract|partial|readonly)\s+)*record\s+(?:struct\s+|class\s+)?([A-Za-z_]\w*)`), nameIndex: 1, keyword: "record"},
		// Property: modifiers Type Name { get ... } or Name => expr. A
		// property whose `{ get` sits on the next line is missed — the
		// scanner is line-at-a-time.
		{regex: regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|static|virtual|override|abstract|sealed|required|readonly|new)\s+)+[\w$.\[\]<>,?]+\s+([A-Za-z_]\w*)\s*(?:\{\s*get|=>)`), nameIndex: 1, keyword: "property"},
		// Method or constructor: modifiers [generics] [return type] name(
		{regex: regexp.MustCompile(`^\s*(?:public|private|protected|internal|static|virtual|override|abstract|sealed|async|extern|partial|new|unsafe|readonly)\s+(?:(?:public|private|protected|internal|static|virtual|override|abstract|sealed|async|extern|partial|new|unsafe|readonly)\s+)*(?:<[^>]+>\s+)?(?:[\w$.\[\]<>,?]+\s+)*?([A-Za-z_]\w*)\s*\(`), nameIndex: 1, keyword: "method"},
	}
	// Ruby. Methods need no parentheses, so the def pattern stops at the
	// name. The trailing `?`/`!`/`=` IS captured: `valid?` and `save!` are the
	// real method names a user greps for, and a substring search for "valid"
	// still matches. Dropping them would report a name no one can jump to.
	rubyPatterns = []patternConfig{
		// Method: def name / def self.name / def name? / def name=(v)
		{regex: regexp.MustCompile(`^\s*def\s+(?:self\.)?([A-Za-z_]\w*[?!=]?)`), nameIndex: 1, keyword: "def"},
		// Class: class Name / class Name < Base. `[A-Z]` also rules out the
		// singleton-class form `class << self`.
		{regex: regexp.MustCompile(`^\s*class\s+([A-Z]\w*)`), nameIndex: 1, keyword: "class"},
		// Module: module Name
		{regex: regexp.MustCompile(`^\s*module\s+([A-Z]\w*)`), nameIndex: 1, keyword: "module"},
		// Attribute accessors: attr_accessor :name
		//
		// ponytail: only the FIRST symbol of a multi-attribute line
		// (`attr_accessor :a, :b`) is captured — the extraction loop takes one
		// submatch per pattern per line. Upgrade: FindAllStringSubmatch in
		// extractSymbolsFromFile, worth it only if multi-name declarations
		// prove to be a real miss in practice.
		{regex: regexp.MustCompile(`^\s*attr_(?:accessor|reader|writer)\s+:([A-Za-z_]\w*)`), nameIndex: 1, keyword: "attr"},
		// Top-level constant: NAME = value (Ruby constants are capitalized;
		// UPPER_CASE is the constant-as-value convention).
		{regex: regexp.MustCompile(`^([A-Z][A-Z0-9_]*)\s*=[^=]`), nameIndex: 1, keyword: "const"},
	}
)

// getPatternsForExtension returns appropriate regex patterns for a given file
// extension. Every case here must appear in symbolSupportedExtensions
// (symbol_scan.go) and vice versa — an extension in the slice with no case
// gets scanned and yields nothing.
func getPatternsForExtension(ext string) []patternConfig {
	switch ext {
	case ".go":
		return goPatterns
	case ".ts", ".tsx", ".js":
		return tsPatterns
	case ".vue":
		return vuePatterns
	case ".py":
		return pyPatterns
	case ".rs":
		return rustPatterns
	case ".java":
		return javaPatterns
	case ".cs":
		return csPatterns
	case ".rb":
		return rubyPatterns
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
