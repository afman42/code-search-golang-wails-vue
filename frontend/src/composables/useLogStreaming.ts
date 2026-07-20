import { ref, computed, shallowRef, onMounted, onUnmounted } from "vue";

// Wails bindings for log streaming — imported statically like all other Wails
// bindings in the codebase (see useSearch.ts).
import {
  GetInitialLogs as WailsGetInitialLogs,
  GetNewLogs as WailsGetNewLogs,
} from "../../wailsjs/go/main/App";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

// ---------------------------------------------------------------------------
// Log parsing helpers
//
// The backend sends LogMessage objects: { type: "log", content: ... }
// where content is either an already-parsed JSON object (from structured
// logrus logs) or a plain string (from non-JSON log lines).
// ---------------------------------------------------------------------------

/** Resolve the raw content value into a structured object or fallback string. */
function resolveContent(raw: any): Record<string, any> | string | null {
  if (typeof raw === "string") {
    try {
      const parsed = JSON.parse(raw);
      return parsed && typeof parsed === "object" ? parsed : raw;
    } catch {
      return raw; // Not JSON — keep as plain text
    }
  }
  if (typeof raw === "object" && raw !== null) return raw;
  return raw; // number / boolean / undefined — will be stringified downstream
}

/** Safely read a possibly-nested string field using a list of candidate keys. */
function readField(
  obj: Record<string, any>,
  candidates: string[],
): string | undefined {
  for (const key of candidates) {
    const val = obj[key];
    if (val !== undefined && val !== null) return String(val);
  }
  return undefined;
}

/** Return true when the content should be filtered out (noisy / internal). */
function isNoisy(raw: any): boolean {
  const msg =
    typeof raw === "string"
      ? raw
      : readField(raw, ["msg", "message"]) || "";
  return msg.includes("Skipping") || msg.includes("Sending file");
}

/** Extract a display-friendly log level, always uppercased. */
function pickLevel(obj: Record<string, any>): string {
  return (
    readField(obj, ["level", "Level", "LEVEL", "lvl"]) || "INFO"
  ).toUpperCase();
}

/** Extract the human-readable message from a parsed log object. */
function pickMessage(obj: Record<string, any>, fallback: string): string {
  return readField(obj, ["msg", "message"]) || fallback;
}

/** Format a timestamp from a log object, or return the current time. */
function formatTime(obj: Record<string, any>): string {
  const raw = readField(obj, ["time", "timestamp", "Time", "Timestamp"]);
  if (!raw) return new Date().toLocaleTimeString();
  const d = new Date(raw);
  return isNaN(d.getTime())
    ? new Date().toLocaleTimeString()
    : d.toLocaleTimeString();
}

/**
 * Parse a raw LogMessage (from the backend's Wails binding) into a LogEntry,
 * or return null to skip (noisy / internal messages).
 *
 * This is exported so Vue templates and tests can access it directly.
 */
export function parseLogEntry(data: any): LogEntry | null {
  const content = resolveContent(data.content);

  // Falsy / missing content — show a descriptive message rather than silently
  // dropping the entry so users know something happened.
  if (!content) {
    return {
      timestamp: new Date().toLocaleTimeString(),
      level: "INFO",
      message: String(content ?? "Received log event without content"),
    };
  }

  // Plain-text content — no further parsing needed
  if (typeof content === "string") {
    if (isNoisy(content)) return null;
    return {
      timestamp: new Date().toLocaleTimeString(),
      level: "INFO",
      message: content,
    };
  }

  // Structured JSON object from Logrus
  if (isNoisy(content)) return null;
  return {
    timestamp: formatTime(content),
    level: pickLevel(content),
    message: pickMessage(content, JSON.stringify(content)),
  };
}

// ---------------------------------------------------------------------------
// Composable
// ---------------------------------------------------------------------------

/**
 * useLogStreaming — encapsulates live log streaming from the Go backend.
 *
 * Provides:
 * - `logs` / `previewLogs` — reactive log entry arrays
 * - `isStreaming` — whether the polling interval is active
 * - `logLevelFilter` — current filter selection
 * - `filteredLogs` — computed, level-filtered view of `logs`
 * - `maxLogsToDisplay` — sliding window size (default 250)
 * - `toggleLogStream()` — start / stop polling
 * - `clearLogs()` — reset both live and preview logs
 * - `addLogEntry(data)` — manually inject a log entry (useful for tests)
 *
 * Lifecycle: starts polling on mount, stops on unmount.
 */
