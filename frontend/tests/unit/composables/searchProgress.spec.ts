import { describe, test, expect } from "vitest";
import { coerceProgress, coerceResultBatch } from "@/composables/searchProgress";

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

describe("coerceResultBatch", () => {
  test("valid batch with single result", () => {
    const b = coerceResultBatch({
      seq: 1,
      results: [{ filePath: "/a/b.go", lineNum: 10, content: "foo", matchedText: "foo", contextBefore: [], contextAfter: [] }],
    });
    expect(b).not.toBeNull();
    expect(b!.seq).toBe(1);
    expect(b!.results.length).toBe(1);
    expect(b!.results[0].filePath).toBe("/a/b.go");
  });

  test("filters rows missing filePath or lineNum", () => {
    const b = coerceResultBatch({
      seq: 2,
      results: [
        { filePath: "/a.go", lineNum: 1, content: "x" },
        { filePath: null, lineNum: 2, content: "x" },
        { filePath: "/b.go", lineNum: "not-number", content: "x" },
        null,
        "string",
        { filePath: "/c.go", lineNum: 3, content: "x", matchedText: "m", contextBefore: ["a"], contextAfter: ["b"] },
      ],
    });
    expect(b).not.toBeNull();
    expect(b!.results.length).toBe(2);
    expect(b!.results[0].filePath).toBe("/a.go");
    expect(b!.results[1].filePath).toBe("/c.go");
  });

  test("contextBefore/After filtered to strings only", () => {
    const b = coerceResultBatch({
      seq: 3,
      results: [{ filePath: "/x.go", lineNum: 1, content: "c", matchedText: "c", contextBefore: ["ok", 123, null], contextAfter: ["a", false] }],
    });
    expect(b!.results[0].contextBefore).toEqual(["ok"]);
    expect(b!.results[0].contextAfter).toEqual(["a"]);
  });

  test("returns null when seq missing or not number", () => {
    expect(coerceResultBatch({ results: [] })).toBeNull();
    expect(coerceResultBatch({ seq: "1", results: [] })).toBeNull();
    expect(coerceResultBatch({ seq: 1 })).toBeNull();
  });

  test("returns null when results not array", () => {
    expect(coerceResultBatch({ seq: 1, results: "not-array" })).toBeNull();
    expect(coerceResultBatch({ seq: 1, results: null })).toBeNull();
  });

  test("returns null when no valid results after filtering", () => {
    const b = coerceResultBatch({
      seq: 4,
      results: [{ filePath: null, lineNum: 1 }],
    });
    expect(b).toBeNull();
  });

  test("returns null for empty results array", () => {
    expect(coerceResultBatch({ seq: 5, results: [] })).toBeNull();
  });

  test("null/undefined/non-object payload returns null", () => {
    expect(coerceResultBatch(null)).toBeNull();
    expect(coerceResultBatch(undefined)).toBeNull();
    expect(coerceResultBatch("string")).toBeNull();
    expect(coerceResultBatch(123)).toBeNull();
  });

  test("defaults content/matchedText to empty string when missing", () => {
    const b = coerceResultBatch({
      seq: 6,
      results: [{ filePath: "/a.go", lineNum: 1 }],
    });
    expect(b!.results[0].content).toBe("");
    expect(b!.results[0].matchedText).toBe("");
  });
});