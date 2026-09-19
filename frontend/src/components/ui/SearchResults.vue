<template>
  <div v-if="data.searchResults && Array.isArray(data.searchResults) && data.searchResults.length > 0" class="results-container">
    <div class="results-header">
      <h2 class="results-title">Search Results</h2>
      <div class="results-summary" role="status" aria-live="polite">
        Found {{ resultsCount }} matches <span v-if="data.truncatedResults">(truncated)</span>
      </div>
    </div>

    <!-- Find & Replace (hidden under regex mode — backend rejects regex replace) -->
    <div v-if="!data.useRegex" class="replace-row">
      <label for="replace-input" class="sr-only">Replace text</label>
      <input
        id="replace-input"
        v-model="replacement"
        class="replace-input"
        placeholder="Replace matches with…"
        aria-label="Replace matches with"
                :disabled="isPreviewing || isApplying"
      />
      <button
        class="replace-btn"
        aria-label="Preview replace"
        @click="previewReplace"
                :disabled="isPreviewing || isApplying || !replacement"
      >
        <span v-if="isPreviewing" class="replace-spinner" aria-hidden="true"></span>
        {{ isPreviewing ? 'Previewing…' : 'Preview Replace' }}
      </button>
      <button
        class="replace-btn apply"
        :aria-label="preview && preview.filesChanged > 0 ? `Apply replace to ${preview.filesChanged} files` : 'Apply replace'"
        @click="applyReplace"
                :disabled="isApplying || isPreviewing || !preview || preview.filesChanged === 0"
      >
        <span v-if="isApplying" class="replace-spinner" aria-hidden="true"></span>
        {{ isApplying ? 'Applying…' : `Apply ${preview && preview.filesChanged > 0 ? preview.linesChanged : ''}` }}
      </button>
    </div>
    <ReplaceProgress :progress="progress" :format-file-path="formatFilePath" />
    <ReplacePreview v-if="!data.useRegex" :preview="preview" :format-file-path="formatFilePath" @clear="clearPreview" />

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
    <ResultRow
      v-for="(result, index) in paginatedResults"
      :key="result.filePath + result.lineNum + index"
      :result="result"
      :index="startIndex + index"
      :is-selected="isSelected(startIndex + index)"
      :format-file-path="formatFilePath"
      :available-editors="data.availableEditors"
      :query="data.query"
      :case-sensitive="data.caseSensitive"
      @toggle="handleToggleSelected(startIndex + index)"
      @open-location="openFileLocation"
      @open-preview="openFilePreview"
      @copy="copyToClipboard"
      @editor-select="(event, filePath) => handleEditorSelect(event, filePath)"
    />

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
  <div v-else-if="shouldShowEmptyState" class="empty-state-container" role="status" aria-live="polite">
    <div class="empty-state-icon" aria-hidden="true">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
        <line x1="8" y1="11" x2="14" y2="11" />
      </svg>
    </div>
    <h3 class="empty-state-title">No matches found</h3>
    <p class="empty-state-text">
      No results for <strong>"{{ data.query }}"</strong><span v-if="data.extension"> in <code>{{ data.extension }}</code> files</span>.
      Try adjusting your query, extension filter, or search options.
    </p>
    <p v-if="data.searchProgress?.failedFiles > 0" class="empty-state-hint">
      {{ data.searchProgress.failedFiles }} file(s) could not be read and were skipped.
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import type { SearchState, SearchResult } from "@/types";
import CodeModal from "./CodeModal.vue";
import ExportActions from "./ExportActions.vue";
import PaginationControls from "./PaginationControls.vue";
import ReplacePreview from "./ReplacePreview.vue";
import ReplaceProgress from "./ReplaceProgress.vue";
import { ReadFile, ExportSearchResults } from "@wails/go/main/App";
import { toastManager, useSelectionManager, useReplace } from "@/composables";
// From the file directly: the '@/composables' barrel doesn't re-export it.
import type { ExportFormat } from "@/composables/useSelectionManager";
import { handleEditorSelect, toErrorMessage } from "@/utils";
import ResultRow from "./ResultRow.vue";

interface Props {
  data: SearchState;
  formatFilePath: (filePath: string) => string;
  openFileLocation: (filePath: string) => Promise<void>;
  copyToClipboard: (text: string) => Promise<boolean>;
  onSearch?: () => Promise<void>;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: "update:resultText", value: string): void;
  (e: "update:error", value: string | null): void;
}>();

// Find & Replace: literal replacement with dry-run preview + explicit apply.
// Re-runs the search after apply so results reflect the changed files.
const { replacement, preview, progress, isPreviewing, isApplying, previewReplace, applyReplace, clearPreview } =
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

const shouldShowEmptyState = computed(() => {
  if (props.data.isSearching) return false
  if (!Array.isArray(props.data.searchResults)) return false
  if (props.data.searchResults.length > 0) return false
  // Only show after a search has been attempted (query + resultText/error present)
  const hasAttempted = (props.data.resultText && props.data.resultText !== '') || !!props.data.error
  if (!hasAttempted) return false
  if (!props.data.query) return false
  return true
});

