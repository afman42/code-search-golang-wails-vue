<template>
  <div
    v-if="
      data.searchResults &&
      Array.isArray(data.searchResults) &&
      data.searchResults.length > 0
    "
    class="results-container"
  >
    <div class="results-header">
      <h3>Search Results:</h3>
      <div class="results-summary">
        Found
        {{
          data.searchResults && Array.isArray(data.searchResults)
            ? data.searchResults.length
            : 0
        }}
        matches
        <span v-if="data.truncatedResults">(truncated)</span>
      </div>
    </div>

    <!-- Batch actions: select-all checkbox, copy selected, export -->
    <div v-if="totalResults > 0" class="batch-actions">
      <label class="select-all-label">
        <input type="checkbox" :checked="allVisibleSelected" @change="toggleSelectAll" />
        Select All (page)
      </label>
      <span class="selected-count" v-if="selectedCount > 0">{{ selectedCount }} selected</span>
      <button class="batch-btn" :disabled="selectedCount === 0" @click="copySelected">Copy Selected</button>
      <button class="batch-btn" :disabled="totalResults === 0" @click="exportResults('csv')">Export CSV</button>
      <button class="batch-btn" :disabled="totalResults === 0" @click="exportResults('json')">Export JSON</button>
    </div>

    <!-- Pagination controls -->
    <div v-if="totalPages > 1" class="pagination-controls">
      <div class="pagination-info">
        Showing {{ startIndex + 1 }}-{{ Math.min(endIndex, totalResults) }} of
        {{ totalResults }} results
      </div>
      <div class="pagination-actions">          <button
            class="pagination-btn"
            :disabled="currentPage === 1"
            @click="goToPage(currentPage - 1)"
          >
            Previous
          </button>
          <span class="page-info">{{ currentPage }} of {{ totalPages }}</span>
          <button
            class="pagination-btn"
            :disabled="currentPage === totalPages"
            @click="goToPage(currentPage + 1)"
          >
            Next
          </button>
      </div>
    </div>

    <div
      v-for="(result, index) in paginatedResults"
      :key="result.filePath + result.lineNum + result.content.substring(0, 20)"
      class="result-item"
      :data-index="startIndex + index"
    >
      <div class="result-header">
        <div class="file-info">
          <input
            type="checkbox"
            class="result-checkbox"
            :checked="isSelected(startIndex + index)"
            @change="toggleSelected(startIndex + index)"
          />
          <span
            class="file-path"
            @click="openFileLocation(result.filePath)"
            title="Click to show in folder"
          >
            {{ formatFilePath(result.filePath) }}
          </span>
          <span class="line-num">Line {{ result.lineNum }}</span>
          <span
            class="matched-text"
            v-if="result.matchedText && result.matchedText !== data.query"
          >
            (Matched: "{{ result.matchedText }}")
          </span>
        </div>
        <div class="result-actions">
          <button
            class="view-btn"
            style="margin-right: 5px"
            @click="openFilePreview(result.filePath)"
            title="View full file"
          >
            View
          </button>
          <button
            class="copy-btn"
            style="margin-right: 5px"
            @click="copyToClipboard(result.content)"
            title="Copy line"
          >
            Copy
          </button>
          <!-- Editor selection dropdown -->
          <EditorSelect
            :available-editors="data.availableEditors"
            @editor-select="handleEditorSelect($event, result.filePath)"
          />
        </div>
      </div>

      <!-- Display context before, match line with diff, and context after -->
      <InlineDiffView
        :content="result.content"
        :line-num="result.lineNum"
        :context-before="result.contextBefore"
        :context-after="result.contextAfter"
        :query="data.query"
        :case-sensitive="data.caseSensitive"
        :fuzzy-match-score="result.similarityScore"
        @copy="copyToClipboard"
      />
    </div>

    <!-- Pagination controls at the bottom -->
    <div v-if="totalPages > 1" class="pagination-controls bottom">
      <div class="pagination-info">
        Showing {{ startIndex + 1 }}-{{ Math.min(endIndex, totalResults) }} of
        {{ totalResults }} results
      </div>
      <div class="pagination-actions">          <button
            class="pagination-btn"
            :disabled="currentPage === 1"
            @click="goToPage(currentPage - 1)"
          >
            Previous
          </button>
          <span class="page-info">{{ currentPage }} of {{ totalPages }}</span>
          <button
            class="pagination-btn"
            :disabled="currentPage === totalPages"
            @click="goToPage(currentPage + 1)"
          >
            Next
          </button>
      </div>
    </div>

    <!-- Code Modal for viewing full files -->
    <CodeModal
      :is-visible="showCodeModal"
      :file-path="selectedFilePath"
      :file-content="selectedFileContent"
      :query="data.query"
      :files="resultFilePaths"
      @close="closeFilePreview"
      @copy="handleCopyFromModal"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, reactive } from "vue";
import type { SearchState } from "../../types/search";
import CodeModal from "./CodeModal.vue";
import EditorSelect from "./EditorSelect.vue";
import InlineDiffView from "./InlineDiffView.vue";
import { ReadFile, ExportSearchResults } from "../../../wailsjs/go/main/App";
import { toastManager } from "../../composables/useToast";
import { handleEditorSelect } from "../../utils/fileUtils";
import { toErrorMessage } from "../../utils/errorUtils";

