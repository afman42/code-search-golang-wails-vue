import { describe, test, expect, vi } from "vitest";
import { highlightMatch, openInEditor } from "../../../src/utils/searchUiUtils";
import type { SearchState } from "../../../src/types/search";
import { makeDefaultEditorAvailability } from "../../../src/composables/useEditorDetection";

// Helper to build a minimal SearchState for highlightMatch calls
function makeState(overrides: Partial<SearchState> = {}): SearchState {
  return {
    directory: "",
    query: "",
    extension: "",
    caseSensitive: false,
    useRegex: false,
    includeBinary: false,
    maxFileSize: 10485760,
    maxResults: 1000,
    searchSubdirs: true,
    resultText: "",
    searchResults: [],
    truncatedResults: false,
    isSearching: false,
    searchProgress: { processedFiles: 0, totalFiles: 0, currentFile: "", resultsCount: 0, status: "" },
    showProgress: false,
    minFileSize: 0,
    excludePatterns: [],
    allowedFileTypes: [],
    recentSearches: [],
    error: null,
    availableEditors: makeDefaultEditorAvailability(),
    editorDetectionStatus: {
      detectionComplete: false, totalAvailable: 0, message: "", detectionProgress: 0,
      detectingEditors: true, detectedEditors: [], availableEditors: {} as any,
    },
    ...overrides,
  };
}

