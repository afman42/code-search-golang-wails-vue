// logFrontend pushes frontend-origin logs into the backend's unified buffer so
// they appear in the same LogViewer as backend logs. When Wails is absent
// (browser/mock) it falls back to console so nothing is lost.

export function logFrontend(
  level: "debug" | "info" | "warn" | "error",
  message: string,
  fields?: Record<string, unknown>,
): void {
  // SAFETY: window.go is injected by Wails at runtime; absent in browser/mocks.
  const wailsFn = (window as unknown as { go?: { main?: { App?: { LogFrontend?: (a: string, b: string, c: Record<string, unknown>) => void } } } }).go
    ?.main?.App?.LogFrontend;
  if (typeof wailsFn === "function") {
    try {
      wailsFn(level, message, fields ?? {});
      return;
    } catch {
      // fall through to console
    }
  }
  const fn =
    level === "error"
      ? console.error
      : level === "warn"
        ? console.warn
        : level === "debug"
          ? console.debug
          : console.info;
  fn(`[frontend:${level}] ${message}`, fields ?? "");
}
