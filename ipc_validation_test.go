package main

import (
	"encoding/json"
	"testing"
)

// TestSearchRequestFuzzyFields verifies that SearchRequest properly serializes
// fuzzySearch and contextLines fields when sent from frontend to backend.
func TestSearchRequestFuzzyFields(t *testing.T) {
	tests := []struct {
		name         string
		request      SearchRequest
		expectFuzzy  bool
		expectCtx    int
	}{
		{
			name: "fuzzy search enabled with custom context",
			request: SearchRequest{
				Directory:    "/test/dir",
				Query:        "test query",
				FuzzySearch:  true,
				ContextLines: 5,
			},
			expectFuzzy: true,
			expectCtx:   5,
		},
		{
			name: "fuzzy search disabled with default context",
			request: SearchRequest{
				Directory:    "/test/dir",
				Query:        "exact match",
				FuzzySearch:  false,
				ContextLines: 3, // Default
			},
			expectFuzzy: false,
			expectCtx:   3,
		},
		{
			name: "zero context lines allowed",
			request: SearchRequest{
				Directory:    "/test/dir",
				Query:        "minimal context",
				FuzzySearch:  true,
				ContextLines: 0,
			},
			expectFuzzy: true,
			expectCtx:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize to JSON (as would happen via Wails binding)
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Deserialize back to verify round-trip
			var deserialized SearchRequest
			if err := json.Unmarshal(data, &deserialized); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			// Verify fields preserved correctly
			if deserialized.FuzzySearch != tt.expectFuzzy {
				t.Errorf("Expected fuzzy=%v, got %v", tt.expectFuzzy, deserialized.FuzzySearch)
			}
			if deserialized.ContextLines != tt.expectCtx {
				t.Errorf("Expected contextLines=%d, got %d", tt.expectCtx, deserialized.ContextLines)
			}
			
			// Verify JSON contains expected keys
			var raw map[string]interface{}
			json.Unmarshal(data, &raw)
			
			if raw["fuzzySearch"] != tt.expectFuzzy {
				t.Errorf("JSON field 'fuzzySearch' expected %v, got %v", tt.expectFuzzy, raw["fuzzySearch"])
			}
			if int(raw["contextLines"].(float64)) != tt.expectCtx {
				t.Errorf("JSON field 'contextLines' expected %d, got %v", tt.expectCtx, raw["contextLines"])
			}
		})
	}
}

// TestWailsBindingFieldPreservation ensures Wails can pass all new fields through bindings.
func TestWailsBindingFieldPreservation(t *testing.T) {
	// Simulate what happens when frontend calls SearchWithProgress with new params
	req := SearchRequest{
		Directory:       "/home/user/project",
		Query:           "import 'fmt'",
		Extension:       "go",
		CaseSensitive:   false,
		IncludeBinary:   false,
		MaxFileSize:     10485760,
		MinFileSize:     0,
		MaxResults:      1000,
		SearchSubdirs:   true,
		UseRegex:        BoolPtr(false),
		ExcludePatterns: []string{"node_modules", ".git"},
		AllowedFileTypes: []string{"go", "ts"},
		FuzzySearch:     true,    // NEW FIELD
		ContextLines:    5,       // NEW FIELD
	}

	// Test full serialization chain
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Should be able to deserialize without data loss
	var result SearchRequest
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all fields including new ones are preserved
	if result.FuzzySearch != req.FuzzySearch {
		t.Errorf("FuzzySearch not preserved: wanted %v, got %v", req.FuzzySearch, result.FuzzySearch)
	}
	if result.ContextLines != req.ContextLines {
		t.Errorf("ContextLines not preserved: wanted %d, got %d", req.ContextLines, result.ContextLines)
	}
	
	// Verify JSON has both new keys
	var raw map[string]interface{}
	json.Unmarshal(jsonBytes, &raw)
	
	if val, ok := raw["fuzzySearch"]; !ok {
		t.Error("JSON missing 'fuzzySearch' key")
	} else if val != req.FuzzySearch {
		t.Errorf("JSON 'fuzzySearch' value mismatch: wanted %v, got %v", req.FuzzySearch, val)
	}
	
	if val, ok := raw["contextLines"]; !ok {
		t.Error("JSON missing 'contextLines' key")
	} else if int(val.(float64)) != req.ContextLines {
		t.Errorf("JSON 'contextLines' value mismatch: wanted %d, got %v", req.ContextLines, val)
	}
}

// Helper function
func BoolPtr(b bool) *bool {
	return &b
}
