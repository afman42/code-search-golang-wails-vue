import { isSearchStatus } from "@/types";
import type { SearchProgress, SearchResult, SearchResultBatch } from "@/types";

// Coerce an untyped Wails "search-progress" event payload into a SearchProgress.
// The payload crosses the JS bridge as `unknown`; read each field defensively
// so a missing/renamed field degrades to a default rather than throwing.
// The symbol-scan event ("symbol-progress") uses `processed`/`total` instead
// of `processedFiles`/`totalFiles`, so both spellings are accepted.
export function coerceProgress(payload: unknown): SearchProgress {
  const p = (payload && typeof payload === "object" ? payload : {}) as Record<string, unknown>;
  const num = (v: unknown): number => (typeof v === "number" ? v : 0);
  const str = (v: unknown): string => (typeof v === "string" ? v : "");
  return {
    processedFiles: num(p.processedFiles) || num(p.processed),
    totalFiles: num(p.totalFiles) || num(p.total),
    currentFile: str(p.currentFile),
    resultsCount: num(p.resultsCount),
    failedFiles: num(p.failedFiles),
    status: isSearchStatus(p.status) ? p.status : "started",
    // Go marshals a nil slice as null, so the absent case is null (not []).
    failedPaths: Array.isArray(p.failedPaths)
      ? p.failedPaths.filter((v): v is string => typeof v === "string")
      : [],
  };
}

// Coerce an untyped Wails "search-results" event payload into a
// SearchResultBatch. Returns null when the payload carries no usable results,
// so the caller can ignore it instead of appending an empty batch and bumping
// its sequence tracking.
//
// Only the fields the results list actually renders are required; a row
// missing filePath or lineNum is dropped rather than rendered as a blank hit.
export function coerceResultBatch(payload: unknown): SearchResultBatch | null {
  const p = (payload && typeof payload === "object" ? payload : {}) as Record<string, unknown>;
  if (typeof p.seq !== "number" || !Array.isArray(p.results)) return null;

  const strArray = (v: unknown): string[] =>
    Array.isArray(v) ? v.filter((s): s is string => typeof s === "string") : [];

  const results = p.results.reduce<SearchResult[]>((acc, raw) => {
    if (!raw || typeof raw !== "object") return acc;
    const r = raw as Record<string, unknown>;
    if (typeof r.filePath !== "string" || typeof r.lineNum !== "number") return acc;
    acc.push({
      filePath: r.filePath,
      lineNum: r.lineNum,
      content: typeof r.content === "string" ? r.content : "",
      matchedText: typeof r.matchedText === "string" ? r.matchedText : "",
      contextBefore: strArray(r.contextBefore),
      contextAfter: strArray(r.contextAfter),
    });
    return acc;
  }, []);

  if (results.length === 0) return null;
  return { seq: p.seq, results };
}