export function useLogStreaming() {
  // -----------------------------------------------------------------------
  // State
  // -----------------------------------------------------------------------

  const logs = shallowRef<LogEntry[]>([]);
  const previewLogs = shallowRef<LogEntry[]>([]);
  const isStreaming = ref(false);
  const logLevelFilter = ref("");
  const maxLogsToDisplay = ref(250);

  let pollingInterval: number | null = null;

  // -----------------------------------------------------------------------
  // Computed
  // -----------------------------------------------------------------------

  const filteredLogs = computed(() => {
    let result: LogEntry[];
    if (!logLevelFilter.value) {
      result = logs.value;
    } else {
      result = logs.value.filter(
        (log) =>
          log.level &&
          log.level.toLowerCase() === logLevelFilter.value.toLowerCase(),
      );
    }
    // Return the last maxLogsToDisplay entries to maintain a sliding window
    return result.slice(-maxLogsToDisplay.value);
  });

  // -----------------------------------------------------------------------
  // Private helpers
  // -----------------------------------------------------------------------

  function addLogEntryInternal(data: any) {
    const logEntry = parseLogEntry(data);
    if (!logEntry) return;

    // Create a new array to trigger shallowRef reactivity
    logs.value = [...logs.value, logEntry];

    // Limit logs to last 1000 entries for performance
    if (logs.value.length > 1000) {
      logs.value = logs.value.slice(-1000);
    }
  }

  const getInitialLogsWithRetry = async (maxRetries = 5) => {
    let attempts = 0;
    while (attempts < maxRetries) {
      try {
        const result = await WailsGetInitialLogs();

        if (Array.isArray(result)) {
          // Populate preview logs from the backend's in-memory buffer
          const preview: LogEntry[] = [];
          result.forEach((log: any) => {
            const entry = parseLogEntry(log);
            if (entry) preview.push(entry);
          });
          previewLogs.value = preview;

          // Also add to live logs for streaming
          result.forEach((log: any) => {
            addLogEntryInternal(log);
          });
          return; // Success, exit the retry loop
        } else {
          throw new Error("GetInitialLogs returned non-array");
        }
      } catch (error) {
        attempts++;
        console.log(`Attempt ${attempts} failed to get initial logs:`, error);
        if (attempts >= maxRetries) {
          console.error("Failed to get initial logs after", maxRetries, "attempts:", error);
          addLogEntryInternal({
            type: "log",
            content: `Failed to get initial logs after ${maxRetries} attempts: ${error}`,
          });
          return;
        }
        // Wait 500ms before retry
        await new Promise((resolve) => setTimeout(resolve, 500));
      }
    }
  };

  const getNewLogs = async () => {
    try {
      const result = await WailsGetNewLogs();

      if (Array.isArray(result)) {
        result.forEach((log: any) => {
          addLogEntryInternal(log);
        });
      }
    } catch (error) {
      console.error("Error fetching new logs via Wails binding:", error);
      addLogEntryInternal({
        type: "log",
        content: `Error fetching new logs: ${error}`,
      });
    }
  };

  // -----------------------------------------------------------------------
  // Public actions
  // -----------------------------------------------------------------------

  /** Manually inject a log entry (useful for tests and error reporting). */
  function addLogEntry(data: any) {
    addLogEntryInternal(data);
  }

  async function startPolling() {
    if (pollingInterval) {
      console.log("Polling already active, skipping new polling start");
      return;
    }

    // Get initial logs with retry mechanism
    await getInitialLogsWithRetry();

    // Start polling every 1 second
    pollingInterval = window.setInterval(async () => {
      await getNewLogs();
    }, 1000);

    console.log("Started polling for log updates");
    isStreaming.value = true;
  }

  function stopPolling() {
    if (pollingInterval) {
      clearInterval(pollingInterval);
      pollingInterval = null;
    }
    isStreaming.value = false;
  }

  function toggleLogStream() {
    if (isStreaming.value) {
      stopPolling();
    } else {
      startPolling();
    }
  }

  function clearLogs() {
    logs.value = [];
    previewLogs.value = [];
  }

  // -----------------------------------------------------------------------
  // Lifecycle
  // -----------------------------------------------------------------------

  onMounted(() => {
    startPolling();
  });

  onUnmounted(() => {
    stopPolling();
  });

  // -----------------------------------------------------------------------
  // Return
  // -----------------------------------------------------------------------

  return {
    // State (readonly consumer-facing aliases)
    logs,
    previewLogs,
    isStreaming,
    logLevelFilter,
    maxLogsToDisplay,
    // Computed
    filteredLogs,
    // Actions
    toggleLogStream,
    clearLogs,
    addLogEntry,
    // Exposed for advanced use
    stopPolling,
    startPolling,
  };
}
