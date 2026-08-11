<template>
  <div v-if="data.searchResults && Array.isArray(data.searchResults) && data.searchResults.length > 0" class="results-container">
    <div class="results-header">
      <h3>Search Results:</h3>
      <div class="results-summary">
        Found {{ resultsCount }} matches <span v-if="data.truncatedResults">(truncated)</span>
      </div>
    </div>

    <!-- Batch actions -->
    <ExportActions
      :total-results="resultsCount"
      :selected-count="selectedCount"
      :all-visible-selected="allVisibleSelected"
      @toggle-select-all="handleToggleSelectAll"
      @copy-selected="handleCopySelected"
      @export-results="handleExportResults"
    />

    <!-- Pagination controls (top) -->
    <PaginationControls
      :current-page="currentPage"
      :items-per-page="itemsPerPage"
      :total-results="resultsCount"
      :start-index="startIndex"
      :end-index="endIndex"
      @go-to-page="goToPage"
    />

    <!-- Result items -->
    <div v-for="(result, index) in paginatedResults" :key="result.filePath + result.lineNum + result.content.substring(0, 20)" class="result-item" :data-index="startIndex + index">
      <div class="result-header">
        <div class="file-info">
          <input type="checkbox" class="result-checkbox" :checked="isSelected(startIndex + index)" @change="handleToggleSelected(startIndex + index)" />
          <span class="file-path" @click="openFileLocation(result.filePath)" title="Click to show in folder">{{ formatFilePath(result.filePath) }}</span>
          <span class="line-num">Line {{ result.lineNum }}</span>
          <span class="matched-text" v-if="result.matchedText && result.matchedText !== data.query" >(Matched: "{{ result.matchedText }}")</span>
        </div>
        <div class="result-actions">
          <button class="view-btn" style="margin-right: 5px" @click="openFilePreview(result.filePath)" title="View full file">View</button>
          <button class="copy-btn" style="margin-right: 5px" @click="copyToClipboard(result.content)" title="Copy line">Copy</button>
          <EditorSelect :available-editors="data.availableEditors" @editor-select="handleEditorSelect($event, result.filePath)" />
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
    <PaginationControls
      :current-page="currentPage"
      :items-per-page="itemsPerPage"
      :total-results="resultsCount"
      :start-index="startIndex"
      :end-index="endIndex"
      @go-to-page="goToPage"
    />

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
import { ref, computed, watch } from "vue";
import type { SearchState, SearchResult } from "@/types";
import { CodeModal, EditorSelect, InlineDiffView } from "@/components/ui";
import { ReadFile, ExportSearchResults } from "@wails/go/main/App";
import { toastManager, useSelectionManager } from "@/composables";
import { handleEditorSelect, toErrorMessage } from "@/utils";

interface Props {
  data: SearchState;
  formatFilePath: (filePath: string) => string;
  openFileLocation: (filePath: string) => Promise<void>;
  copyToClipboard: (text: string) => Promise<boolean>;
}

const props = defineProps<Props>();

// Pagination state
const currentPage = ref(1);
const itemsPerPage = ref(10);

// Modal state
const showCodeModal = ref(false);
const selectedFilePath = ref("");
const selectedFileContent = ref("");

// Derived values
const resultsCount = computed(() => {
  return props.data.searchResults && Array.isArray(props.data.searchResults)
    ? props.data.searchResults.length
    : 0;
});

const totalPages = computed(() => {
  return Math.ceil(resultsCount.value / itemsPerPage.value);
});

const startIndex = computed(() => {
  return (currentPage.value - 1) * itemsPerPage.value;
});

const endIndex = computed(() => {
  return Math.min(startIndex.value + itemsPerPage.value, resultsCount.value);
});

// Results on the current page. InlineDiffView computes its own match/diff HTML
// from the raw content, so no pre-highlighting is done here.
const paginatedResults = computed(() => {
  if (!props.data.searchResults || !Array.isArray(props.data.searchResults)) {
    return [];
  }
  return props.data.searchResults.slice(startIndex.value, endIndex.value);
});

