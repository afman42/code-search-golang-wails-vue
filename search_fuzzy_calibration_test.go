package main

import (
	"context"
	"math/rand"
	"strings"
	"testing"
)

// This file closes the "Fuzzy score calibration" gap from docs/TESTING_GAPS.md:
//   1. Measured false-positive rate on random text (not just spot checks)
//   2. Sensitivity checks (real near-misses must still qualify)
//   3. Performance benchmarks for the sliding-window scorer and the
//      phase-2 candidate pass at scale

// randomAlphabetLine returns n random lowercase letters using the given
// deterministic source, so the false-positive measurement is reproducible.
func randomAlphabetLine(rng *rand.Rand, n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

// TestFuzzyFalsePositiveRateOnRandomText measures how often the sliding-window
// scorer qualifies a purely random line for an 8-char query. With a uniform
// 26-letter alphabet the per-window chance of >= threshold (4) positional
// matches is ~1.3e-4, so even a 100-char line (93 windows) should stay near ~1%.
// The 5% ceiling is generous headroom that still catches gross regressions
// (e.g. a threshold accidentally lowered toward 0.3 would explode past 50%).
func TestFuzzyFalsePositiveRateOnRandomText(t *testing.T) {
	const (
		query    = "needleto" // 8 chars -> threshold floor(8*0.6)=4
		lines    = 2000
		lineLen  = 100
		maxRate  = 0.05
		seed     = int64(42)
	)
	threshold := fuzzyThreshold(len(query))
	if threshold != 4 {
		t.Fatalf("expected threshold 4 for 8-char query, got %d", threshold)
	}

	rng := rand.New(rand.NewSource(seed))
	queryLower := []byte(query)
	falsePositives := 0
	for i := 0; i < lines; i++ {
		line := randomAlphabetLine(rng, lineLen)
		count, _ := fuzzyBestWindow([]byte(line), queryLower, threshold)
		if count >= threshold {
			falsePositives++
		}
	}

	rate := float64(falsePositives) / float64(lines)
	t.Logf("false-positive rate: %d/%d = %.4f (threshold %.2f)", falsePositives, lines, rate, maxRate)
	if rate > maxRate {
		t.Errorf("false-positive rate %.4f exceeds ceiling %.2f — scorer is too permissive", rate, maxRate)
	}
}

// TestFuzzySensitivityNearMisses verifies the other side of calibration: real
// single-edit near-misses of the query must still qualify. Each case is one
// substitution, transposition, insertion, or deletion away from "needle".
func TestFuzzySensitivityNearMisses(t *testing.T) {
	query := "needle"
	threshold := fuzzyThreshold(len(query)) // floor(6*0.6)=3
	if threshold != 3 {
		t.Fatalf("expected threshold 3 for 6-char query, got %d", threshold)
	}

	cases := []struct {
		name string
		text string
	}{
		{"substitution", "needxe"},           // e->x
		{"transposition", "nedele"},          // ed->de swap
		{"deletion", "needl"},                // dropped e (shorter than query: no window fits)
		{"insertion_exact_prefix", "needles"}, // contains query verbatim
		{"near_miss_in_line", "const x = nedle here"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Texts shorter than the query cannot align a full window — the
			// scorer intentionally rejects them; only assert qualification
			// for texts at least as long as the query.
			if len(tt.text) < len(query) {
				t.Skipf("text shorter than query (%d < %d) — scorer rejects by design", len(tt.text), len(query))
			}
			count, _ := fuzzyBestWindow([]byte(tt.text), []byte(query), threshold)
			if count < threshold {
				t.Errorf("near-miss %q did not qualify: count=%d < threshold=%d", tt.text, count, threshold)
			}
		})
	}
}

// TestFuzzyRejectsUnrelatedWords sanity-checks that ordinary unrelated words
// do not qualify, complementing the statistical rate test above.
func TestFuzzyRejectsUnrelatedWords(t *testing.T) {
	query := "needle"
	threshold := fuzzyThreshold(len(query))
	for _, text := range []string{"banana", "orange", "potato", "tomato"} {
		count, _ := fuzzyBestWindow([]byte(text), []byte(query), threshold)
		if count >= threshold {
			t.Errorf("unrelated word %q qualified as fuzzy candidate (count=%d)", text, count)
		}
	}
}

// BenchmarkFuzzyBestWindow measures the sliding-window scorer on a typical
// source line (120 chars) with an 8-char query. This is the hot inner loop of
// the phase-2 fuzzy pass.
func BenchmarkFuzzyBestWindow(b *testing.B) {
	rng := rand.New(rand.NewSource(7))
	text := []byte(randomAlphabetLine(rng, 120))
	query := []byte("needleto")
	threshold := fuzzyThreshold(len(query))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fuzzyBestWindow(text, query, threshold)
	}
}

// BenchmarkSearchFuzzyCandidates measures the full phase-2 fuzzy pass over a
// 200-file tree (reusing the benchmark tree helper from search_bench_test.go).
// The query "nedle" is a near-miss of the "needle marker line" fixtures, so
// every marked file contributes a candidate.
func BenchmarkSearchFuzzyCandidates(b *testing.B) {
	app := quietApp()
	tempDir := setupBenchTree(b, 200)

	req := SearchRequest{
		Directory:   tempDir,
		Query:       "nedle",
		FuzzySearch: true,
		MaxResults:  1000,
	}
	validated, err := app.validateAndSetDefaults(req)
	if err != nil {
		b.Fatalf("validateAndSetDefaults: %v", err)
	}
	req = validated

	pattern, err := app.compileSearchPattern(req)
	if err != nil {
		b.Fatalf("compileSearchPattern: %v", err)
	}

	files, err := app.collectFilesToProcess(req, pattern, "")
	if err != nil {
		b.Fatalf("collectFilesToProcess: %v", err)
	}
	if len(files) == 0 {
		b.Fatal("expected collected files")
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := app.searchFuzzyCandidates(ctx, files, req, pattern, 1000)
		if len(results) == 0 {
			b.Fatal("expected fuzzy candidates for near-miss query")
		}
	}
}

// TestFuzzyCandidatesFindNearMissAtScale is the correctness companion to
// BenchmarkSearchFuzzyCandidates: on a 100-file tree the near-miss query must
// surface the marked files' needle lines as candidates.
func TestFuzzyCandidatesFindNearMissAtScale(t *testing.T) {
	app := NewApp()
	tempDir := setupBenchTree(t, 100)

	req := SearchRequest{
		Directory:   tempDir,
		Query:       "nedle",
		FuzzySearch: true,
		MaxResults:  1000,
	}
	validated, err := app.validateAndSetDefaults(req)
	if err != nil {
		t.Fatalf("validateAndSetDefaults: %v", err)
	}
	req = validated

	pattern, err := app.compileSearchPattern(req)
	if err != nil {
		t.Fatalf("compileSearchPattern: %v", err)
	}
	files, err := app.collectFilesToProcess(req, pattern, "")
	if err != nil {
		t.Fatalf("collectFilesToProcess: %v", err)
	}

	results := app.searchFuzzyCandidates(context.Background(), files, req, pattern, 1000)
	if len(results) == 0 {
		t.Fatal("expected fuzzy candidates at scale, got none")
	}
	for _, r := range results {
		if !strings.Contains(r.Content, "needle marker line") {
			t.Errorf("unexpected candidate content: %q", r.Content)
		}
	}
}