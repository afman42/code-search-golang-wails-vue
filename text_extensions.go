package main

import (
	"path/filepath"
	"strings"
)

// knownTextExtensions is the set of file extensions that are always text and
// never need the 512-byte binary detection probe. This covers the common
// programming languages, markup formats, config files, and documentation
// formats that a code-search tool encounters. By skipping the binary probe
// for these extensions we avoid an open+read+close syscall per file — on a
// tree of 2000 .go/.ts/.js files that's 2000 fewer syscalls in the
// collection phase (#3 of the high-impact collection optimizations).
//
// The set is intentionally conservative: only extensions that are
// universally text (never binary) are listed. Edge cases like .json (which
// could theoretically contain binary in a malformed file) are included
// because in practice JSON is always text. If a user explicitly sets
// IncludeBinary=true, the set is not consulted at all.
var knownTextExtensions = map[string]bool{
	// Programming languages
	".go":   true,
	".rs":   true,
	".py":   true,
	".pyw":  true,
	".js":   true,
	".mjs":  true,
	".cjs":  true,
	".ts":   true,
	".tsx":  true,
	".jsx":  true,
	".java": true,
	".kt":   true,
	".kts":  true,
	".scala": true,
	".groovy": true,
	".gradle": true,
	".c":    true,
	".h":    true,
	".cpp":  true,
	".cxx":  true,
	".cc":   true,
	".hpp":  true,
	".hxx":  true,
	".cs":   true,
	".rb":   true,
	".php":  true,
	".phtml": true,
	".swift": true,
	".m":    true,
	".mm":   true,
	".dart": true,
	".lua":  true,
	".pl":   true,
	".pm":   true,
	".r":    true,
	".jl":   true,
	".ex":   true,
	".exs":  true,
	".erl":  true,
	".hrl":  true,
	".clj":  true,
	".cljs": true,
	".cljc": true,
	".edn":  true,
	".elm":  true,
	".hs":   true,
	".lhs":  true,
	".ml":   true,
	".mli":  true,
	".nim":  true,
	".v":    true,
	".sv":   true,
	".vhd":  true,
	".vhdl": true,
	".asm":  true,
	".s":    true,
	".f":    true,
	".f90":  true,
	".f95":  true,
	".f03":  true,
	".for":  true,
	".pas":  true,
	".pp":   true,
	".d":    true,
	".zig":  true,
	".cr":   true,

	// Shells and scripting
	".sh":    true,
	".bash":  true,
	".zsh":   true,
	".fish":  true,
	".ps1":   true,
	".psm1":  true,
	".psd1":  true,
	".bat":   true,
	".cmd":   true,
	".awk":   true,
	".sed":   true,
	".vim":   true,
	".tcl":   true,
	".exp":   true,
	".wish":  true,
	".cgi":   true,
	".rpy":   true,

	// Web markup and style
	".html":  true,
	".htm":   true,
	".xhtml": true,
	".css":   true,
	".scss":  true,
	".sass":  true,
	".less":  true,
	".styl":  true,
	".stylus": true,
	".vue":   true,
	".svelte": true,
	".astro": true,
	".svg":   true, // SVG is XML (text), not a raster image
	".xml":   true,
	".xsl":   true,
	".xslt":  true,
	".dtd":   true,
	".rng":   true,

	// Data and config (always text in practice)
	".json":   true,
	".json5":  true,
	".jsonc":  true,
	".yaml":   true,
	".yml":    true,
	".toml":   true,
	".ini":    true,
	".cfg":    true,
	".conf":   true,
	".config": true,
	".properties": true,
	".env":    true,
	".editorconfig": true,
	".gitignore": true,
	".gitattributes": true,
	".dockerignore": true,
	".envrc":  true,

	// Documentation
	".md":    true,
	".markdown": true,
	".mdx":   true,
	".rst":   true,
	".txt":   true,
	".tex":   true,
	".latex": true,
	".adoc":  true,
	".asciidoc": true,
	".org":   true,
	".pod":   true,
	".man":   true,
	".roff":  true,
	".1":     true,
	".2":     true,
	".3":     true,
	".4":     true,
	".5":     true,
	".6":     true,
	".7":     true,
	".8":     true,
	".9":     true,

	// Build and project files
	".mk":    true,
	".makefile": true,
	".cmake": true,
	".gemspec": true,
	".podspec": true,
	".rake":  true,
	".thor":  true,
	".rakefile": true,
	".dockerfile": true,
	".containerfile": true,
	".jenkinsfile": true,

	// Query and database (text formats)
	".sql":   true,
	".psql":  true,
	".mysql": true,
	".graphql": true,
	".gql":   true,
	".prisma": true,

	// Other text formats
	".csv":   true,
	".tsv":   true,
	".log":   true,
	".diff":  true,
	".patch": true,
	".rej":   true,
	".lock":  true, // lockfiles are text (npm, yarn, cargo, etc.)
	".sum":   true, // go.sum, etc.
	".mod":   true, // go.mod, etc.
	".work":  true, // go.work
	".proto": true,
	".thrift": true,
	".avsc":  true,
	".wasm":  false, // explicitly NOT text
	".wat":   true,  // WAT is text
}

// isKnownTextExtension reports whether the file at the given path has an
// extension that is universally text and therefore does NOT need the binary
// detection probe. The check is case-insensitive (both .GO and .go match).
//
// Returns false for any extension not in the set, so unknown extensions
// still get the binary probe — the safe default. Only well-known text
// extensions skip the probe.
func isKnownTextExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return knownTextExtensions[ext]
}