// Selection manager composable. Passing the result set lets it auto-clear the
// selection when a new search replaces the results (stale indices would
// otherwise point at a different set).
const selectionManager = useSelectionManager({
  totalResults: resultsCount,
  startIndex,
  endIndex,
  results: () => props.data.searchResults,
});
const { selectedCount, allVisibleSelected, isSelected, toggleSelected, toggleSelectAll } = selectionManager;

// Go to a specific page
const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value && page !== currentPage.value) {
    currentPage.value = page;
  }
};

// Reset pagination and clear selection when results change
// New results make an outstanding dry-run preview stale: the diff list and
// the Apply button refer to the previous match set. Clear it so the user
// previews against what's actually on screen.
watch(
  () => props.data.searchResults,
  () => {
    currentPage.value = 1;
    selectionManager.clearSelection();
    clearPreview();
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
      emit("update:resultText", `Cannot read file in dev mode. Run 'wails dev' or 'wails build'. Error: ${errorMsg}`);
      toastManager.error(`Wails not running: Cannot read files without backend. Use 'wails dev' instead of 'npm run dev'.`);
    } else {
      emit("update:resultText", `Failed to read file: ${errorMsg}`);
      toastManager.error(`File read error: ${errorMsg}`);
    }
    emit("update:error", `File read error: ${errorMsg}`);
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
  try {
    // copyToClipboardWithToast already shows its own error toast on failure
    // and returns false — gate the success toast on it instead of trusting
    // selection state, so a failed copy is never celebrated.
    const copied = await selectionManager.copySelectedResults(results, props.copyToClipboard);
    if (copied) {
      toastManager.success(`Copied ${selectedCount.value} results`);
    }
  } catch (error: unknown) {
    console.error("[SearchResults] Failed to copy selected results:", error);
    toastManager.error(`Failed to copy selected results: ${toErrorMessage(error)}`);
  }
};

const handleExportResults = async (format: ExportFormat = "csv") => {
  const results = props.data.searchResults;
  if (!Array.isArray(results) || results.length === 0) return;
  // exportSelectedResults writes the selected subset when anything is
  // selected, so the toast reports that count instead of the full result set.
  const exportedCount = selectedCount.value > 0 ? selectedCount.value : results.length;
  try {
    const savedPath = await selectionManager.exportSelectedResults(
      results,
      (toExport, fmt) => ExportSearchResults(toExport as SearchResult[], fmt),
      format
    );
    if (savedPath) {
      toastManager.success(`Exported ${exportedCount} results to ${savedPath}`);
    }
  } catch (error: unknown) {
    console.error("[SearchResults] Failed to export results:", error);
    toastManager.error(`Failed to export results: ${toErrorMessage(error)}`);
  }
};

const handleCopyFromModal = () => {
  emit("update:resultText", "File content copied to clipboard");
};
</script>

<style scoped>
.results-container {
  max-width: 50rem;
  margin: var(--space-5) auto;
  padding: 0 var(--space-5);
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-3);
  gap: var(--space-3);
  flex-wrap: wrap;
}

.results-title {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--color-text-primary);
  line-height: 1.4;
}

.results-summary {
  color: var(--color-text-muted);
  font-size: var(--font-size-sm);
}

/* Empty state – shown when search returned 0 matches */
.empty-state-container {
  max-width: 32rem;
  margin: var(--space-6) auto;
  padding: var(--space-6) var(--space-5);
  text-align: center;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.empty-state-icon {
  color: var(--color-text-muted);
  margin-bottom: var(--space-3);
  display: flex;
  justify-content: center;
}

.empty-state-title {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--color-text-primary);
}

.empty-state-text {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.empty-state-text code {
  font-family: var(--font-mono);
  background: var(--color-bg-tertiary);
  padding: 1px var(--space-1);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
}

.empty-state-hint {
  margin: var(--space-3) 0 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.result-item {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
  padding: var(--space-3);
  background-color: var(--color-bg-secondary);
  transition: box-shadow var(--transition-fast);
}

.result-item:hover {
  box-shadow: var(--shadow-sm);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-1);
  flex-wrap: wrap;
  gap: var(--space-1);
}

.file-info {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex: 1;
}

.result-checkbox {
  margin-right: var(--space-2);
  cursor: pointer;
}

.file-path {
  font-weight: 600;
  color: var(--color-accent);
  cursor: pointer;
  text-decoration: underline;
}

.file-path:hover {
  color: var(--color-accent-dark);
}

.line-num {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
  background-color: var(--color-bg-tertiary);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
}

.matched-text {
  color: var(--color-success);
  font-size: var(--font-size-xs);
  font-style: italic;
  margin-left: var(--space-3);
}

.copy-btn {
  background-color: var(--color-text-muted);
  color: var(--color-text-inverse);
  border: none;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-size-xs);
}

.copy-btn:hover {
  background-color: var(--color-text-secondary);
}

/* Replace row */
.replace-row {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  margin-bottom: var(--space-3);
  padding: var(--space-2) var(--space-3);
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
.replace-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}
.replace-spinner {
  width: 0.8em;
  height: 0.8em;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: replace-spin 0.7s linear infinite;
}
@keyframes replace-spin {
  to { transform: rotate(360deg); }
}

</style>
