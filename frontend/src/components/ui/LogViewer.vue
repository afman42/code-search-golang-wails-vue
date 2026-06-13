<template>
  <div class="log-viewer-container" :class="{ 'log-collapsed': isCollapsed }">
    <!-- Toggle button to expand/collapse logs -->
    <div class="log-toggle-button" @click="toggleCollapseAndScroll">
      <svg
        v-if="!isCollapsed"
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        class="toggle-icon"
      >
        <polyline points="18 15 12 9 6 15"></polyline>
      </svg>
      <svg
        v-else
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        class="toggle-icon"
      >
        <polyline points="6 9 12 15 18 9"></polyline>
      </svg>
    </div>

    <!-- Log content - only shown when not collapsed -->
    <div v-if="!isCollapsed" class="log-content-wrapper">
      <div class="log-header">
        <h3>Live Log Viewer</h3>
        <div class="log-controls">
          <select
            class="editor-select"
            @change="handleEditorSelect($event, 'app.log')"
            title="Open in editor"
          >
            <option value="">Editor...</option>
            <option v-if="data.availableEditors.vscode" value="vscode">
              VSCode
            </option>
            <option v-if="data.availableEditors.vscodium" value="vscodium">
              VSCodium
            </option>
            <option v-if="data.availableEditors.sublime" value="sublime">
              Sublime Text
            </option>
            <option v-if="data.availableEditors.atom" value="atom">Atom</option>
            <option v-if="data.availableEditors.jetbrains" value="jetbrains">
              JetBrains
            </option>
            <option v-if="data.availableEditors.geany" value="geany">
              Geany
            </option>
            <option v-if="data.availableEditors.goland" value="goland">
              GoLand
            </option>
            <option v-if="data.availableEditors.pycharm" value="pycharm">
              PyCharm
            </option>
            <option v-if="data.availableEditors.intellij" value="intellij">
              IntelliJ IDEA
            </option>
            <option v-if="data.availableEditors.webstorm" value="webstorm">
              WebStorm
            </option>
            <option v-if="data.availableEditors.phpstorm" value="phpstorm">
              PhpStorm
            </option>
            <option v-if="data.availableEditors.clion" value="clion">
              CLion
            </option>
            <option v-if="data.availableEditors.rider" value="rider">
              Rider
            </option>
            <option
              v-if="data.availableEditors.androidstudio"
              value="androidstudio"
            >
              Android Studio
            </option>
            <option v-if="data.availableEditors.emacs" value="emacs">
              Emacs
            </option>
            <option v-if="data.availableEditors.neovide" value="neovide">
              Neovide
            </option>
            <option v-if="data.availableEditors.codeblocks" value="codeblocks">
              Code::Blocks
            </option>
            <option v-if="data.availableEditors.devcpp" value="devcpp">
              Dev-C++
            </option>
            <option
              v-if="data.availableEditors.notepadplusplus"
              value="notepadplusplus"
            >
              Notepad++
            </option>
            <option
              v-if="data.availableEditors.visualstudio"
              value="visualstudio"
            >
              Visual Studio
            </option>
            <option v-if="data.availableEditors.eclipse" value="eclipse">
              Eclipse
            </option>
            <option v-if="data.availableEditors.netbeans" value="netbeans">
              NetBeans
            </option>
            <option value="default">System Default</option>
          </select>

          <button @click="toggleLogStream" class="btn btn-primary">
            {{ isStreaming ? "Stop Streaming" : "Start Streaming" }}
          </button>
          <button @click="clearLogs" class="btn btn-secondary">Clear</button>
          <select v-model="logLevelFilter" class="log-filter">
            <option value="">All Levels</option>
            <option value="trace">Trace</option>
            <option value="debug">Debug</option>
            <option value="info">Info</option>
            <option value="warn">Warn</option>
            <option value="error">Error</option>
            <option value="fatal">Fatal</option>
          </select>
        </div>
      </div>
      <div ref="containerRef" class="log-content">
        <!-- Preview: show actual log content from backend file when no live logs yet -->
        <div v-if="logs.length === 0 && previewLogs.length > 0" class="log-preview">
          <div class="preview-header">
            <span class="preview-badge">PREVIEW</span>
            <span class="preview-source">logs/app.log</span>
          </div>
          <div class="preview-entries">
            <div
              v-for="(log, index) in previewLogs"
              :key="'prev-' + index"
              :class="['log-entry', 'log-preview-entry', `log-${log.level || 'info'}`]"
            >
              <span class="log-timestamp">[{{ log.timestamp }}]</span>
              <span class="log-level">[{{ log.level || 'INFO' }}]</span>
              <span class="log-message">{{ log.message }}</span>
            </div>
          </div>
        </div>
        <!-- Fallback placeholder when no logs at all -->
        <div v-else-if="logs.length === 0" class="log-placeholder">
          <div class="placeholder-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="4 17 10 11 4 5"></polyline>
              <line x1="12" y1="19" x2="20" y2="19"></line>
            </svg>
          </div>
          <div class="placeholder-title">No logs yet</div>
          <div class="placeholder-hint">Start streaming or check that the backend polling server is running.</div>
        </div>
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          :class="[
            'log-entry',
            `log-${log.level || 'info'}`,
            { placeholder: !log.message && !log.timestamp },
          ]"
        >
          <span class="log-timestamp">[{{ log.timestamp }}]</span>
          <span class="log-level">[{{ log.level || "INFO" }}]</span>
          <span class="log-message">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { SearchState } from "../../types/search";
