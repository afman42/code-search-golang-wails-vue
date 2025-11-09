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
  watch,
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

// Toggle collapse/expand and scroll to bottom
const toggleCollapseAndScroll = () => {
  toggleCollapse();
  // Wait for the collapse state to update before scrolling
  nextTick(() => {
    if (!isCollapsed.value) {
      // Only scroll if expanded
      scrollToBottom();
    }
  });
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

// Watch for changes in logLevelFilter and reset the visible log indices
watch(logLevelFilter, () => {
  // Reset to show the most recent logs after filtering
  startLogIndex.value = 0;
  endLogIndex.value = Math.min(filteredLogs.value.length, 100);
  // Update the display immediately
  updateVisibleLogs();
  // Scroll to bottom after filter change
  nextTick(() => {
    if (logContent.value && autoScroll.value) {
      scrollToBottom();
    }
  });
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

  // Handle different message types from the backend
  if (data.type === "log" || data.type === "connected") {
    // Handle log messages from the backend
    if (typeof data.content === "string") {
      try {
        // Try to parse as JSON (from structured Logrus logs)
        const parsed = JSON.parse(data.content);

        if (parsed.msg.includes("Skipping")) {
          return;
        }
        // Extract fields from structured log format
        logEntry = {
          timestamp: new Date().toLocaleTimeString(),
          level: (parsed.level || parsed.Level || "info")
            .toString()
            .toUpperCase(),
          message: parsed.msg || parsed.message || data.content,
        };

        // Add timestamp if present in the parsed content
        if (parsed.time || parsed.timestamp || parsed.Time) {
          const timeVal = parsed.time || parsed.timestamp || parsed.Time;
          const timeObj = new Date(timeVal);
          if (!isNaN(timeObj.getTime())) {
            logEntry.timestamp = timeObj.toLocaleTimeString();
          }
        }
      } catch (e) {
        // If not JSON, treat as plain text
        logEntry = {
          timestamp: new Date().toLocaleTimeString(),
          level: "info",
          message: data.content,
        };
      }
    } else if (typeof data.content === "object") {
      // Handle object directly
      logEntry = {
        timestamp: data.content.time
          ? new Date(data.content.time).toLocaleTimeString()
          : new Date().toLocaleTimeString(),
        level: (data.content.level || data.content.Level || "info")
          .toString()
          .toUpperCase(),
        message:
          data.content.msg ||
          data.content.message ||
          JSON.stringify(data.content),
      };
    } else {
      logEntry = {
        timestamp: new Date().toLocaleTimeString(),
        level: "info",
        message: data.content
          ? String(data.content)
          : "Received log event without content",
      };
    }

    // Handle connection events
    if (data.type === "connected") {
      logEntry.message = data.content || "Connected to log stream";
    }
  } else if (data.type === "search-progress") {
    // Handle search progress updates from the backend
    const processedFiles = data.processedFiles || 0;
    const totalFiles = data.totalFiles || 1; // Default to 1 to prevent division by zero
    const resultsCount = data.resultsCount || 0;
    const currentFile = data.currentFile
      ? `Processing: ${data.currentFile.split("/").pop() || data.currentFile}`
      : "";
    const status = data.status || "in-progress";

    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `${status} - Progress: ${processedFiles}/${totalFiles} files (${Math.round((processedFiles / totalFiles) * 100)}%), ${resultsCount} results | ${currentFile}`,
    };
  } else if (data.type === "search-result") {
    // Handle search result updates from the backend
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `🔍 Found match in ${data.filePath || "unknown"} at line ${data.lineNum || 0}: ${(data.content || "").substring(0, 50)}${(data.content || "").length > 50 ? "..." : ""}`,
    };
  } else if (data.type === "editor-detection-start") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `🔍 Starting editor detection...`,
    };
  } else if (data.type === "editor-detection-progress") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `🔍 Editor: ${data.editor || "unknown"}, Available: ${data.available ? "✓" : "✗"}, Progress: ${(data.progress || 0).toFixed(1)}%`,
    };
  } else if (data.type === "editor-detection-complete") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `✅ Editor detection complete! Found ${data.totalFound || 0} editor(s)`,
    };
  } else if (data.type === "app-ready") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `🚀 Application ready! Timestamp: ${data.timestamp || Date.now()}`,
    };
  } else if (data.type === "search-progress" && data.status === "cancelled") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "warn",
      message: `⚠️ Search cancelled - Processed ${data.processedFiles || 0} of ${data.totalFiles || 0} files`,
    };
  } else if (data.type === "search-progress" && data.status === "completed") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: `✅ Search completed - Processed ${data.processedFiles || 0} files, found ${data.resultsCount || 0} results`,
    };
  } else if (data.timestamp && data.level && data.message) {
    // Handle direct Logrus-style format
    logEntry = {
      timestamp: new Date(data.timestamp).toLocaleTimeString(),
      level: (data.level || "info").toString().toUpperCase(),
      message: data.message || JSON.stringify(data),
    };
  } else if (data.type === "connected") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "info",
      message: "🔌 Connected to WebSocket",
    };
  } else if (data.type === "disconnected") {
    logEntry = {
      timestamp: new Date().toLocaleTimeString(),
      level: "warn",
      message: "🔌 Disconnected from WebSocket",
    };
  } else if (data.type === "ping" || data.type === "pong") {
    // Ignore ping/pong messages to reduce clutter
    return;
  } else {
    // Fallback for unrecognized data types - check if it has the typical backend log fields
    if (data.level && data.msg) {
      logEntry = {
        timestamp: data.time
          ? new Date(data.time).toLocaleTimeString()
          : new Date().toLocaleTimeString(),
        level: data.level.toUpperCase(),
        message: data.msg,
      };
    } else {
      // Generic handler
      logEntry = {
        timestamp: new Date().toLocaleTimeString(),
        level: "info",
        message: `Event: ${JSON.stringify(data)}`,
      };
    }
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

  // Check if auto-scroll should happen based on user's current scroll position
  if (logContent.value) {
    const { scrollTop, scrollHeight, clientHeight } = logContent.value;
    // Check if user is near the bottom (within 5px)
    const isUserAtBottom = scrollHeight - scrollTop - clientHeight < 5;

    // Only auto-scroll if the user was already at the bottom
    if (isUserAtBottom) {
      nextTick(() => {
        if (logContent.value) {
          logContent.value.scrollTop = logContent.value.scrollHeight;
        }
      });
    }
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
    // Only enable auto-scroll if user is near the bottom (within 5px)
    const isNearBottom = scrollHeight - scrollTop - clientHeight < 5;
    autoScroll.value = isNearBottom;
    showScrollButton.value = !isNearBottom;

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
  toggleLogStream();

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