// Define props with TypeScript
interface Props {
  data: SearchState;
  formatFilePath: (filePath: string) => string;
  highlightMatch: (text: string, query: string) => string;
  openFileLocation: (filePath: string) => Promise<void>;
  copyToClipboard: (text: string) => Promise<boolean>;
}
const props = defineProps<Props>();

// Pagination state
const currentPage = ref(1);
const itemsPerPage = ref(10); // Default to 10 items per page

// Modal state
const showCodeModal = ref(false);
const selectedFilePath = ref("");
const selectedFileContent = ref("");

// Computed properties for pagination
const totalResults = computed(() => {
  return props.data.searchResults && Array.isArray(props.data.searchResults)
    ? props.data.searchResults.length
    : 0;
});

const totalPages = computed(() => {
  return Math.ceil(totalResults.value / itemsPerPage.value);
});

const startIndex = computed(() => {
  return (currentPage.value - 1) * itemsPerPage.value;
});

const endIndex = computed(() => {
  return Math.min(
    startIndex.value + itemsPerPage.value,
    totalResults.value,
  );
});

// Highlight only the results on the current page. Highlighting runs a regex +
// DOMPurify sanitize per line, so doing it for every result (up to the 1000
// backend cap) when only 10 are visible is wasteful. Slice first, highlight the
// visible page only — this scales highlighting cost with page size, not total
// result count, and recomputes when the page, results, or query change.
const paginatedResults = computed(() => {
  if (
    !props.data.searchResults ||
    !Array.isArray(props.data.searchResults)
  ) {
    return [];
  }

  const query = props.data.query || "";
  return props.data.searchResults
    .slice(startIndex.value, endIndex.value)
    .map((result) => ({
      ...result,
      highlightedContent: props.highlightMatch(result.content || "", query),
      highlightedContextBefore: result.contextBefore.map((context) =>
        props.highlightMatch(context, query),
      ),
      highlightedContextAfter: result.contextAfter.map((context) =>
        props.highlightMatch(context, query),
      ),
    }));
});

// Method to change page
const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value && page !== currentPage.value) {
    // Change the page immediately — all results are already loaded client-side,
    // so there's no actual loading work to do. No fake spinner needed.
    currentPage.value = page;
  }
};

// Reset to first page and clear selection when results change.
watch(
  () => props.data.searchResults,
  () => {
    currentPage.value = 1;
    selectedIndices.clear();
  },
);

// Unique file paths across all results — feeds the file explorer tree in the
// code preview modal (CodeModal passes these down to TreeViewPanel).
const resultFilePaths = computed(() => {
  if (!props.data.searchResults || !Array.isArray(props.data.searchResults)) {
    return [];
  }
  return Array.from(
    new Set(
      props.data.searchResults
        .map((result) => result.filePath)
        .filter(Boolean),
    ),
  );
});


// Open file preview in modal
const openFilePreview = async (filePath: string) => {
  try {
    console.log('[SearchResults] Opening file preview:', filePath);
    
    // Set the selected file path
    selectedFilePath.value = filePath;

    // Read the file content
    const content = await ReadFile(filePath);
    
    console.log('[SearchResults] File loaded successfully', {
      filePath,
      contentLength: content?.length || 0,
    });
    
    selectedFileContent.value = content;

    // Show the modal
    showCodeModal.value = true;
    
    toastManager.success(`Loaded ${filePath}`);
  } catch (error: unknown) {
    const errorMsg = toErrorMessage(error);
    const errorCode =
      error && typeof error === "object" && "code" in error ? error.code : undefined;
    console.error("[SearchResults] Failed to read file:", {
      filePath,
      error: errorMsg,
      errorCode,
    });

    // Check if this is a Wails binding error
    if (errorMsg.includes('ReadFile') || errorMsg.includes('window')) {
      props.data.resultText = `Cannot read file in dev mode. Run 'wails dev' or 'wails build'. Error: ${errorMsg}`;
      toastManager.error(`Wails not running: Cannot read files without backend. Use 'wails dev' instead of 'npm run dev'.`);
    } else {
      props.data.resultText = `Failed to read file: ${errorMsg}`;
      toastManager.error(`File read error: ${errorMsg}`);
    }
    
    props.data.error = `File read error: ${errorMsg}`;
    // Close modal on error
    showCodeModal.value = false;
  }
};

// Close file preview modal
const closeFilePreview = () => {
  showCodeModal.value = false;
  selectedFilePath.value = "";
  selectedFileContent.value = "";
};

// --- Multi-select + batch export ---

// Set of selected result indices (global, not per-page).
const selectedIndices = reactive(new Set<number>());

const selectedCount = computed(() => selectedIndices.size);

const allVisibleSelected = computed(() => {
  if (totalResults.value === 0) return false;
  for (let i = startIndex.value; i < endIndex.value; i++) {
    if (!selectedIndices.has(i)) return false;
  }
  return true;
});

