import { describe, test, expect, vi, beforeEach } from "vitest";
import type { HLJSApi } from "highlight.js";

// The global setup (tests/setup.ts) calls loadHighlightJs() in a beforeAll,
// warming the module-level `isHighlightingLoaded` cache. Static imports below
// therefore observe the loaded state. To assert the pre-load state we use
// vi.resetModules() + a fresh dynamic import — a legitimate module-loading
// boundary test (the only sanctioned use of await import()).

import {
  loadHighlightJs,
  isHighlightJsLoaded,
  isHighlightingReady,
  detectLanguage,
  highlightCode,
  getHighlightJs,
} from "@/services/syntaxHighlightingService";

describe("syntaxHighlightingService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("isHighlightJsLoaded / isHighlightingReady", () => {
    test("reflect the loaded flag after the global preload", () => {
      // tests/setup.ts already called loadHighlightJs() in beforeAll.
      expect(isHighlightJsLoaded()).toBe(true);
      expect(isHighlightingReady()).toBe(true);
    });

    test("return false before highlight.js has been loaded (fresh module)", async () => {
      vi.resetModules();
      const fresh = await import("@/services/syntaxHighlightingService");

      expect(fresh.isHighlightJsLoaded()).toBe(false);
      expect(fresh.isHighlightingReady()).toBe(false);
    });

    test("isHighlightJsLoaded and isHighlightingReady stay in sync", async () => {
      vi.resetModules();
      const fresh = await import("@/services/syntaxHighlightingService");

      expect(fresh.isHighlightJsLoaded()).toBe(fresh.isHighlightingReady());

      const loaded = await fresh.loadHighlightJs();
      expect(loaded).toBe(true);

      expect(fresh.isHighlightJsLoaded()).toBe(true);
      expect(fresh.isHighlightingReady()).toBe(true);
      expect(fresh.isHighlightJsLoaded()).toBe(fresh.isHighlightingReady());
    });
  });

  describe("loadHighlightJs", () => {
    test("returns true once already loaded (idempotent)", async () => {
      // Already preloaded by setup; a second call short-circuits.
      const result = await loadHighlightJs();
      expect(result).toBe(true);
    });

    test("returns true on fresh load", async () => {
      vi.resetModules();
      const fresh = await import("@/services/syntaxHighlightingService");

      const result = await fresh.loadHighlightJs();
      expect(result).toBe(true);
      expect(fresh.isHighlightJsLoaded()).toBe(true);
    });
  });

  describe("detectLanguage", () => {
    test("returns 'text' for an empty string", () => {
      expect(detectLanguage("")).toBe("text");
    });

    test("returns 'text' for a path with no extension", () => {
      expect(detectLanguage("Makefile")).toBe("makefile");
      expect(detectLanguage("README")).toBe("text");
      expect(detectLanguage("/some/path/noext")).toBe("text");
    });

    test("maps common extensions to highlight.js language ids", () => {
      expect(detectLanguage("main.go")).toBe("go");
      expect(detectLanguage("app.ts")).toBe("typescript");
      expect(detectLanguage("app.tsx")).toBe("typescript");
      expect(detectLanguage("script.py")).toBe("python");
      expect(detectLanguage("script.pyw")).toBe("python");
      expect(detectLanguage("run.sh")).toBe("bash");
      expect(detectLanguage("README.md")).toBe("markdown");
      expect(detectLanguage("README.markdown")).toBe("markdown");
      expect(detectLanguage("app.js")).toBe("javascript");
      expect(detectLanguage("app.mjs")).toBe("javascript");
      expect(detectLanguage("main.rs")).toBe("rust");
      expect(detectLanguage("style.css")).toBe("css");
      expect(detectLanguage("data.json")).toBe("json");
      expect(detectLanguage("config.yaml")).toBe("yaml");
      expect(detectLanguage("config.yml")).toBe("yaml");
      expect(detectLanguage("config.toml")).toBe("ini");
      expect(detectLanguage("Dockerfile")).toBe("dockerfile");
      expect(detectLanguage("Containerfile")).toBe("dockerfile");
      expect(detectLanguage("query.sql")).toBe("sql");
      expect(detectLanguage("schema.graphql")).toBe("graphql");
      expect(detectLanguage("changes.diff")).toBe("diff");
    });

    test("returns 'text' for unknown extensions", () => {
      expect(detectLanguage("file.xyzunknown")).toBe("text");
      expect(detectLanguage("archive.zzz")).toBe("text");
      expect(detectLanguage("weird.qqq")).toBe("text");
    });

    test("is case-insensitive on the extension", () => {
      expect(detectLanguage("MAIN.GO")).toBe("go");
      expect(detectLanguage("App.TS")).toBe("typescript");
      expect(detectLanguage("Script.PY")).toBe("python");
      expect(detectLanguage("RUN.SH")).toBe("bash");
      expect(detectLanguage("Readme.MD")).toBe("markdown");
    });

    test("only considers the final extension after the last dot", () => {
      expect(detectLanguage("/path/to/main.test.go")).toBe("go");
      expect(detectLanguage("src/app.component.ts")).toBe("typescript");
      expect(detectLanguage("foo.bar.json")).toBe("json");
    });

    test("handles dotfiles and dotted names", () => {
      // A bare name like ".go" splits to ["", "go"] -> ext "go".
      expect(detectLanguage(".go")).toBe("go");
      // ".gitignore" -> ext "gitignore" -> plaintext (mapped).
      expect(detectLanguage(".gitignore")).toBe("plaintext");
      // ".env" -> ext "env" -> properties (mapped).
      expect(detectLanguage(".env")).toBe("properties");
      // ".editorconfig" -> ext "editorconfig" -> ini (mapped).
      expect(detectLanguage(".editorconfig")).toBe("ini");
    });
  });

  describe("highlightCode", () => {
    test("returns empty string for empty code", async () => {
      const result = await highlightCode("");
      expect(result).toBe("");
    });

    test("returns empty string for empty code even with options", async () => {
      const result = await highlightCode("", {
        language: "go",
        query: "foo",
        addLineNumbers: false,
      });
      expect(result).toBe("");
    });

    test("wraps each line in a code-line span when loaded", async () => {
      // setup.ts preloaded highlight.js, so isHighlightingLoaded is true.
      const result = await highlightCode("hello\nworld", {
        language: "go",
        addLineNumbers: false,
      });

      expect(result).toContain('<span class="code-line">');
      // Two lines => two code-line spans.
      const matches = result.match(/<span class="code-line">/g);
      expect(matches).not.toBeNull();
      expect(matches!.length).toBe(2);
    });

    test("adds line-number spans when addLineNumbers is true", async () => {
      const result = await highlightCode("line1\nline2", {
        language: "go",
        addLineNumbers: true,
      });

      expect(result).toContain('<span class="line-number"');
      expect(result).toContain('data-line="1"');
      expect(result).toContain('data-line="2"');
    });

    test("does not add line-number spans when addLineNumbers is false", async () => {
      const result = await highlightCode("line1", {
        language: "go",
        addLineNumbers: false,
      });

      expect(result).not.toContain('<span class="line-number"');
    });

    test("highlights query matches with a mark element", async () => {
      const result = await highlightCode("foo bar foo", {
        language: "go",
        query: "foo",
        addLineNumbers: false,
      });

      expect(result).toContain('<mark class="highlight-match">foo</mark>');
    });

    test("escapes query regex special characters", async () => {
      const result = await highlightCode("a(b)c", {
        language: "go",
        query: "a(b)",
        addLineNumbers: false,
      });

      // The literal "a(b)" should be matched, not treated as a regex group.
      expect(result).toContain('<mark class="highlight-match">a(b)</mark>');
    });

    test("falls back to escaped text for an unsupported language", async () => {
      const result = await highlightCode("<script>alert(1)</script>", {
        language: "totally-not-a-language",
        addLineNumbers: false,
      });

      // Unsupported language => escapeHtml path: angle brackets escaped.
      expect(result).toContain("&lt;script&gt;");
      expect(result).not.toContain("<script>alert(1)</script>");
    });

    test("preserves empty lines as a single space inside code-line span", async () => {
      const result = await highlightCode("a\n\nb", {
        language: "go",
        addLineNumbers: false,
      });

      // The empty middle line becomes `${" "}` inside the code-line span.
      expect(result).toContain('<span class="code-line"> </span>');
    });

    test("uses the large-file path for >1000 lines without crashing", async () => {
      const lines: string[] = [];
      for (let i = 0; i < 1005; i++) lines.push(`line ${i}`);
      const code = lines.join("\n");

      const result = await highlightCode(code, {
        language: "go",
        addLineNumbers: true,
      });

      // Large-file path caps at 10000 lines and still emits line numbers.
      expect(result).toContain('<span class="line-number"');
      expect(result).toContain('data-line="1000"');
      // 1005 lines is under the 10000 cap, so no truncation notice.
      expect(result).not.toContain("File truncated");
    });

    test("adds a truncation notice beyond 10000 lines", async () => {
      const lines: string[] = [];
      for (let i = 0; i < 10005; i++) lines.push(`line ${i}`);
      const code = lines.join("\n");

      const result = await highlightCode(code, {
        language: "go",
        addLineNumbers: true,
      });

      expect(result).toContain("File truncated");
      expect(result).toContain("10,000 lines");
    });

    test("defaults to language 'text' and addLineNumbers true", async () => {
      // No options object supplied.
      const result = await highlightCode("hello");

      expect(result).toContain('<span class="line-number"');
      expect(result).toContain('data-line="1"');
    });
  });

  describe("getHighlightJs", () => {
    test("returns the loaded HLJSApi instance after load", () => {
      // setup.ts preloaded highlight.js.
      const instance = getHighlightJs();
      expect(instance).not.toBeNull();
      // The instance exposes the highlight.js API surface.
      expect(typeof (instance as unknown as HLJSApi).highlight).toBe("function");
      expect(typeof (instance as unknown as HLJSApi).getLanguage).toBe("function");
      expect(typeof (instance as unknown as HLJSApi).registerLanguage).toBe(
        "function",
      );
    });

    test("returns null before highlight.js has been loaded (fresh module)", async () => {
      vi.resetModules();
      const fresh = await import("@/services/syntaxHighlightingService");

      expect(fresh.getHighlightJs()).toBeNull();
    });

    test("returns a non-null instance after a fresh load", async () => {
      vi.resetModules();
      const fresh = await import("@/services/syntaxHighlightingService");

      expect(fresh.getHighlightJs()).toBeNull();

      const loaded = await fresh.loadHighlightJs();
      expect(loaded).toBe(true);

      const instance = fresh.getHighlightJs();
      expect(instance).not.toBeNull();
      expect(typeof (instance as unknown as HLJSApi).highlight).toBe("function");
    });
  });
});
