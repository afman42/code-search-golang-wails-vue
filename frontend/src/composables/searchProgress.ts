import type { SearchProgress } from "@/types";

// Coerce an untyped Wails "search-progress" event payload into a SearchProgress.
// The payload crosses the JS bridge as `unknown`; read each field defensively
// so a missing/renamed field degrades to a default rather than throwing.
export function coerceProgress(payload: unknown): SearchProgress {
  const p = (payload && typeof payload === "object" ? payload : {}) as Record<string, unknown>;
  const num = (v: unknown): number => (typeof v === "number" ? v : 0);
  const str = (v: unknown): string => (typeof v === "string" ? v : "");
  return {
    processedFiles: num(p.processedFiles),
    totalFiles: num(p.totalFiles),
    currentFile: str(p.currentFile),
    resultsCount: num(p.resultsCount),
    status: str(p.status),
  };
}