describe("highlightMatch", () => {
  test("should wrap matched text in <mark> tags for literal search", () => {
    const state = makeState({ useRegex: false, caseSensitive: false });
    const result = highlightMatch("hello world hello", "world", state);
    expect(result).toContain('<mark class="highlight">world</mark>');
    expect(result).toContain("hello");
  });

  test("should be case-insensitive by default", () => {
    const state = makeState({ useRegex: false, caseSensitive: false });
    const result = highlightMatch("Hello World", "hello", state);
    expect(result).toContain('<mark class="highlight">Hello</mark>');
  });

  test("should be case-sensitive when configured", () => {
    const state = makeState({ useRegex: false, caseSensitive: true });
    const result = highlightMatch("Hello hello", "hello", state);
    // Only the lowercase "hello" should match
    expect(result).not.toContain('<mark class="highlight">Hello</mark>');
    expect(result).toContain('<mark class="highlight">hello</mark>');
  });

  test("should return empty string for empty text", () => {
    const state = makeState();
    expect(highlightMatch("", "test", state)).toBe("");
  });

  test("should return original text for empty query", () => {
    const state = makeState();
    expect(highlightMatch("some text", "", state)).toBe("some text");
  });

  test("should skip highlighting for very long queries (> 1000 chars)", () => {
    const state = makeState();
    const longQuery = "a".repeat(1001);
    const result = highlightMatch("some text", longQuery, state);
    expect(result).toBe("some text");
  });

  test("ReDoS protection: should return text as-is for > 10KB text in regex mode", () => {
    const state = makeState({ useRegex: true, caseSensitive: false });
    // Create text > 10,000 chars with a pattern that could cause catastrophic backtracking
    const longText = "a".repeat(10001);
    const result = highlightMatch(longText, "(a+)+b", state);
    // Should return the original text unmodified (no regex processing on long text)
    expect(result).toBe(longText);
  });

  test("ReDoS protection: literal mode still highlights text > 10KB correctly", () => {
    // The 10KB cap applies only in regex mode. In literal mode, the escaped query is
    // safe to run via the regex engine on any text length.
    const state = makeState({ useRegex: false, caseSensitive: false });
    const longText = "a".repeat(10001);
    const result = highlightMatch(longText, "aaa", state);
    expect(result).toContain('<mark class="highlight">');
  });

  test("should handle regex mode correctly", () => {
    const state = makeState({ useRegex: true, caseSensitive: false });
    const result = highlightMatch("abc123def456", "\\d+", state);
    expect(result).toContain('<mark class="highlight">123</mark>');
    expect(result).toContain('<mark class="highlight">456</mark>');
  });

  test("should sanitize output to prevent XSS", () => {
    const state = makeState({ useRegex: false, caseSensitive: false });
    // Search for a safe term inside HTML-like text. The non-matched HTML is
    // passed through and DOMPurify strips dangerous tags like <script>.
    const result = highlightMatch(
      'safe text <script>alert(1)</script>',
      'safe',
      state,
    );
    expect(result).toContain('class="highlight"');
    expect(result).not.toContain('<script>alert');
    // The highlight wrapping the matched term is preserved
    expect(result).toContain('<mark class="highlight">safe</mark>');
  });

  test("should handle invalid regex gracefully by falling back to literal match", () => {
    const state = makeState({ useRegex: true, caseSensitive: false });
    // [invalid is not a valid regex
    const result = highlightMatch("test [invalid pattern", "[invalid", state);
    expect(result).toContain('<mark class="highlight">');
  });

  test("should work with special regex characters in literal mode", () => {
    const state = makeState({ useRegex: false, caseSensitive: false });
    const result = highlightMatch("price is $5.00", "$5.00", state);
    expect(result).toContain('<mark class="highlight">$5.00</mark>');
  });

  describe("edge cases", () => {
    test("should return empty string for null-like text", () => {
      const state = makeState();
      expect(highlightMatch(null as any, "test", state)).toBe("");
    });

    test("should return empty string for non-string text (number)", () => {
      const state = makeState();
      expect(highlightMatch(123 as any, "test", state)).toBe("");
    });

    test("should return original text when state is null/undefined", () => {
      const result = highlightMatch("hello world", "world", null as any);
      // Should fall back to defaults and not crash
      expect(result).toContain('<mark class="highlight">world</mark>');
    });

    test("should return text as-is when query is whitespace-only", () => {
      const state = makeState({ useRegex: false });
      const result = highlightMatch("hello world", "   ", state);
      expect(result).toBe("hello world");
    });

    test("should handle overlapping matches correctly", () => {
      const state = makeState({ useRegex: false, caseSensitive: false });
      const result = highlightMatch("aaaa", "aa", state);
      // "aaaa" contains "aa" at positions 0-1 and 2-3
      expect(result).toContain('<mark class="highlight">aa</mark>');
      expect(result).toContain('<mark class="highlight">aa</mark>');
    });

    test("should handle regex with alternation", () => {
      const state = makeState({ useRegex: true, caseSensitive: false });
      const result = highlightMatch("cat dog bird", "cat|dog", state);
      expect(result).toContain('<mark class="highlight">cat</mark>');
      expect(result).toContain('<mark class="highlight">dog</mark>');
    });

    test("should handle regex with word boundaries", () => {
      const state = makeState({ useRegex: true, caseSensitive: false });
      const result = highlightMatch("cat cats cat", "\\bcat\\b", state);
      // Only exact word "cat", not "cats"
      expect(result).toContain('<mark class="highlight">cat</mark>');
      expect(result).not.toContain('cats</mark>');
    });

    test("should handle very long text with no matches efficiently", () => {
      const state = makeState({ useRegex: false, caseSensitive: false });
      const longText = "a".repeat(5000) + "b".repeat(5000);
      const result = highlightMatch(longText, "z", state);
      expect(result).toBe(longText);
    });

    test("should handle regex with lookahead", () => {
      const state = makeState({ useRegex: true, caseSensitive: false });
      // \\d(?=\\d) matches a digit followed by another digit
      const result = highlightMatch("a12b34c", "\\d(?=\\d)", state);
      expect(result).toContain('<mark class="highlight">1</mark>');
      expect(result).toContain('<mark class="highlight">3</mark>');
      // '2' and '4' are not followed by digits, so they should NOT be highlighted
      expect(result).not.toContain('2</mark>');
      expect(result).not.toContain('4</mark>');
    });

    test("should handle case-insensitive regex", () => {
      const state = makeState({ useRegex: true, caseSensitive: false });
      const result = highlightMatch("Hello HELLO hello", "hello", state);
      expect(result).toContain('<mark class="highlight">Hello</mark>');
      expect(result).toContain('<mark class="highlight">HELLO</mark>');
      expect(result).toContain('<mark class="highlight">hello</mark>');
    });

    test("should handle case-sensitive regex mode", () => {
      const state = makeState({ useRegex: true, caseSensitive: true });
      const result = highlightMatch("Hello HELLO hello", "hello", state);
      // Only the exact case match
      expect(result).not.toContain('<mark class="highlight">Hello</mark>');
      expect(result).not.toContain('<mark class="highlight">HELLO</mark>');
      expect(result).toContain('<mark class="highlight">hello</mark>');
    });
  });
});
