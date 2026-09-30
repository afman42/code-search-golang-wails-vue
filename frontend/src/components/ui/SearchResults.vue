<template>
  <div v-if="data.searchResults && Array.isArray(data.searchResults) && data.searchResults.length > 0" class="results-container">
    <div class="results-header">
      <h2 class="results-title">Search Results</h2>
      <div class="results-summary" role="status" aria-live="polite">
        Found {{ resultsCount }} matches <span v-if="data.truncatedResults">(truncated)</span>
      </div>
    </div>

    <!-- Find & Replace (hidden under regex mode — backend rejects regex replace) -->
    <ReplaceBar
      v-if="!data.useRegex"
      v-model="replacement"
      :is-previewing="isPreviewing"
      :is-applying="isApplying"
      :can-apply="canApply"
      :apply-label="applyLabel"
      @preview="previewReplace"
      @apply="applyReplace"
    />
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
      :key="result.filePath + result.lineNum"
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

    <!-- Code Modal for viewing full files (lazy chunk, fetched on first open) -->
    <Suspense v-if="showCodeModal">
      <CodeModal
        :is-visible="showCodeModal"
        :file-path="selectedFilePath"
        :file-content="selectedFileContent"
        :query="data.query"
        :files="resultFilePaths"
        @close="closeFilePreview"
        @copy="handleCopyFromModal"
      />
      <template #fallback><div aria-hidden="true" /></template>
    </Suspense>
  </div>
  <EmptyState
    v-else-if="shouldShowEmptyState"
    :query="data.query"
    :extension="data.extension"
    :failed-files="data.searchProgress?.failedFiles ?? 0"
  />
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref, computed, shallowRef, watch } from "vue";
import type { SearchState, SearchResult } from "@/types";
// On-demand file preview: separate chunk, only fetched on first "View" click.
const CodeModal = defineAsyncComponent(() => import("@/components/ui/CodeModal.vue"));
import {
  EmptyState,
  ExportActions,
  PaginationControls,
  ReplaceBar,
  ReplacePreview,
  ReplaceProgress,
  ResultRow,
} from "@/components/ui";
import { ReadFile, ExportSearchResults } from "@wails/go/main/App";
import { useSelectionManager, useReplace } from "@/composables";
import { toastManager } from "@/composables/useToast";
// From the file directly: the '@/composables' barrel doesn't re-export it.
import type { ExportFormat } from "@/composables/useSelectionManager";
import { handleEditorSelect, toErrorMessage } from "@/utils";

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

// Apply state for ReplaceBar: applying is only meaningful after a dry-run
// preview reported changed files; the label shows the lines to write.
const canApply = computed(() => !!preview.value && preview.value.filesChanged > 0);
const applyLabel = computed(() =>
  preview.value && preview.value.filesChanged > 0 ? `Apply ${preview.value.linesChanged}` : "Apply "
);

// Pagination state
const currentPage = ref(1);
const itemsPerPage = ref(10);

// Modal state
const showCodeModal = shallowRef(false);
const selectedFilePath = shallowRef("");
const selectedFileContent = shallowRef("");

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
  /* Collapsed LogViewer is position:fixed bottom:0 height:40px (LogViewer.vue).
     Without this the last result row slides under it and its View/Copy
     buttons are unclickable until the user scrolls. */
  padding-bottom: 56px;
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

</style>
