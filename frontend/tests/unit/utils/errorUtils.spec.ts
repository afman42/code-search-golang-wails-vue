import { describe, test, expect } from "vitest";
import { toErrorMessage, asRecord } from "../../../src/utils/errorUtils";

describe("toErrorMessage", () => {
  test("returns error.message for Error instances", () => {
    expect(toErrorMessage(new Error("boom"))).toBe("boom");
  });

  test("returns the string itself when error is a string", () => {
    expect(toErrorMessage("plain string error")).toBe("plain string error");
  });

  test("returns .message from a plain object with string message", () => {
    expect(toErrorMessage({ message: "obj error" })).toBe("obj error");
  });

  test("returns fallback for null", () => {
    expect(toErrorMessage(null)).toBe("Unknown error occurred");
  });

  test("returns fallback for undefined", () => {
    expect(toErrorMessage(undefined)).toBe("Unknown error occurred");
  });

  test("returns fallback for number", () => {
    expect(toErrorMessage(42)).toBe("Unknown error occurred");
  });

  test("returns fallback for object without message", () => {
    expect(toErrorMessage({ code: 500 })).toBe("Unknown error occurred");
  });

  test("returns fallback for object with non-string message", () => {
    expect(toErrorMessage({ message: 123 })).toBe("Unknown error occurred");
  });

  test("uses custom fallback when provided", () => {
    expect(toErrorMessage(null, "custom fallback")).toBe("custom fallback");
  });

  test("uses custom fallback for object without message", () => {
    expect(toErrorMessage({ foo: "bar" }, "no msg")).toBe("no msg");
  });
});

describe("asRecord", () => {
  test("returns the object cast for plain objects", () => {
    const result = asRecord({ a: 1, b: "two" });
    expect(result.a).toBe(1);
    expect(result.b).toBe("two");
  });

  test("returns the object cast for arrays", () => {
    const result = asRecord([1, 2, 3]);
    expect(result).toEqual([1, 2, 3]);
  });

  test("returns empty object for null", () => {
    expect(asRecord(null)).toEqual({});
  });

  test("returns empty object for undefined", () => {
    expect(asRecord(undefined)).toEqual({});
  });

  test("returns empty object for string primitive", () => {
    expect(asRecord("hello")).toEqual({});
  });

  test("returns empty object for number primitive", () => {
    expect(asRecord(42)).toEqual({});
  });

  test("returns empty object for boolean primitive", () => {
    expect(asRecord(true)).toEqual({});
  });
});
