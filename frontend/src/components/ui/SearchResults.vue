<template>
  <div v-if="data.searchResults && Array.isArray(data.searchResults) && data.searchResults.length > 0" class="results-container">
    <div class="results-header">
      <h3>Search Results:</h3>
      <div class="results-summary">
        Found {{ resultsCount }} matches <span v-if="data.truncatedResults">(truncated)</span>
      </div>
    </div>

    <!-- Find & Replace (hidden under regex mode — backend rejects regex replace) -->
    <div v-if="!data.useRegex" class="replace-row">
      <input
        v-model="replacement"
        class="replace-input"
        placeholder="Replace matches with…"
        :disabled="isReplacing"
      />
      <button
        class="replace-btn"
        @click="previewReplace"
        :disabled="isReplacing || !replacement"
      >Preview Replace</button>
      <button
        class="replace-btn apply"
        @click="applyReplace"
        :disabled="isReplacing || !preview || preview.filesChanged === 0"
      >Apply {{ preview && preview.filesChanged > 0 ? preview.linesChanged : '' }}</button>
    </div>

    <!-- Replace preview: old → new line diffs from the dry-run -->
    <div v-if="preview && preview.files.length > 0" class="replace-preview">
      <div class="replace-preview-header">
        <span>{{ preview.filesChanged }} file(s), {{ preview.linesChanged }} line(s) to change</span>
        <button class="replace-clear" @click="clearPreview">×</button>
      </div>
      <ul class="replace-preview-list">
        <li v-for="file in preview.files.slice(0, 20)" :key="file.filePath + file.lineNum" class="replace-preview-item">
          <span class="replace-file">{{ formatFilePath(file.filePath) }}:{{ file.lineNum }}</span>
          <span class="replace-old" title="before">{{ file.oldLine }}</span>
          <span class="replace-arrow">→</span>
          <span class="replace-new" title="after">{{ file.newLine }}</span>
        </li>
      </ul>
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
      class="bottom"
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
import CodeModal from "./CodeModal.vue";
import EditorSelect from "./EditorSelect.vue";
import InlineDiffView from "./InlineDiffView.vue";
import ExportActions from "./ExportActions.vue";
import PaginationControls from "./PaginationControls.vue";
import { ReadFile, ExportSearchResults } from "@wails/go/main/App";
import { toastManager, useSelectionManager, useReplace } from "@/composables";
import { handleEditorSelect, toErrorMessage } from "@/utils";

interface Props {
  data: SearchState;
  formatFilePath: (filePath: string) => string;
  openFileLocation: (filePath: string) => Promise<void>;
  copyToClipboard: (text: string) => Promise<boolean>;
  onSearch?: () => Promise<void>;
}

const props = defineProps<Props>();

// Find & Replace: literal replacement with dry-run preview + explicit apply.
// Re-runs the search after apply so results reflect the changed files.
const { replacement, preview, isReplacing, previewReplace, applyReplace, clearPreview } =
  useReplace(props.data, props.onSearch || (async () => {}));

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

.result-checkbox {
  margin-right: 6px;
  cursor: pointer;
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

/* Replace row */
.replace-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 10px;
  padding: 8px 10px;
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-sm);
}
.replace-input {
  flex: 1;
  padding: 0.375rem 0.5rem;
  font-size: 0.85rem;
  font-family: var(--font-mono, monospace);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
}
.replace-input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px var(--color-accent-light);
}
.replace-btn {
  padding: 0.375rem 0.75rem;
  font-size: 0.85rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
  cursor: pointer;
  white-space: nowrap;
}
.replace-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.replace-btn.apply {
  background: var(--color-accent);
  color: var(--color-text-inverse);
  border-color: var(--color-accent);
}
.replace-btn.apply:disabled {
  opacity: 0.5;
}

/* Replace preview */
.replace-preview {
  margin-bottom: 10px;
  border: 1px solid var(--color-warning);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.replace-preview-header {
  background: var(--color-bg-tertiary);
  padding: 4px 8px;
  font-size: 0.85rem;
  font-weight: 500;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.replace-clear {
  background: none;
  border: none;
  font-size: 1.2rem;
  cursor: pointer;
  color: var(--color-text-muted);
  line-height: 1;
}
.replace-preview-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 200px;
  overflow-y: auto;
}
.replace-preview-item {
  display: flex;
  gap: 0.5rem;
  align-items: baseline;
  padding: 3px 8px;
  font-size: 0.8rem;
  font-family: var(--font-mono, monospace);
  border-bottom: 1px solid var(--color-border);
}
.replace-preview-item:last-child {
  border-bottom: none;
}
.replace-file {
  color: var(--color-accent);
  flex-shrink: 0;
  min-width: 20%;
}
.replace-old {
  color: var(--color-danger);
  text-decoration: line-through;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.replace-arrow {
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.replace-new {
  color: var(--color-success);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
