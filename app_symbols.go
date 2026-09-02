// Package main implements the backend functionality for the code search application.
package main

import (
	"github.com/sirupsen/logrus"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetAllSymbols is a Wails binding that scans source files in the given
// directory and extracts symbol definitions (functions, classes, variables,
// constants, interfaces, types). Returns up to maxResults symbols.
//
// Supported languages: Go (.go), TypeScript (.ts/.tsx), JavaScript (.js),
// Vue (.vue), Python (.py), Rust (.rs), Java (.java), C# (.cs) and Ruby
// (.rb). Build outputs and dependency caches (node_modules, target,
// __pycache__, .venv, obj, …) are skipped automatically — symbol_scan.go
// holds the authoritative extension and skip lists.
//
// Returns an empty array (never nil) if no symbols are found.
func (a *App) GetAllSymbols(directory string, maxResults int) []SymbolInfo {
	a.logDebug("Extracting all symbols", logrus.Fields{
		"directory":  directory,
		"maxResults": maxResults,
	})

	if directory == "" {
		a.logDebug("Empty directory provided to GetAllSymbols", nil)
		return []SymbolInfo{}
	}

	// globalSymbolIndex is set once in NewApp; do not reassign here (data race).
	// Stream per-file scan progress to the frontend so the symbol panel can show
	// a real progress bar instead of a synthetic one. Guard on ctx: unit tests
	// construct App without a Wails runtime context.
	symbols := GetAllSymbolsWithProgress(directory, maxResults, func(processed, total int, currentFile string) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "symbol-progress", map[string]interface{}{
				"processed":   processed,
				"total":       total,
				"currentFile": currentFile,
			})
		}
	})

	a.logDebug("Symbol extraction complete", logrus.Fields{
		"directory":  directory,
		"count":      len(symbols),
		"maxResults": maxResults,
	})

	// Guarantee a non-nil empty slice for the frontend
	if symbols == nil {
		return []SymbolInfo{}
	}
	return symbols
}

// SearchSymbols is a Wails binding that searches for symbols whose name (or
// signature) matches the given query, case-insensitively. Returns up to
// maxResults matches. If name is empty, behaves like GetAllSymbols.
//
// Returns an empty array (never nil) if no matches are found.
func (a *App) SearchSymbols(name string, directory string, maxResults int) []SymbolInfo {
	a.logDebug("Searching symbols", logrus.Fields{
		"name":       name,
		"directory":  directory,
		"maxResults": maxResults,
	})

	if directory == "" {
		a.logDebug("Empty directory provided to SearchSymbols", nil)
		return []SymbolInfo{}
	}

	symbols := searchSymbols(name, directory, maxResults)

	a.logDebug("Symbol search complete", logrus.Fields{
		"name":       name,
		"directory":  directory,
		"count":      len(symbols),
		"maxResults": maxResults,
	})

	// Guarantee a non-nil empty slice for the frontend
	if symbols == nil {
		return []SymbolInfo{}
	}
	return symbols
}
