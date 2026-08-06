import { describe, test, expect, vi, beforeEach } from "vitest";
import { ref, nextTick } from "vue";
import { useCodeHighlighting } from '@/composables';

// Mock the syntaxHighlightingService module so the composable's
// detectLanguage / highlightCode dependencies are deterministic and the
// global setup.ts preload (which imports loadHighlightJs from this module)
// does not pull in the real highlight.js bundle.
vi.mock("@/services", () => ({
  initializeAppServices: vi.fn(),
  detectLanguage: vi.fn((filePath: string) => {
    // Minimal stand-in mirroring the real extension-based mapping so the
    // delegation assertion has a meaningful return value to check.
    if (!filePath) return "text";
    return "go";
  }),
  highlightCode: vi.fn().mockResolvedValue("<span class='hljs'>highlighted</span>"),
  loadHighlightJs: vi.fn().mockResolvedValue(true),
  isHighlightJsLoaded: vi.fn().mockReturnValue(true),
  isHighlightingReady: vi.fn().mockReturnValue(true),
  getHighlightJs: vi.fn().mockReturnValue(null),
}));

import { detectLanguage, highlightCode } from '@/services';

describe("useCodeHighlighting", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const makeComposable = (overrides?: {
    content?: string;
    path?: string;
    query?: string;
    addLineNumbers?: boolean;
  }) => {
    const contentRef = ref(overrides?.content ?? "");
    const pathRef = ref(overrides?.path ?? "");
    const queryRef = ref(overrides?.query ?? "");
    const addLineNumbersRef = ref(overrides?.addLineNumbers ?? true);
    const composable = useCodeHighlighting(
      () => contentRef.value,
      () => pathRef.value,
      () => queryRef.value,
      addLineNumbersRef,
    );
    return { composable, contentRef, pathRef, queryRef, addLineNumbersRef };
  };

  test("renderPlainText returns empty string for empty content", () => {
    const { composable } = makeComposable({ content: "" });
    expect(composable.renderPlainText()).toBe("");
  });

  test("renderPlainText returns empty string for whitespace-only content is non-empty but no content", () => {
    // Empty string is the only falsy content path; a single space is truthy
    // and renders one line. Verify the falsy guard specifically.
    const { composable } = makeComposable({ content: "" });
    expect(composable.renderPlainText()).toBe("");
  });

  test("renderPlainText escapes HTML special characters in line content", () => {
    const { composable } = makeComposable({
      content: '<script>alert("x") & \'y\'</script>',
      addLineNumbers: false,
    });
    const result = composable.renderPlainText();
    expect(result).not.toContain("<script>");
    expect(result).toContain("&lt;script&gt;");
    expect(result).toContain("&quot;x&quot;");
    expect(result).toContain("&#039;y&#039;");
    expect(result).toContain("&amp;");
  });

  test("renderPlainText wraps query matches in <mark class='highlight-match'> case-insensitively", () => {
    const { composable } = makeComposable({
      content: "Hello World hello",
      query: "hello",
      addLineNumbers: false,
    });
    const result = composable.renderPlainText();
    const marks = result.match(/<mark class="highlight-match">/g);
    expect(marks).not.toBeNull();
    expect(marks!.length).toBe(2);
    expect(result).toContain('<mark class="highlight-match">Hello</mark>');
    expect(result).toContain('<mark class="highlight-match">hello</mark>');
  });

  test("renderPlainText escapes regex metacharacters in the query (literal match, no throw)", () => {
    // A query full of regex metacharacters must be treated literally and
    // must not throw when constructing the RegExp.
    const { composable } = makeComposable({
      content: "price is ($50) + tax",
      query: "($50) + tax",
      addLineNumbers: false,
    });
    expect(() => composable.renderPlainText()).not.toThrow();
    const result = composable.renderPlainText();
    expect(result).toContain('<mark class="highlight-match">($50) + tax</mark>');
  });

  test("renderPlainText applies no highlight when a regex-like query does not match content (no throw)", () => {
    // Query that would be an invalid regex if unescaped; escaping makes it a
    // valid literal pattern that simply does not match, so no <mark> appears.
    const { composable } = makeComposable({
      content: "hello world",
      query: "(*+?^$",
      addLineNumbers: false,
    });
    expect(() => composable.renderPlainText()).not.toThrow();
    const result = composable.renderPlainText();
    expect(result).not.toContain("<mark");
  });

  test("renderPlainText adds line-number spans when addLineNumbers is true", () => {
    const { composable } = makeComposable({
      content: "line one\nline two\nline three",
      addLineNumbers: true,
    });
    const result = composable.renderPlainText();
    const lineNumbers = result.match(/<span class="line-number"/g);
    expect(lineNumbers).not.toBeNull();
    expect(lineNumbers!.length).toBe(3);
    expect(result).toContain('data-line="1"');
    expect(result).toContain('data-line="2"');
    expect(result).toContain('data-line="3"');
  });

  test("renderPlainText omits line-number spans when addLineNumbers is false", () => {
    const { composable } = makeComposable({
      content: "line one\nline two",
      addLineNumbers: false,
    });
    const result = composable.renderPlainText();
    expect(result).not.toContain('class="line-number"');
    // code-line spans are still emitted per line
    const codeLines = result.match(/<span class="code-line">/g);
    expect(codeLines).not.toBeNull();
    expect(codeLines!.length).toBe(2);
  });

  test("renderPlainText truncates at MAX_LINES_FOR_RENDERING (10000) with a truncation notice", () => {
    const totalLines = 10001;
    const content = Array.from({ length: totalLines }, (_, i) => `line ${i + 1}`).join("\n");
    const { composable } = makeComposable({ content, addLineNumbers: true });
    const result = composable.renderPlainText();

    // The first 10000 lines are rendered as normal code-line spans.
    const codeLineSpans = result.match(/class="code-line"/g);
    expect(codeLineSpans).not.toBeNull();
    expect(codeLineSpans!.length).toBe(10000);

    // One extra span carrying the truncation comment (note: its class is
    // "code-line comment", so the bare-class regex above does not match it).
    expect(result).toContain('class="code-line comment"');

    // The exact truncation notice appended after the loop.
    expect(result).toContain(
      '<span class="line-number" data-line="...">...</span><span class="code-line comment">/* File truncated - showing first 10,000 lines */</span>',
    );
  });

  test("renderPlainText does not truncate files at exactly MAX_LINES_FOR_RENDERING", () => {
    const content = Array.from({ length: 10000 }, (_, i) => `line ${i + 1}`).join("\n");
    const { composable } = makeComposable({ content, addLineNumbers: true });
    const result = composable.renderPlainText();
    expect(result).not.toContain("File truncated");
  });

  test("detectedLanguage delegates to detectLanguage(filePath)", () => {
    vi.mocked(detectLanguage).mockReturnValue("go");
    const { composable } = makeComposable({ path: "/some/path/main.go" });
    expect(composable.detectedLanguage.value).toBe("go");
    expect(detectLanguage).toHaveBeenCalledWith("/some/path/main.go");
  });

  test("detectedLanguage is reactive to filePath changes", async () => {
    vi.mocked(detectLanguage).mockImplementation((p: string) =>
      p.endsWith(".rs") ? "rust" : "text",
    );
    const { composable, pathRef } = makeComposable({ path: "main.go" });
    expect(composable.detectedLanguage.value).toBe("text");
    pathRef.value = "main.rs";
    await nextTick();
    expect(composable.detectedLanguage.value).toBe("rust");
  });

  test("loadAndHighlight sets isReady to true", async () => {
    const { composable } = makeComposable({ content: "some content" });
    expect(composable.isReady.value).toBe(false);
    await composable.loadAndHighlight();
    expect(composable.isReady.value).toBe(true);
  });

  test("loadAndHighlight sets isReady true and clears output for empty content", async () => {
    const { composable } = makeComposable({ content: "" });
    await composable.loadAndHighlight();
    expect(composable.isReady.value).toBe(true);
    expect(composable.highlightedCodeRef.value).toBe("");
    expect(highlightCode).not.toHaveBeenCalled();
  });

  test("loadAndHighlight populates highlightedCodeRef with the highlighted result when present", async () => {
    vi.mocked(highlightCode).mockResolvedValue("<span class='hljs'>highlighted</span>");
    const { composable } = makeComposable({ content: "package main" });
    await composable.loadAndHighlight();
    expect(composable.highlightedCodeRef.value).toBe("<span class='hljs'>highlighted</span>");
    expect(highlightCode).toHaveBeenCalledTimes(1);
    expect(highlightCode).toHaveBeenCalledWith(
      "package main",
      expect.objectContaining({ language: expect.any(String), query: "", addLineNumbers: true }),
    );
  });

  test("loadAndHighlight keeps plain-text rendering when highlightCode resolves falsy", async () => {
    vi.mocked(highlightCode).mockResolvedValue("");
    const { composable } = makeComposable({ content: "plain text", addLineNumbers: false });
    await composable.loadAndHighlight();
    // Falsy result means the plain-text rendering is retained.
    expect(composable.highlightedCodeRef.value).toContain('<span class="code-line">');
    expect(composable.highlightedCodeRef.value).not.toContain("<span class='hljs'>");
  });

  test("loadAndHighlight keeps plain-text rendering when highlightCode rejects", async () => {
    vi.mocked(highlightCode).mockRejectedValue(new Error("boom"));
    // Silence the expected console.error from the catch path.
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { composable } = makeComposable({ content: "plain text", addLineNumbers: false });
    await composable.loadAndHighlight();
    expect(composable.isReady.value).toBe(true);
    expect(composable.highlightedCodeRef.value).toContain('<span class="code-line">');
    spy.mockRestore();
  });

  test("watch re-runs loadAndHighlight and keeps isReady true on content change", async () => {
    vi.mocked(highlightCode).mockResolvedValue("<span class='hljs'>highlighted</span>");
    const { composable, contentRef } = makeComposable({ content: "initial" });
    await composable.loadAndHighlight();
    expect(composable.isReady.value).toBe(true);

    contentRef.value = "changed";
    // The watch no longer blanks isReady when content already exists —
    // the old highlighted HTML (with data-line spans) stays in the DOM
    // until the new pass replaces it, so pending waitForHighlightReady
    // polls don't miss the target element.
    await nextTick();
    await composable.loadAndHighlight();
    expect(composable.isReady.value).toBe(true);
  });

  test("watch does not blank isReady when transitioning between non-empty content", async () => {
    // Make highlightCode resolve slowly so we can observe isReady during
    // the re-highlight. With the fix, isReady stays true throughout — the
    // old highlighted HTML (with data-line spans) remains in the DOM until
    // the new pass replaces it, so pending waitForHighlightReady polls
    // don't miss the target element.
    vi.mocked(highlightCode).mockResolvedValue("<span class='hljs'>highlighted</span>");
    const { composable, contentRef } = makeComposable({ content: "initial" });
    await composable.loadAndHighlight();
    expect(composable.isReady.value).toBe(true);

    // Swap to a slow highlightCode for the re-highlight pass.
    const { promise: slowHighlight, resolve: resolveHighlight } =
      Promise.withResolvers<string>();
    vi.mocked(highlightCode).mockReturnValueOnce(slowHighlight);

    contentRef.value = "changed";
    await nextTick();

    // While highlightCode is still pending, isReady must still be true.
    expect(composable.isReady.value).toBe(true);
    expect(composable.highlightedCodeRef.value).toContain('data-line');

    resolveHighlight("<span class='hljs'>highlighted</span>");
    await composable.loadAndHighlight();
    expect(composable.isReady.value).toBe(true);
  });
});
