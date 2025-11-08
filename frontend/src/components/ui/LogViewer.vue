<template>
  <div class="log-viewer-container" :class="{ 'log-collapsed': isCollapsed }">
    <!-- Toggle button to expand/collapse logs -->
    <div class="log-toggle-button" @click="toggleCollapse">
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
      <div class="log-content" ref="logContent" @scroll="onScroll">
        <div v-if="logs.length === 0" class="no-logs">No logs to display</div>
        <div
          v-for="(log, index) in displayLogs"
          :key="index + startLogIndex"
          :class="['log-entry', `log-${log.level || 'info'}`]"
        >
          <span class="log-timestamp">[{{ log.timestamp }}]</span>
          <span class="log-level">[{{ log.level || "INFO" }}]</span>
          <span class="log-message">{{ log.message }}</span>
        </div>
        <div
          v-if="showScrollButton"
          class="scroll-to-bottom"
          @click="scrollToBottom"
        >
          ↓
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ref,
  onMounted,
  onUnmounted,
  computed,
  nextTick,
  shallowRef,
} from "vue";

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

const logs = shallowRef<LogEntry[]>([]); // Using shallowRef for better performance
const isStreaming = ref(false);
const logLevelFilter = ref("");
const logContent = ref<HTMLDivElement | null>(null);
const showScrollButton = ref(false);
const autoScroll = ref(true);
const intersectionObserver = ref<IntersectionObserver | null>(null);
const isCollapsed = ref(true); // Track whether logs are collapsed

// Virtual scrolling: only show logs that are visible or near viewport
const visibleLogsThreshold = ref(100); // Show 100 logs around current viewport
const startLogIndex = ref(0);
const endLogIndex = ref(100); // Will be updated dynamically

// Toggle collapse/expand of the log viewer
const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value;
};

const filteredLogs = computed(() => {
  if (!logLevelFilter.value) {
    return logs.value;
  }
  return logs.value.filter(
    (log) =>
      log.level &&
      log.level.toLowerCase() === logLevelFilter.value.toLowerCase(),
  );
});

// Compute the logs that should be displayed for virtual scrolling
const displayLogs = computed(() => {
  const filtered = filteredLogs.value;
  const start = Math.max(0, startLogIndex.value);
  const end = Math.min(filtered.length, endLogIndex.value);
  return filtered.slice(start, end);
});

let ws: WebSocket | null = null;

const connectWebSocket = () => {
  // Prevent multiple connections
  if (
    ws &&
    (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN)
  ) {
    console.log("WebSocket connection already active, skipping new connection");
    return;
  }

  // In Wails applications, the WebSocket server runs on localhost:34116
  // The frontend runs in a webview which may have various hostnames
  const wsUrl = "ws://localhost:34116/ws";

  // Log connection attempt
  console.log(`Attempting to connect to WebSocket: ${wsUrl}`);

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log("Connected to log stream");
    isStreaming.value = true;
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      addLogEntry(data);
    } catch (e) {
      console.warn(
        "Could not parse WebSocket message as JSON, treating as string:",
        event.data,
      );
      // If it's not JSON, treat as a simple log message
      addLogEntry({
        type: "log",
        content: event.data,
      });
    }
  };

  ws.onclose = () => {
    console.log("Disconnected from log stream");
    isStreaming.value = false;
    // Attempt to reconnect after 3 seconds
    if (isStreaming.value) {
      console.log("Attempting to reconnect to WebSocket in 3 seconds...");
      setTimeout(connectWebSocket, 3000);
    }
  };

  ws.onerror = (error) => {
    console.error("WebSocket error:", error);
    // Log to internal logs as well
    addLogEntry({
      type: "log",
      content: `WebSocket connection error: ${error}`,
    });
  };
};

const disconnectWebSocket = () => {
  if (ws) {
    // Remove all event listeners to prevent any further processing
    ws.onopen = null;
    ws.onmessage = null;
    ws.onclose = null;
    ws.onerror = null;

    ws.close();
    ws = null;
  }
  isStreaming.value = false;
};

