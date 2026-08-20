<template>
  <main>
    <!-- Searching overlay -->
    <div v-if="data.isSearching" class="searching-overlay">
      <div class="searching-content">
        <div class="spinner"></div>
        <p>Searching...</p>
        <p v-if="data.searchProgress?.totalFiles > 0" class="progress-text">
          {{ data.searchProgress.processedFiles }} / {{ data.searchProgress.totalFiles }} files processed
        </p>
      </div>
    </div>

    <div class="app-layout">
      <button
        class="theme-toggle"
        :title="isDark === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
        @click="toggleTheme"
      >
        {{ isDark === 'dark' ? '☀' : '☾' }}
      </button>
      <SearchHistorySidebar
        :recent-searches="data.recentSearches"
        :current-query="data.query"
        :current-extension="data.extension"
        :current-directory="data.directory"
        @re-search="handleReSearch"
        @remove="removeRecentSearch"
        @clear-all="clearAllRecentSearches"
      />
      <div class="main-content">
        <!-- Symbol Search Panel -->
        <SymbolSearch :directory="data.directory" />

        <SearchForm
          :data="data"
          :searchCode="searchCode"
          :selectDirectory="selectDirectory"
          :cancelSearch="cancelSearch"
        />

        <div id="result" class="result" :class="{ error: data.error }">
          {{ data.resultText }}
        </div>

        <div v-if="data.error" class="error-message" id="error-display">
          {{ data.error }}
        </div>

        <ProgressIndicator :data="data" :formatFilePath="formatFilePath" />

        <SearchResults
          :data="data"
          :formatFilePath="formatFilePath"
          :openFileLocation="openFileLocation"
          :copyToClipboard="copyToClipboard"
          :onSearch="searchCode"
        />
        <div style="margin-top: 40px">&nbsp;</div>
        <LogViewer :data="data" />

        <!-- Top-level file-preview modal (driven by useFilePreview singleton).
             Used by symbol-search navigation and any component that calls
             openFile(). SearchResults keeps its own CodeModal for "View" clicks. -->
        <CodeModal
          :is-visible="previewState.isVisible"
          :file-path="previewState.filePath"
          :file-content="previewState.fileContent"
          :query="previewState.query"
          :files="previewState.files"
          :initial-line="previewState.initialLine"
          @close="closePreview"
        />
      </div>
    </div>
  </main>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted } from "vue";
import {
  CodeModal,
  LogViewer,
  ProgressIndicator,
  SearchForm,
  SearchHistorySidebar,
  SearchResults,
  SymbolSearch,
} from "@/components/ui";
import {
  useFilePreview,
  useKeyboardShortcuts,
  useSearch,
  useTheme,
} from "@/composables";
import type { SymbolInfo } from "@/types";

const { isDark, toggleTheme } = useTheme();

const {
  data,
  searchCode,
  cancelSearch,
  selectDirectory,
  formatFilePath,
  copyToClipboard,
  openFileLocation,
  cleanup,
  focusSearch,
  executeSearch,
  clearSearch,
} = useSearch();

const { previewState, openFile, closePreview } = useFilePreview();

// Symbol-search → code preview: SymbolSearch dispatches a 'symbol-selected'
// CustomEvent. Listen for it and open the preview modal at the symbol's line.
const handleSymbolSelected = (event: Event) => {
  const detail = (event as CustomEvent).detail as SymbolInfo;
  if (!detail?.file) return;
  openFile(detail.file, { initialLine: detail.line });
};

onMounted(() => {
  window.addEventListener("symbol-selected", handleSymbolSelected as EventListener);
});
onUnmounted(() => {
  window.removeEventListener("symbol-selected", handleSymbolSelected as EventListener);
  cleanup();
});

useKeyboardShortcuts({
  onFocusSearch: focusSearch,
  onExecuteSearch: executeSearch,
  onClearSearch: clearSearch,
});

const handleReSearch = (search: {
  query: string;
  extension: string;
  directory?: string;
}) => {
  data.query = search.query;
  data.extension = search.extension;
  // Restore the directory the search was originally run against so history
  // entries stay genuinely re-runnable.
  if (search.directory) {
    data.directory = search.directory;
  }
  searchCode();
};

const removeRecentSearch = (index: number) => {
  data.recentSearches.splice(index, 1);
};

const clearAllRecentSearches = () => {
  data.recentSearches = [];
};

onUnmounted(() => {
  cleanup();
});
</script>

<style scoped>
.app-layout {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  grid-template-areas: "sidebar main";
  min-height: 100vh;
  width: 100%;
}

.search-history-sidebar {
  grid-area: sidebar;
  height: 100vh;
  position: sticky;
  top: 0;
}

.theme-toggle {
  position: fixed;
  top: var(--space-2);
  right: var(--space-3);
  z-index: 900;
  width: 34px;
  height: 34px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-medium);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: var(--font-size-md);
  cursor: pointer;
  opacity: 0.75;
  transition: opacity var(--transition-fast), background var(--transition-fast);
}

.theme-toggle:hover {
  opacity: 1;
  background: var(--color-bg-hover);
}

.main-content {
  grid-area: main;
  min-width: 0;
  overflow-x: auto;
  padding: var(--space-3) var(--space-5) 0;
}

/* Narrow screens: stack sidebar above the main content */
@media (max-width: 768px) {
  .app-layout {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas:
      "sidebar"
      "main";
  }

  .search-history-sidebar {
    height: auto;
    position: static;
    width: 100%;
  }

  .main-content {
    padding: var(--space-3) var(--space-4);
  }
}

.result {
  height: 20px;
  line-height: 20px;
  margin: 1.5rem auto;
  text-align: center;
}

.result.error {
  color: var(--color-danger);
}

.error-message {
  max-width: 600px;
  margin: 0.5rem auto;
  padding: 10px;
  background-color: color-mix(in srgb, var(--color-danger) 15%, var(--color-bg));
  border: 1px solid var(--color-danger);
  border-radius: var(--radius-sm);
  color: var(--color-danger-dark);
  text-align: center;
  font-size: 0.9em;
}

.searching-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--color-bg-overlay);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.searching-content {
  text-align: center;
  color: var(--color-text-inverse);
  font-size: 1.2rem;
}

.spinner {
  width: 50px;
  height: 50px;
  border: 5px solid var(--color-bg-secondary);
  border-top: 5px solid var(--color-accent);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.progress-text {
  font-size: 1rem;
  color: var(--color-text-muted);
  margin-top: 10px;
}
</style>