// Unique file paths across all results
const resultFilePaths = computed(() => {
  if (!props.data.searchResults || !Array.isArray(props.data.searchResults)) {
    return [];
  }
  return Array.from(new Set(props.data.searchResults.map((r) => r.filePath).filter(Boolean)));
});

// Selection manager composable
const selectionManager = useSelectionManager({ totalResults: resultsCount, startIndex, endIndex });
const { selectedCount, allVisibleSelected, isSelected, toggleSelected, toggleSelectAll } = selectionManager;

// Go to a specific page
const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value && page !== currentPage.value) {
    currentPage.value = page;
  }
};

// Reset pagination and clear selection when results change
watch(
  () => props.data.searchResults,
  () => {
    currentPage.value = 1;
    selectionManager.clearSelection();
  }
);

// File preview modal functions
const openFilePreview = async (filePath: string) => {
  try {
    selectedFilePath.value = filePath;
    const content = await ReadFile(filePath);
    selectedFileContent.value = content;
    showCodeModal.value = true;
    toastManager.success(`Loaded ${filePath}`);
  } catch (error: unknown) {
    const errorMsg = toErrorMessage(error);
    const errorCode = (error && typeof error === "object" && "code" in error) ? error.code : undefined;
    console.error("[SearchResults] Failed to read file:", { filePath, error: errorMsg, errorCode });
    if (errorMsg.includes("ReadFile") || errorMsg.includes("window")) {
      props.data.resultText = `Cannot read file in dev mode. Run 'wails dev' or 'wails build'. Error: ${errorMsg}`;
      toastManager.error(`Wails not running: Cannot read files without backend. Use 'wails dev' instead of 'npm run dev'.`);
    } else {
      props.data.resultText = `Failed to read file: ${errorMsg}`;
      toastManager.error(`File read error: ${errorMsg}`);
    }
    props.data.error = `File read error: ${errorMsg}`;
    showCodeModal.value = false;
  }
};

const closeFilePreview = () => {
  showCodeModal.value = false;
  selectedFilePath.value = "";
  selectedFileContent.value = "";
};

// Selection handlers
const handleToggleSelected = (idx: number) => {
  toggleSelected(idx);
};

const handleToggleSelectAll = () => {
  toggleSelectAll();
};

const handleCopySelected = async () => {
  const results = props.data.searchResults;
  if (!Array.isArray(results)) return;
  await selectionManager.copySelectedResults(results, props.copyToClipboard);
  if (selectionManager.isAnySelected()) {
    toastManager.success(`Copied ${selectedCount.value} results`);
  }
};

const handleExportResults = async () => {
  const results = props.data.searchResults;
  if (!Array.isArray(results) || results.length === 0) return;
  const savedPath = await selectionManager.exportSelectedResults(results, (toExport, fmt) => ExportSearchResults(toExport as SearchResult[], fmt));
  if (savedPath) {
    toastManager.success(`Exported ${results.length} results to ${savedPath}`);
  }
};

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

.result-item {
  margin-bottom: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
  background-color: var(--color-bg-secondary);
}

.file-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex: 1;
}

.result-checkbox {
  margin-right: 5px;
}

.file-path {
  cursor: pointer;
  color: var(--color-link);
  text-decoration: underline;
}

.file-path:hover {
  color: var(--color-link-hover);
}

.line-num {
  font-size: 0.85em;
  color: var(--color-text-muted);
}

.matched-text {
  font-size: 0.85em;
  color: var(--color-text-muted);
  font-style: italic;
}

.result-actions {
  display: flex;
  gap: 5px;
}

.view-btn,
.copy-btn {
  padding: 4px 8px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xs);
  cursor: pointer;
  font-size: 0.8em;
}

.view-btn:hover,
.copy-btn:hover {
  background-color: var(--color-bg-secondary);
}
</style>
