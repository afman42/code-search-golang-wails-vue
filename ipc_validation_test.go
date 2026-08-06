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
		UseRegex:         false,
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

// TestSearchRequestUseRegexSerialization defends the backend→frontend IPC
// contract for the useRegex field. UseRegex is a plain bool (not *bool): a
// nil pointer would marshal to "useRegex":null, which is not assignable to the
// frontend's `useRegex: boolean` type (and the Wails-generated models.ts
// declares it as a required boolean). This test pins that the zero value
// serializes to false and that all three modes round-trip losslessly.
func TestSearchRequestUseRegexSerialization(t *testing.T) {
	tests := []struct {
		name    string
		useRegex bool
	}{
		{name: "zero value is false (not null)", useRegex: false},
		{name: "explicit true", useRegex: true},
		{name: "explicit false", useRegex: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SearchRequest{
				Directory: "/test/dir",
				Query:     "test",
				UseRegex:  tt.useRegex,
			}
			data, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			// The JSON value must be a JSON boolean, never null. null would
			// violate the frontend's `useRegex: boolean` type contract.
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal to raw map failed: %v", err)
			}
			val, ok := raw["useRegex"]
			if !ok {
				t.Fatal("JSON missing 'useRegex' key")
			}
			if string(val) == "null" {
				t.Errorf("useRegex serialized as null — not assignable to frontend boolean type (was *bool previously)")
			}
			var got bool
			if err := json.Unmarshal(val, &got); err != nil {
				t.Fatalf("useRegex JSON value %q is not a boolean: %v", string(val), err)
			}
			if got != tt.useRegex {
				t.Errorf("useRegex round-trip mismatch: wanted %v, got %v", tt.useRegex, got)
			}

			// Full struct round-trip must preserve the value.
			var deserialized SearchRequest
			if err := json.Unmarshal(data, &deserialized); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if deserialized.UseRegex != tt.useRegex {
				t.Errorf("UseRegex not preserved: wanted %v, got %v", tt.useRegex, deserialized.UseRegex)
			}
		})
	}
}

// TestSearchRequestUseRegexOmittedField verifies that a JSON payload from an
// older caller that omits useRegex entirely deserializes to false (literal
// search) rather than panicking on a nil pointer dereference. This is the
// programmatic-caller compatibility path.
func TestSearchRequestUseRegexOmittedField(t *testing.T) {
	// JSON payload with no useRegex key at all — simulates an older caller.
	payload := []byte(`{"directory":"/test/dir","query":"literal [abc]"}`)

	var req SearchRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("unmarshal of payload missing useRegex failed: %v", err)
	}
	if req.UseRegex != false {
		t.Errorf("omitted useRegex must default to false (literal), got %v", req.UseRegex)
	}
}
