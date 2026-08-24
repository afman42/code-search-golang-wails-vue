import { describe, test, expect } from "vitest";
import { escapeRegExp } from "@/utils";

describe("escapeRegExp", () => {
  test("escapes all 14 regex metacharacters", () => {
    const input = ".*+?^${}()|[\\]";
    expect(escapeRegExp(input)).toBe("\\.\\*\\+\\?\\^\\$\\{\\}\\(\\)\\|\\[\\\\\\]");
  });

  test("returns plain string unchanged when no metacharacters", () => {
    expect(escapeRegExp("hello world 123")).toBe("hello world 123");
  });

  test("escapes backslash explicitly", () => {
    expect(escapeRegExp("\\")).toBe("\\\\");
  });

  test("escapes character class brackets", () => {
    expect(escapeRegExp("[test]")).toBe("\\[test\\]");
  });

  test("empty string returns empty string", () => {
    expect(escapeRegExp("")).toBe("");
  });
});