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
		content = renderResultsCSV(results)
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
func renderResultsCSV(results []SearchResult) string {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	writer.Write([]string{"File Path", "Line Number", "Content", "Matched Text", "Context Before", "Context After"})

	for _, r := range results {
		writer.Write([]string{
			r.FilePath,
			fmt.Sprintf("%d", r.LineNum),
			r.Content,
			r.MatchedText,
			strings.Join(r.ContextBefore, "\n"),
			strings.Join(r.ContextAfter, "\n"),
		})
	}

	writer.Flush()
	return sb.String()
}
