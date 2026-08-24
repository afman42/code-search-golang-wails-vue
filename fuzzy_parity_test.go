package main

import (
	"os"
	"strings"
	"testing"
)

// TestFuzzyParityConstants is a tripwire: the Go fuzzy scanner
// (searchFuzzyCandidates in search_fuzzy.go) and the frontend scorer
// (frontend/src/utils/fuzzyMatch.ts) must agree on their tuning constants,
// or Go-side candidate selection and TS-side scoring silently diverge and
// users see different results than were searched for.
func TestFuzzyParityConstants(t *testing.T) {
	const path = "frontend/src/utils/fuzzyMatch.ts"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s (run from repo root): %v", path, err)
	}
	src := string(data)

	for _, want := range []struct{ decl, value string }{
		{"SLIDING_WINDOW_SIMILARITY_THRESHOLD", "0.6"},
		{"MAX_TEXT_LENGTH_FOR_FUZZY_SEARCH", "50000"},
	} {
		if !strings.Contains(src, "const "+want.decl) {
			t.Errorf("fuzzyMatch.ts no longer declares const %s — update this tripwire AND search_fuzzy.go together", want.decl)
		}
		if !strings.Contains(src, "= "+want.value+";") {
			t.Errorf("%s is no longer %s in fuzzyMatch.ts — parity with search_fuzzy.go broken; update both sides together", want.decl, want.value)
		}
	}
}