const toggleLogStream = () => {
  if (isStreaming.value) {
    disconnectWebSocket();
  } else {
    connectWebSocket();
  }
};

const addLogEntry = (data: any) => {
  let logEntry: LogEntry;

  if (data.type === "log" || data.type === "connected") {
    // Initialize with default values
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message:
        typeof data.content === "string"
          ? data.content
          : JSON.stringify(data.content),
    };

    // Handle different content formats
    if (typeof data.content === "string") {
      try {
        // Try to parse as JSON (from structured Logrus logs)
        const parsed = JSON.parse(data.content);

        // Skip if this is a "Skipping file" message to reduce noise
        if (parsed.msg && parsed.msg.includes("Skipping")) {
          return; // Don't add this log entry
        }
        // Extract fields from Logrus JSON format - handle all possible level field names
        if (parsed.level) {
          logEntry.level = parsed.level.toString().toUpperCase();
        } else if (parsed.Level) {
          logEntry.level = parsed.Level.toString().toUpperCase();
        } else if (parsed.lvl) {
          logEntry.level = parsed.lvl.toString().toUpperCase();
        }

        // Extract message - handle all possible message field names
        if (parsed.msg) {
          logEntry.message = parsed.msg;
        } else if (parsed.message) {
          logEntry.message = parsed.message;
        } else if (parsed.Message) {
          logEntry.message = parsed.Message;
        } else {
          // If no specific message field, use the whole object as message
          logEntry.message = data.content;
        }

        // Extract timestamp - handle all possible timestamp field names
        if (parsed.time || parsed.timestamp || parsed.Time) {
          const timeValue = parsed.time || parsed.timestamp || parsed.Time;
          const logTime = new Date(timeValue);
          if (!isNaN(logTime.getTime())) {
            logEntry.timestamp = logTime.toLocaleTimeString();
          }
        }

        // Extract other fields that might be useful for debugging
        if (parsed.file || parsed.func || parsed.line) {
          const locationInfo = [];
          if (parsed.file) locationInfo.push(parsed.file);
          if (parsed.func) locationInfo.push(parsed.func);
          if (parsed.line) locationInfo.push(`line ${parsed.line}`);

          if (locationInfo.length > 0) {
            logEntry.message += ` [${locationInfo.join(":")}]`;
          }
        }
      } catch (e) {
        // If not JSON, use as plain text message
        logEntry.message = data.content;
      }
    } else if (typeof data.content === "object" && data.content !== null) {
      // Handle object content directly
      if (data.content.level) {
        logEntry.level = data.content.level.toString().toUpperCase();
      } else if (data.content.Level) {
        logEntry.level = data.content.Level.toString().toUpperCase();
      } else if (data.content.lvl) {
        logEntry.level = data.content.lvl.toString().toUpperCase();
      }

      if (data.content.msg) {
        logEntry.message = data.content.msg;
      } else if (data.content.message) {
        logEntry.message = data.content.message;
      } else if (data.content.Message) {
        logEntry.message = data.content.Message;
      } else {
        logEntry.message = JSON.stringify(data.content);
      }

      if (data.content.time || data.content.timestamp || data.content.Time) {
        const timeValue =
          data.content.time || data.content.timestamp || data.content.Time;
        const logTime = new Date(timeValue);
        if (!isNaN(logTime.getTime())) {
          logEntry.timestamp = logTime.toLocaleTimeString();
        }
      }
    }

    // Add connection message if it's a connection event
    if (data.type === "connected" && !logEntry.message.includes("Connected")) {
      logEntry.message = data.content || "Connected to log stream";
    }
  } else if (
    data.type === "search-progress" ||
    (data.processedFiles !== undefined && data.totalFiles !== undefined)
  ) {
    // Handle search progress updates with proper null checks
    // This handles both explicit "search-progress" type and objects with progress data
    const processedFiles = data.processedFiles || 0;
    const totalFiles = data.totalFiles || 0;
    const resultsCount = data.resultsCount || 0;

    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `Search progress: ${processedFiles}/${totalFiles} files, ${resultsCount} results`,
    };
  } else if (data.type === "search-result" || data.filePath) {
    // Handle search result updates with proper null checks
    // This handles both explicit "search-result" type and objects with filePath data
    const filePath = data.filePath || "unknown";
    const lineNum = data.LineNum || 0;

    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `Found match in ${filePath}:${lineNum}`,
    };
  } else {
    // Handle unknown message types
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: JSON.stringify(data),
    };
  }

  logs.value.push(logEntry);

  // Limit logs to prevent memory issues
  if (logs.value.length > 1000) {
    logs.value = logs.value.slice(-500); // Keep last 500 logs
  }

  // Update visible logs range for virtual scrolling
  if (logContent.value) {
    updateVisibleLogs();
  }

  // Auto-scroll if user is at the bottom
  if (autoScroll.value) {
    nextTick(() => scrollToBottom());
  }
};

