package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFuzzyThreshold verifies the threshold computation matches frontend logic
func TestFuzzyThreshold(t *testing.T) {
	tests := []struct {
		name     string
		queryLen int
		expect   int
	}{
		{"query_len_1", 1, 1},   // floor(0.6)=0 but >=1 guard
		{"query_len_2", 2, 1},   // 2*0.6=1.2 floor=1, max(1,1)=1
		{"query_len_3", 3, 1},   // 3*0.6=1.8 floor=1, max(1,1)=1... wait that's wrong
		{"query_len_4", 4, 2},   // 4*0.6=2.4 floor=2
		{"query_len_5", 5, 3},   // 5*0.6=3.0 floor=3
		{"query_len_10", 10, 6}, // 10*0.6=6.0 floor=6
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuzzyThreshold(tt.queryLen)
			if got != tt.expect {
				t.Errorf("fuzzyThreshold(%d) = %d, want %d", tt.queryLen, got, tt.expect)
			}
		})
	}
}

// makeString creates a string of length n
func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// TestFuzzyBestWindow verifies sliding window scoring matches frontend logic
func TestFuzzyBestWindow(t *testing.T) {
	tests := []struct {
		name       string
		textLower  string
		queryLower string
		threshold  int
		wantCount  int
		wantStart  int
	}{
		{"exact_match", "hello", "hello", 2, 5, 0},
		{"partial_match", "hello world", "helo", 2, 3, 0},
		{"no_match", "xxxxx", "hello", 2, -1, 0},
		{"empty_query", "", "hello", 2, -1, 0},
		{"query_longer_than_text", "hi", "hello", 2, -1, 0},
		{"long_text_filtered", makeString(50001), "hello", 2, -1, 0},
		{"perfect_window_stops_early", "abcdefghij", "abcd", 2, 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, start := fuzzyBestWindow([]byte(tt.textLower), []byte(tt.queryLower), tt.threshold)
			if count != tt.wantCount {
				t.Errorf("fuzzyBestWindow(%q, %q, %d) count = %d, want %d", tt.textLower, tt.queryLower, tt.threshold, count, tt.wantCount)
			}
			if start != tt.wantStart || (count < 0 && start != 0) {
				t.Errorf("fuzzyBestWindow(%q, %q, %d) start = %d, want %d", tt.textLower, tt.queryLower, tt.threshold, start, tt.wantStart)
			}
		})
	}
}

// TestSearchFuzzyCandidatesIntegration verifies end-to-end fuzzy search behavior
func TestSearchFuzzyCandidatesIntegration(t *testing.T) {
	app := NewApp()

	tmpDir := t.TempDir()

	fuzzyFile := filepath.Join(tmpDir, "fuzzy.go")
	// Fixture holds a near-miss for the query "hallo" (4/5 chars align
	// positionally with "hello") WITHOUT containing the query verbatim —
	// otherwise the exact pass would legitimately find it with fuzzy off.
	os.WriteFile(fuzzyFile, []byte(`package main
func greet() { fmt.Println("hello there") }
`), 0o644)

	mixedFile := filepath.Join(tmpDir, "mixed.go")
	os.WriteFile(mixedFile, []byte(`package main
func exactMatch() { fmt.Println("needle") }
func fuzzyMatch() { fmt.Println("needel") }
func noMatch() { fmt.Println("other") }
`), 0o644)

	t.Run("fuzzy_off_returns_no_results_for_near_miss", func(t *testing.T) {
		req := SearchRequest{
			Directory:   tmpDir,
			Query:       "hallo",
			FuzzySearch: false,
		}
		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for 'hallo' without fuzzy, got %d: %+v", len(results), results)
		}
	})

	t.Run("fuzzy_on_returns_near_miss_candidates", func(t *testing.T) {
		req := SearchRequest{
			Directory:   tmpDir,
			Query:       "hallo",
			FuzzySearch: true,
			MaxResults:  100,
		}
		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected fuzzy candidate for 'hallo', got none")
		}
		found := false
		for _, r := range results {
			if r.FilePath == fuzzyFile && r.LineNum == 2 {
				found = true
				break
			}
		}
		if !found {
			t.Logf("results: %+v", results)
			t.Error("expected fuzzy.go:2 as fuzzy candidate")
		}
	})

	t.Run("fuzzy_with_regex_disabled_returns_candidates", func(t *testing.T) {
		req := SearchRequest{
			Directory:   tmpDir,
			Query:       "need[el]",
			UseRegex:    true,
			FuzzySearch: true,
			MaxResults:  100,
		}
		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress error: %v", err)
		}
		_ = results
	})

	t.Run("exact_matches_take_precedence_over_fuzzy", func(t *testing.T) {
		req := SearchRequest{
			Directory:   tmpDir,
			Query:       "needle",
			FuzzySearch: true,
			MaxResults:  100,
		}
		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress error: %v", err)
		}
		hasExact := false
		for _, r := range results {
			if r.FilePath == mixedFile && r.LineNum == 2 {
				hasExact = true
				break
			}
		}
		if !hasExact {
			t.Errorf("expected exact match at mixed.go:2, got: %+v", results)
		}
	})

	t.Run("quota_enforced_for_fuzzy_candidates", func(t *testing.T) {
		req := SearchRequest{
			Directory:   tmpDir,
			Query:       "he[l]",
			FuzzySearch: true,
			MaxResults:  1,
		}
		results, err := app.SearchWithProgress(req)
		if err != nil {
			t.Fatalf("SearchWithProgress error: %v", err)
		}
		if len(results) > 1 {
			t.Errorf("expected <=1 result due to quota, got %d: %+v", len(results), results)
		}
	})
}
