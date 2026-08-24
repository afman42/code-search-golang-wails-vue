import { isSearchStatus } from "@/types";
import type { SearchProgress } from "@/types";

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
  };
}
