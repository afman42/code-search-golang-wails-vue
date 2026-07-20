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
          <EditorSelect
            :available-editors="data.availableEditors"
            @editor-select="handleEditorSelect($event, 'app.log')"
          />

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
          <div class="placeholder-hint">Start streaming to see live logs from the backend.</div>
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
import EditorSelect from "./EditorSelect.vue";
import { handleEditorSelect } from "../../utils/fileUtils";
import { ref, nextTick, onUpdated } from "vue";

// All log-streaming state and logic lives in the useLogStreaming composable.
// The component merely wires it to the template.
import { useLogStreaming } from "../../composables/useLogStreaming";

const props = defineProps<{ data: SearchState }>();

// Destructure everything the template needs from the composable.
const {
  logs,
  previewLogs,
  isStreaming,
  logLevelFilter,
  filteredLogs,
  toggleLogStream,
  clearLogs,
  addLogEntry,
} = useLogStreaming();

// Component-specific state (not part of the streaming logic)
const isCollapsed = ref(true); // Track whether logs are collapsed
const containerRef = ref<HTMLElement | null>(null);

// Toggle collapse/expand and scroll to bottom
const toggleCollapseAndScroll = () => {
  isCollapsed.value = !isCollapsed.value;
};

onUpdated(() => {
  // Ensure the log content is scrolled to the bottom when new logs are added
  nextTick(() => {
    if (containerRef.value) {
      containerRef.value.scrollTop = containerRef.value.scrollHeight;
    }
  });
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
