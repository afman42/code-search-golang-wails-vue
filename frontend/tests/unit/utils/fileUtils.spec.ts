import { describe, test, expect } from "vitest";
import { formatFilePath, truncatePath } from "@/utils/fileUtils";

describe("formatFilePath", () => {
  test("returns empty string for empty input", () => {
    expect(formatFilePath("")).toBe("");
  });

  test("returns unchanged when length <= 80", () => {
    const path = "/home/user/project/main.go";
    expect(formatFilePath(path)).toBe(path);
  });

  test("returns unchanged when > 80 chars but <= 5 path parts", () => {
    const long = "/a/b/c/" + "x".repeat(80);
    expect(formatFilePath(long)).toBe(long);
  });

  test("truncates to ...last-3-parts when > 80 chars AND > 5 parts", () => {
    const longPath = "/home/user/project/src/deep/nested/dir/subdir/module/very-long-component-name/main.go";
    expect(longPath.length).toBeGreaterThan(80);
    const result = formatFilePath(longPath);
    expect(result).toMatch(/^\.\.\./);
    const parts = result.slice(3).split("/");
    expect(parts.length).toBe(3);
    expect(result).toBe("..." + longPath.split("/").slice(-3).join("/"));
  });
});

describe("truncatePath", () => {
  test("returns empty string for empty input", () => {
    expect(truncatePath("")).toBe("");
  });

  test("returns unchanged when <= maxLength", () => {
    expect(truncatePath("short", 50)).toBe("short");
  });

  test("returns unchanged when exactly maxLength", () => {
    const s = "x".repeat(50);
    expect(truncatePath(s, 50)).toBe(s);
  });

  test("truncates with ... prefix when longer than maxLength", () => {
    const s = "x".repeat(100);
    const result = truncatePath(s, 50);
    expect(result).toMatch(/^\.\.\./);
    expect(result.length).toBe(50);
  });

  test("uses default maxLength of 50", () => {
    const s = "y".repeat(80);
    const result = truncatePath(s);
    expect(result.length).toBe(50);
    expect(result).toMatch(/^\.\.\./);
  });
});
