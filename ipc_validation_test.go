package main

import (
	"encoding/json"
	"testing"
)

// TestSearchRequestContextLines verifies that the contextLines field properly
// serializes when sent from frontend to backend. Fuzzy filtering is a purely
// client-side concern, so it is intentionally absent from the Go request model.
func TestSearchRequestContextLines(t *testing.T) {
	tests := []struct {
		name      string
		request   SearchRequest
		expectCtx int
	}{
		{
			name: "custom context lines preserved",
			request: SearchRequest{
				Directory:    "/test/dir",
				Query:        "test query",
				ContextLines: 5,
			},
			expectCtx: 5,
		},
		{
			name: "default context",
			request: SearchRequest{
				Directory:    "/test/dir",
				Query:        "exact match",
				ContextLines: 3, // Default
			},
			expectCtx: 3,
		},
		{
			name: "zero context lines allowed (means unset -> engine default)",
			request: SearchRequest{
				Directory:    "/test/dir",
				Query:        "minimal context",
				ContextLines: 0,
			},
			expectCtx: 0,
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

			// Verify field preserved correctly
			if deserialized.ContextLines != tt.expectCtx {
				t.Errorf("Expected contextLines=%d, got %d", tt.expectCtx, deserialized.ContextLines)
			}

			// Verify JSON contains expected key
			var raw map[string]interface{}
			json.Unmarshal(data, &raw)

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
		Directory:        "/home/user/project",
		Query:            "import 'fmt'",
		Extension:        "go",
		CaseSensitive:    false,
		IncludeBinary:    false,
		MaxFileSize:      10485760,
		MinFileSize:      0,
		MaxResults:       1000,
		SearchSubdirs:    true,
		UseRegex:         BoolPtr(false),
		ExcludePatterns:  []string{"node_modules", ".git"},
		AllowedFileTypes: []string{"go", "ts"},
		ContextLines:     5, // NEW FIELD
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
	if result.ContextLines != req.ContextLines {
		t.Errorf("ContextLines not preserved: wanted %d, got %d", req.ContextLines, result.ContextLines)
	}

	// Verify JSON has the new key
	var raw map[string]interface{}
	json.Unmarshal(jsonBytes, &raw)

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
