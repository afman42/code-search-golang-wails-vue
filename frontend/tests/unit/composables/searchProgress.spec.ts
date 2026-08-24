import { describe, test, expect } from "vitest";
import { coerceProgress } from "@/composables/searchProgress";

describe("coerceProgress", () => {
  test("valid full payload", () => {
    const p = coerceProgress({
      processedFiles: 10,
      totalFiles: 100,
      currentFile: "src/main.go",
      resultsCount: 5,
      failedFiles: 1,
      status: "in-progress",
    });
    expect(p.processedFiles).toBe(10);
    expect(p.totalFiles).toBe(100);
    expect(p.currentFile).toBe("src/main.go");
    expect(p.resultsCount).toBe(5);
    expect(p.failedFiles).toBe(1);
    expect(p.status).toBe("in-progress");
  });

  test("status guard rejects invalid status", () => {
    const p = coerceProgress({
      processedFiles: 1,
      totalFiles: 10,
      currentFile: "x.txt",
      resultsCount: 0,
      failedFiles: 0,
      status: "invalid",
    });
    // Falls back to "started" for unknown status.
    expect(p.status).toBe("started");
  });

  test("missing fields default to safe values", () => {
    const p = coerceProgress({});
    expect(p.processedFiles).toBe(0);
    expect(p.totalFiles).toBe(0);
    expect(p.currentFile).toBe("");
    expect(p.resultsCount).toBe(0);
    expect(p.failedFiles).toBe(0);
    expect(p.status).toBe("started");
  });

  test("null/undefined payload", () => {
    const p1 = coerceProgress(null);
    expect(p1.processedFiles).toBe(0);
    expect(p1.status).toBe("started");

    const p2 = coerceProgress(undefined);
    expect(p2.processedFiles).toBe(0);
    expect(p2.status).toBe("started");
  });

  test("non-object payload", () => {
    const p = coerceProgress("string");
    expect(p.processedFiles).toBe(0);
    expect(p.status).toBe("started");
  });

  test("accepts both processedFiles and processed field names", () => {
    const p = coerceProgress({
      processed: 42,
      total: 200,
      status: "completed",
    });
    expect(p.processedFiles).toBe(42);
    expect(p.totalFiles).toBe(200);
    expect(p.status).toBe("completed");
  });

  test("processed fallback when processedFiles is 0", () => {
    const p = coerceProgress({
      processedFiles: 0,
      processed: 99,
      totalFiles: 100,
      status: "in-progress",
    });
    expect(p.processedFiles).toBe(99);
  });

  test("completes with null-ish values treated as defaults", () => {
    const p = coerceProgress({
      processedFiles: null,
      totalFiles: undefined,
      status: "completed",
    });
    expect(p.processedFiles).toBe(0);
    expect(p.totalFiles).toBe(0);
    expect(p.status).toBe("completed");
  });

  test("started→completed→cancelled transitions", () => {
    const s1 = coerceProgress({ status: "started" });
    expect(s1.status).toBe("started");

    const s2 = coerceProgress({ status: "completed" });
    expect(s2.status).toBe("completed");

    const s3 = coerceProgress({ status: "cancelled" });
    expect(s3.status).toBe("cancelled");
  });
});