import { handleEditorSelect } from "../../utils/fileUtils";
import {
  ref,
  onMounted,
  onUnmounted,
  computed,
  nextTick,
  shallowRef,
  onUpdated,
} from "vue";

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

let props = defineProps<{ data: SearchState }>();

const logs = shallowRef<LogEntry[]>([]); // Using shallowRef for better performance
const previewLogs = shallowRef<LogEntry[]>([]); // Preview logs loaded from backend file
const isStreaming = ref(false);
const logLevelFilter = ref("");
const isCollapsed = ref(true); // Track whether logs are collapsed
const containerRef = ref<HTMLElement | null>(null);

// For performance optimization without virtual scrolling, we'll just limit the logs shown
const maxLogsToDisplay = ref(250);

// Toggle collapse/expand and scroll to bottom
const toggleCollapseAndScroll = () => {
  isCollapsed.value = !isCollapsed.value;
};

const filteredLogs = computed(() => {
  let result;
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

let pollingInterval: number | null = null;
let lastTimestamp = 0; // Track the timestamp of the last poll

const startPolling = async () => {
  // Prevent multiple polling intervals
  if (pollingInterval) {
    console.log("Polling already active, skipping new polling start");
    return;
  }

  // Get initial logs with retry mechanism in case server is not ready yet
  await getInitialLogsWithRetry();

  // Start polling every 1 second
  pollingInterval = window.setInterval(async () => {
    await getNewLogs();
  }, 1000);

  console.log("Started polling for log updates");
  isStreaming.value = true;
};

// Helper function to get initial logs with retry
const getInitialLogsWithRetry = async (maxRetries = 5) => {
  let attempts = 0;
  while (attempts < maxRetries) {
    try {
      const response = await fetch("http://localhost:34116/initial", {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        const data = await response.json();
        // Populate preview logs from the backend file content
        const preview: LogEntry[] = [];
        data.forEach((log: any) => {
          const entry = parseLogEntry(log);
          if (entry) preview.push(entry);
        });
        previewLogs.value = preview;

        // Also add to live logs for streaming
        data.forEach((log: any) => {
          addLogEntry(log);
        });
        return; // Success, exit the retry loop
      } else {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
    } catch (error) {
      attempts++;
      console.log(`Attempt ${attempts} failed to get initial logs:`, error);
      if (attempts >= maxRetries) {
        console.error("Failed to get initial logs after", maxRetries, "attempts:", error);
        addLogEntry({
          type: "log",
          content: `Failed to connect to log server after ${maxRetries} attempts: ${error}`,
        });
        return;
      }
      // Wait 500ms before retry
      await new Promise(resolve => setTimeout(resolve, 500));
    }
  }
};

const stopPolling = () => {
  if (pollingInterval) {
    clearInterval(pollingInterval);
    pollingInterval = null;
  }
  isStreaming.value = false;
};

const toggleLogStream = () => {
  if (isStreaming.value) {
    stopPolling();
  } else {
    startPolling();
  }
};


const getNewLogs = async () => {
  try {
    // In Wails applications, the frontend runs inside a WebView, so we access the server directly
    // The backend polling server runs on localhost:34116
    const response = await fetch("http://localhost:34116/poll", {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const logs = await response.json();

    // Process each new log entry
    logs.forEach((log: any) => {
      addLogEntry(log);
    });
  } catch (error) {
    console.error("Error fetching new logs:", error);
    addLogEntry({
      type: "log",
      content: `Error fetching new logs: ${error}`,
    });
  }
};
// -------------------------------------------------------------------------
// Log parsing helpers
//
// The polling server sends LogMessage objects: { type: "log", content: ... }
// where content is either an already-parsed JSON object (from structured
// logrus logs) or a plain string (from non-JSON log lines).
// -------------------------------------------------------------------------

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
  return msg.includes("Skipping");
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

/** Parse a raw polling-server LogMessage into a LogEntry, or null to skip. */
function parseLogEntry(data: any): LogEntry | null {
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

function addLogEntry(data: any) {
  const logEntry = parseLogEntry(data);
  if (!logEntry) return;

  // Create a new array to trigger reactivity
  logs.value = [...logs.value, logEntry];

  // Limit logs to last 1000 entries for performance to allow for sufficient history for filtering
  if (logs.value.length > 1000) {
    logs.value = logs.value.slice(-1000);
  }
}
const clearLogs = () => {
  logs.value = [];
  previewLogs.value = [];
};

onMounted(() => {
  toggleLogStream();
});

onUpdated(() => {
  // Ensure the log content is scrolled to the bottom when new logs are added
  nextTick(() => {
    if (containerRef.value) {
      containerRef.value.scrollTop = containerRef.value.scrollHeight;
    }
  });
});

onUnmounted(() => {
  // Make sure to properly stop polling to prevent memory leaks
  stopPolling();
});
</script>

<style scoped>
.log-viewer-container {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  border: 1px solid #ddd;
  border-radius: 8px 8px 0 0;
  margin: 0;
  overflow: hidden;
  z-index: 1000;
  transition: height 0.3s ease;
  background-color: #fff;
}

.log-viewer-container.log-collapsed {
  height: 40px;
}

.log-content-wrapper {
  height: 250px;
  display: flex;
  flex-direction: column;
}

.log-toggle-button {
  position: absolute;
  top: 5px;
  right: 10px;
  width: 30px;
  height: 30px;
  background-color: #f8f9fa;
  border: 1px solid #ddd;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 1001;
  transition: background-color 0.2s;
}

.log-toggle-button:hover {
  background-color: #e9ecef;
}

.toggle-icon {
  width: 16px;
  height: 16px;
  transition: transform 0.2s ease;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem;
  background-color: #f8f9fa;
  border-bottom: 1px solid #ddd;
}

.log-header h3 {
  margin: 0;
  font-size: 1rem;
}

.log-controls {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.log-filter {
  padding: 0.25rem;
  border: 1px solid #ccc;
  border-radius: 4px;
  font-size: 0.875rem;
}

.btn {
  padding: 0.25rem 0.5rem;
  border: 1px solid transparent;
  border-radius: 4px;
  font-size: 0.875rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-primary {
  background-color: #007bff;
  color: white;
}

.btn-primary:hover {
  background-color: #0056b3;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background-color: #545b62;
}

.log-content {
  flex: 1;
  overflow-y: auto;
  font-family: "Courier New", monospace;
  font-size: 0.875rem;
  padding: 0.5rem;
  background-color: #1e1e1e;
  color: #d4d4d4;
  position: relative;
}

.log-preview {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.preview-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  border-bottom: 1px solid #333;
  background-color: #252526;
  flex-shrink: 0;
}

.preview-badge {
  font-size: 0.65rem;
  font-weight: 700;
  color: #888;
  letter-spacing: 0.08em;
  border: 1px solid #555;
  border-radius: 3px;
  padding: 0.1rem 0.35rem;
  text-transform: uppercase;
}

.preview-source {
  font-size: 0.7rem;
  color: #666;
}

.preview-entries {
  flex: 1;
  overflow-y: auto;
  padding: 0.25rem 0.5rem;
  opacity: 0.55;
}

.log-preview-entry {
  font-size: 0.8rem;
  line-height: 1.6;
}

.log-preview-entry .log-level,
.log-preview-entry .log-timestamp {
  opacity: 0.7;
}

.log-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #888;
  text-align: center;
  padding: 2rem;
  gap: 0.75rem;
  user-select: none;
}

.placeholder-icon {
  opacity: 0.5;
  margin-bottom: 0.25rem;
}

.placeholder-title {
  font-size: 1rem;
  font-weight: 600;
  color: #aaa;
}

.placeholder-hint {
  font-size: 0.8rem;
  color: #666;
  max-width: 280px;
  line-height: 1.4;
}

.log-entry {
  margin: 0.125rem 0;
  padding: 0.125rem 0;
  white-space: pre-wrap;
  word-break: break-word;
}

.log-entry.placeholder {
  color: transparent;
  min-height: 1.5em;
  border-bottom: 1px solid transparent;
}

.log-status {
  display: flex;
  align-items: center;
  margin-right: 1rem;
}

.status-badge {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: bold;
  text-align: center;
  min-width: 80px;
}

.status-active {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.status-inactive {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.status-text {
  font-size: 0.65rem;
  text-transform: uppercase;
}

.status-time {
  font-size: 0.6rem;
  font-weight: normal;
  margin-top: 0.1rem;
  opacity: 0.8;
}

.log-debug {
  color: #9cdcfe;
}

.log-info {
  color: #ce9178;
}

.log-warn {
  color: #ffcc02;
}

.log-error {
  color: #f44747;
}

.log-trace {
  color: #b2b2b2; /* Light gray for trace */
}

.log-fatal {
  color: #ff0000; /* Bright red for fatal */
}

.scroll-to-bottom {
  position: absolute;
  bottom: 10px;
  right: 10px;
  width: 30px;
  height: 30px;
  background-color: rgba(0, 123, 255, 0.8);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-weight: bold;
  z-index: 10;
}
</style>
