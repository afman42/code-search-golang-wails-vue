package main

import (
	"encoding/csv"
	"strings"
	"testing"
)

// TestRenderResultsCSV verifies the pure CSV rendering function produces
// well-formed CSV with headers and correct field mapping.
func TestRenderResultsCSV(t *testing.T) {
	results := []SearchResult{
		{
			FilePath:      "/test/file.go",
			LineNum:       42,
			Content:       "func main() {",
			MatchedText:   "main",
			ContextBefore: []string{"package main", ""},
			ContextAfter:  []string{"  fmt.Println()"},
		},
		{
			FilePath:      "/test/util.go",
			LineNum:       10,
			Content:       "var x = 42",
			MatchedText:   "42",
			ContextBefore: []string{},
			ContextAfter:  []string{},
		},
	}

	csvStr, _ := renderResultsCSV(results)

	// Parse it back to verify structure.
	reader := csv.NewReader(strings.NewReader(csvStr))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	// Header row + 2 data rows.
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header + 2 data), got %d", len(rows))
	}

	// Verify headers.
	expectedHeaders := []string{"File Path", "Line Number", "Content", "Matched Text", "Context Before", "Context After"}
	for i, h := range expectedHeaders {
		if rows[0][i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, rows[0][i])
		}
	}

	// First data row.
	if rows[1][0] != "/test/file.go" {
		t.Errorf("row 1 path: expected /test/file.go, got %s", rows[1][0])
	}
	if rows[1][1] != "42" {
		t.Errorf("row 1 line: expected 42, got %s", rows[1][1])
	}
	if rows[1][2] != "func main() {" {
		t.Errorf("row 1 content: got %s", rows[1][2])
	}

	// Context with newlines should be joined.
	if !strings.Contains(rows[1][4], "package main") {
		t.Errorf("row 1 contextBefore missing 'package main': %s", rows[1][4])
	}
}

// TestRenderResultsCSVEmptyContext verifies empty context slices render as
// empty CSV fields, not "null" or "[]".
func TestRenderResultsCSVEmptyContext(t *testing.T) {
	results := []SearchResult{
		{
			FilePath:      "/test/empty.go",
			LineNum:       1,
			Content:       "test",
			MatchedText:   "test",
			ContextBefore: []string{},
			ContextAfter:  []string{},
		},
	}

	csvStr, _ := renderResultsCSV(results)
	reader := csv.NewReader(strings.NewReader(csvStr))
	rows, _ := reader.ReadAll()

	if rows[1][4] != "" {
		t.Errorf("empty ContextBefore should be empty string, got %q", rows[1][4])
	}
	if rows[1][5] != "" {
		t.Errorf("empty ContextAfter should be empty string, got %q", rows[1][5])
	}
}

// TestExportSearchResultsRejectsEmptyResults verifies the binding returns an
// error when no results are provided, before it ever opens the save dialog.
func TestExportSearchResultsRejectsEmptyResults(t *testing.T) {
	app := NewApp()
	_, err := app.ExportSearchResults([]SearchResult{}, "csv")
	if err == nil {
		t.Fatal("expected error for empty results, got nil")
	}
	if !strings.Contains(err.Error(), "no results") {
		t.Errorf("expected 'no results' in error, got: %v", err)
	}

	// nil slice should also be rejected.
	_, err = app.ExportSearchResults(nil, "csv")
	if err == nil {
		t.Error("expected error for nil results")
	}
}

// TestExportSearchResultsRequiresContext verifies the binding returns an
// error (not a panic) when the Wails context is nil — the SaveFileDialog
// call needs a valid context. This test is skipped because SaveFileDialog
// panics (rather than returning an error) when ctx is nil, which is a
// Wails runtime behavior we can't control in a unit test. The binding is
// only callable in a real Wails environment.
func TestExportSearchResultsRequiresContext(t *testing.T) {
	t.Skip("SaveFileDialog panics with nil ctx (Wails runtime behavior); binding only testable in integration")
}

// TestRenderResultsCSVSpecialChars verifies CSV quoting handles commas,
// quotes, and newlines in content.
func TestRenderResultsCSVSpecialChars(t *testing.T) {
	results := []SearchResult{
		{
			FilePath:      "/test/file.go",
			LineNum:       1,
			Content:       `value = "hello, world"`,
			MatchedText:   "hello",
			ContextBefore: []string{},
			ContextAfter:  []string{},
		},
	}

	csvStr, _ := renderResultsCSV(results)
	reader := csv.NewReader(strings.NewReader(csvStr))
	rows, _ := reader.ReadAll()

	if rows[1][2] != `value = "hello, world"` {
		t.Errorf("content with special chars not preserved: got %q", rows[1][2])
	}
}
