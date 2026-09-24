// Shared Wails-call wrapper + bridge payload coercion helpers.
//
// Extracted from useSearch.ts, useReplace.ts, useSymbolSearch.ts and
// toastUtils.ts, which each hand-rolled the same shape: try a Wails binding,
// narrow `unknown` with toErrorMessage, surface toastManager.error with a
// title, optionally twin-write a status message.
import { toastManager } from "@/composables/useToast";
import { toErrorMessage } from "./errorUtils";

/** Read a numeric field defensively (bridge may omit/rename it). */
export function numField(record: Record<string, unknown>, key: string): number {
  const v = record[key];
  return typeof v === "number" ? v : 0;
}

/** Read a string field defensively. */
export function strField(record: Record<string, unknown>, key: string): string {
  const v = record[key];
  return typeof v === "string" ? v : "";
}

/** Narrow an unknown Wails event payload to a record (defaults to {}). */
export function payloadRecord(payload: unknown): Record<string, unknown> {
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return payload as Record<string, unknown>;
  }
  return {};
}

export interface WailsCallOpts {
  title: string;
  fallback?: string;
  setStatus?: (message: string, type: string) => void;
}

/**
 * Run a Wails binding with standardized error UX: toast + optional status
 * twin-write. Returns null on failure so callers can early-return instead of
 * nesting. Success passes the value through unchanged.
 */
export async function withWailsCall<T>(
  run: () => Promise<T>,
  opts: WailsCallOpts,
): Promise<T | null> {
  try {
    return await run();
  } catch (error: unknown) {
    const msg = toErrorMessage(error, opts.fallback ?? "Operation failed");
    toastManager.error(msg, opts.title);
    opts.setStatus?.(`Error: ${msg}`, "error");
    return null;
  }
}
