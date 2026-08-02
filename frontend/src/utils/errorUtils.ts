// Shared helper for extracting a human-readable message from an unknown thrown
// value. Wails binding rejections and plain JS errors both surface in catch
// blocks typed as `unknown`; this narrows without an unchecked `any`.
export function toErrorMessage(error: unknown, fallback = "Unknown error occurred"): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  if (error && typeof error === "object" && "message" in error) {
    const m = error.message;
    if (typeof m === "string") return m;
  }
  return fallback;
}

// Narrow an unknown value (typically a Wails event payload crossing the JS
// bridge) to a keyed record so individual fields can be read with typeof/`in`
// checks. Non-objects collapse to an empty record instead of throwing.
export function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : {};
}