const clearLogs = () => {
  logs.value = [];
  startLogIndex.value = 0;
  endLogIndex.value = 100;
};

const onScroll = () => {
  if (logContent.value) {
    const { scrollTop, scrollHeight, clientHeight } = logContent.value;
    // If we're near the bottom, enable auto-scroll
    autoScroll.value = scrollHeight - scrollTop - clientHeight < 5;
    showScrollButton.value = !autoScroll.value;

    // Update visible logs when scrolling for virtual scrolling
    updateVisibleLogs();
  }
};

// Update the visible log range based on scroll position
const updateVisibleLogs = () => {
  if (!logContent.value) return;

  const container = logContent.value;
  const scrollTop = container.scrollTop;
  const containerHeight = container.clientHeight;

  // Estimate average log entry height (can be adjusted as needed)
  const avgLogHeight = 20; // px

  // Calculate which logs are visible
  const startIdx =
    Math.floor(scrollTop / avgLogHeight) - visibleLogsThreshold.value;
  const endIdx =
    startIdx +
    Math.ceil(containerHeight / avgLogHeight) +
    visibleLogsThreshold.value * 2;

  startLogIndex.value = Math.max(0, startIdx);
  endLogIndex.value = Math.min(filteredLogs.value.length, endIdx);
};

const scrollToBottom = () => {
  if (logContent.value) {
    logContent.value.scrollTop = logContent.value.scrollHeight;
    autoScroll.value = true;
    showScrollButton.value = false;
  }
};

// Set up Intersection Observer for improved performance
const setupIntersectionObserver = () => {
  if (!logContent.value) return;

  // Create the observer to watch for scroll events efficiently
  intersectionObserver.value = new IntersectionObserver(
    (entries) => {
      // When the last log entry comes into view, we may need to update the display range
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          // Trigger an update to the visible logs range
          updateVisibleLogs();
        }
      });
    },
    {
      root: logContent.value,
      rootMargin: "100px", // Load logs 100px before they become visible
      threshold: 0.1,
    },
  );
};

onMounted(() => {
  // Don't auto-start streaming logs - let user control it via the UI
  // This prevents resource usage and potential conflicts when component is mounted

  // Set up intersection observer for performance
  setupIntersectionObserver();

  // Update visible logs when the logs array changes
  updateVisibleLogs();
});

onUnmounted(() => {
  // Make sure to properly disconnect WebSocket to prevent memory leaks
  disconnectWebSocket();

  // Disconnect the intersection observer
  if (intersectionObserver.value) {
    intersectionObserver.value.disconnect();
    intersectionObserver.value = null;
  }

  logs.value = [];
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

.no-logs {
  color: #888;
  text-align: center;
  padding: 1rem;
}

.log-entry {
  margin: 0.125rem 0;
  padding: 0.125rem 0;
  white-space: pre-wrap;
  word-break: break-word;
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
