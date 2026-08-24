import { describe, test, expect } from "vitest";
import { escapeHtml } from "@/utils/htmlUtils";

describe("escapeHtml", () => {
  test("escapes & < > \" ' characters", () => {
    expect(escapeHtml("&<>\"'")).toBe("&amp;&lt;&gt;&quot;&#039;");
  });

  test("returns plain string unchanged when no special chars", () => {
    expect(escapeHtml("hello world")).toBe("hello world");
  });

  test("returns empty string for empty input", () => {
    expect(escapeHtml("")).toBe("");
  });

  test("returns empty string for falsy input", () => {
    expect(escapeHtml(null as unknown as string)).toBe("");
    expect(escapeHtml(undefined as unknown as string)).toBe("");
  });

  test("escapes ampersand first to avoid double-encoding", () => {
    expect(escapeHtml("&lt;")).toBe("&amp;lt;");
  });
});