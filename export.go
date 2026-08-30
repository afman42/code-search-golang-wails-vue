package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportSearchResults opens a native save-file dialog and writes the given
// search results as CSV or JSON (chosen by the selected file filter).
// Returns the saved file path, or an empty string if the user cancelled.
func (a *App) ExportSearchResults(results []SearchResult, format string) (string, error) {
	if len(results) == 0 {
		return "", fmt.Errorf("no results to export")
	}

	formatLower := strings.ToLower(strings.TrimSpace(format))
	if formatLower == "" {
		formatLower = "csv"
	}

	var defaultName, filterDisplayName, filterPattern string
	switch formatLower {
	case "json":
		defaultName = "search-results.json"
		filterDisplayName = "JSON (*.json)"
		filterPattern = "*.json"
	case "csv":
		fallthrough
	default:
		formatLower = "csv"
		defaultName = "search-results.csv"
		filterDisplayName = "CSV (*.csv)"
		filterPattern = "*.csv"
	}

	a.logInfo("Opening export dialog", logrus.Fields{
		"format":   formatLower,
		"results":  len(results),
		"filename": defaultName,
	})

	dialogOptions := wailsRuntime.SaveDialogOptions{
		Title:           "Export Search Results",
		DefaultFilename: defaultName,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: filterDisplayName, Pattern: filterPattern},
		},
	}

	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, dialogOptions)
	if err != nil {
		a.logError("Export dialog failed", err, nil)
		return "", fmt.Errorf("export dialog failed: %w", err)
	}
	if savePath == "" {
		// User cancelled — not an error.
		return "", nil
	}

	// Render the content.
	var content string
	switch formatLower {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		content = string(data)
	case "csv":
		content, err = renderResultsCSV(results)
		if err != nil {
			return "", fmt.Errorf("failed to render CSV: %w", err)
		}
	}

	if err := os.WriteFile(savePath, []byte(content), 0o644); err != nil {
		a.logError("Export write failed", err, logrus.Fields{"path": savePath})
		return "", fmt.Errorf("failed to write export file: %w", err)
	}

	a.logInfo("Export completed", logrus.Fields{
		"path":    savePath,
		"format":  formatLower,
		"results": len(results),
	})

	return savePath, nil
}

// renderResultsCSV converts search results to CSV with headers:
// File Path, Line Number, Content, Matched Text, Context Before, Context After.
func renderResultsCSV(results []SearchResult) (string, error) {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	if err := writer.Write([]string{"File Path", "Line Number", "Content", "Matched Text", "Context Before", "Context After"}); err != nil {
		return "", err
	}

	for _, r := range results {
		if err := writer.Write([]string{
			csvSafeCell(r.FilePath),
			fmt.Sprintf("%d", r.LineNum),
			csvSafeCell(r.Content),
			csvSafeCell(r.MatchedText),
			csvSafeCell(strings.Join(r.ContextBefore, "\n")),
			csvSafeCell(strings.Join(r.ContextAfter, "\n")),
		}); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// csvSafeCell neutralizes spreadsheet formula injection: cells whose first
// non-space character is =, +, -, @, tab, or CR are interpreted as formulas by
// Excel/LibreOffice and can execute when the CSV is opened. Prefixing a
// single quote forces the cell to be read as text. Leading spaces/tabs are
// trimmed before the check because Excel trims them before evaluating the
// formula, so " =2+2" would otherwise bypass the guard. Matched content from
// arbitrary source files is untrusted, so every text field in the export
// passes through here.
func csvSafeCell(s string) string {
	if s == "" {
		return s
	}
	// Trim only spaces for the check — tabs and CR are themselves formula
	// triggers, so "\t=2+2" and " \t=2+2" must both be caught. Trimming tabs
	// would hide the trigger. We trim spaces, then check the first remaining
	// char against all trigger chars (including tab/CR).
	trimmed := strings.TrimLeft(s, " ")
	if trimmed == "" {
		return s
	}
	switch trimmed[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}