const isSelected = (idx: number) => selectedIndices.has(idx);

const toggleSelected = (idx: number) => {
  if (selectedIndices.has(idx)) {
    selectedIndices.delete(idx);
  } else {
    selectedIndices.add(idx);
  }
};

const toggleSelectAll = () => {
  if (allVisibleSelected.value) {
    for (let i = startIndex.value; i < endIndex.value; i++) {
      selectedIndices.delete(i);
    }
  } else {
    for (let i = startIndex.value; i < endIndex.value; i++) {
      selectedIndices.add(i);
    }
  }
};

const copySelected = async () => {
  const results = props.data.searchResults;
  if (!Array.isArray(results)) return;
  const selected = Array.from(selectedIndices)
    .sort((a, b) => a - b)
    .map((i) => results[i])
    .filter(Boolean);
  if (selected.length === 0) return;

  const text = selected
    .map((r) => `${r.filePath}:${r.lineNum} ${r.content}`)
    .join("\n");

  try {
    await navigator.clipboard.writeText(text);
    toastManager.success(`Copied ${selected.length} results`);
  } catch (err: unknown) {
    toastManager.error(toErrorMessage(err, "Copy failed"));
  }
};

const exportResults = async (format: string) => {
  const results = props.data.searchResults;
  if (!Array.isArray(results) || results.length === 0) return;

  // Export selected if any, otherwise all.
  let toExport = results;
  if (selectedIndices.size > 0) {
    toExport = Array.from(selectedIndices)
      .sort((a, b) => a - b)
      .map((i) => results[i])
      .filter(Boolean);
  }

  try {
    const savedPath = await ExportSearchResults(toExport, format);
    if (savedPath) {
      toastManager.success(`Exported ${toExport.length} results to ${savedPath}`);
    }
  } catch (err: unknown) {
    toastManager.error(toErrorMessage(err, "Export failed"), "Export Error");
  }
};

// Handle copy from modal
const handleCopyFromModal = () => {
  props.data.resultText = "File content copied to clipboard";
};
</script>

<style scoped>
.results-container {
  max-width: 800px;
  margin: var(--space-5) auto;
  padding: 0 var(--space-5);
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.results-summary {
  color: var(--color-text-muted);
  font-size: 0.9em;
}

.batch-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: 10px;
  padding: 8px 10px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  flex-wrap: wrap;
}

.select-all-label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 0.85em;
  cursor: pointer;
  user-select: none;
}

.selected-count {
  font-size: 0.8em;
  color: var(--color-accent);
  font-weight: 600;
}

.batch-btn {
  padding: 4px 10px;
  font-size: 0.8em;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.batch-btn:hover:not(:disabled) {
  background: var(--color-bg-hover);
  border-color: var(--color-accent);
}

.batch-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.result-checkbox {
  margin-right: 6px;
  cursor: pointer;
}

/* Pagination styles */
.pagination-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 15px 0;
  padding: 10px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.pagination-controls.bottom {
  margin-top: 15px;
  margin-bottom: var(--space-5);
}

.pagination-info {
  color: var(--color-text-muted);
  font-size: 0.9em;
}

.pagination-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pagination-btn {
  padding: 6px var(--space-3);
  background-color: var(--color-accent);
  color: var(--color-text-inverse);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.9em;
}

.pagination-btn:hover:not(:disabled) {
  background-color: var(--color-accent);
}

.pagination-btn:disabled {
  background-color: var(--color-border-medium);
  border-color: var(--color-border-medium);
  cursor: not-allowed;
  opacity: 0.6;
}

.page-info {
  color: var(--color-text-muted);
  font-size: 0.9em;
  margin: 0 5px;
}

.result-item {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  margin-bottom: 10px;
  padding: 10px;
  background-color: var(--color-bg-secondary);
  transition: box-shadow 0.2s;
}

.result-item:hover {
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 5px;
  flex-wrap: wrap;
  gap: 5px;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.file-path {
  font-weight: bold;
  color: var(--color-accent);
  cursor: pointer;
  text-decoration: underline;
}

.file-path:hover {
  color: var(--color-accent);
}

.line-num {
  color: var(--color-text-muted);
  font-size: 0.9em;
  background-color: var(--color-bg-tertiary);
  padding: 2px 6px;
  border-radius: 3px;
}

.matched-text {
  color: var(--color-success);
  font-size: 0.85em;
  font-style: italic;
  margin-left: 10px;
}

.copy-btn {
  background-color: var(--color-text-muted);
  color: var(--color-text-inverse);
  border: none;
  padding: var(--space-1) var(--space-2);
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8em;
}

.copy-btn:hover {
  background-color: var(--color-text-muted);
}

.result-content {
  font-family: monospace;
  padding: var(--space-2);
  background-color: var(--color-bg-secondary);
  border-left: 3px solid var(--color-accent);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}

.highlight {
  background-color: var(--color-warning);
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

/* Spinner animation for pagination buttons */
.spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-radius: 50%;
  border-top-color: #fff;
  animation: spin 1s ease-in-out infinite;
  margin-right: var(--space-1);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
