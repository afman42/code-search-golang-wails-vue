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
      <SearchHistorySidebar
        :recent-searches="data.recentSearches"
        :current-query="data.query"
        :current-extension="data.extension"
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
          :highlightMatch="highlightMatch"
          :openFileLocation="openFileLocation"
          :copyToClipboard="copyToClipboard"
        />
        <div style="margin-top: 40px">&nbsp;</div>
        <LogViewer :data="data" />
      </div>
    </div>
  </main>
</template>

<script lang="ts" setup>
import SearchForm from "./ui/SearchForm.vue";
import ProgressIndicator from "./ui/ProgressIndicator.vue";
import SearchResults from "./ui/SearchResults.vue";
import LogViewer from "./ui/LogViewer.vue";
import SearchHistorySidebar from "./ui/SearchHistorySidebar.vue";
import SymbolSearch from "./ui/SymbolSearch.vue";
import { useSearch } from "../composables/useSearch";
import { onUnmounted } from "vue";
import { useKeyboardShortcuts } from "../composables/useKeyboardShortcuts";

const {
  data,
  searchCode,
  cancelSearch,
  selectDirectory,
  formatFilePath,
  highlightMatch,
  copyToClipboard,
  openFileLocation,
  cleanup,
  focusSearch,
  executeSearch,
  clearSearch,
} = useSearch();
useKeyboardShortcuts({
  onFocusSearch: focusSearch,
  onExecuteSearch: executeSearch,
  onClearSearch: clearSearch,
});

const handleReSearch = (search: { query: string; extension: string }) => {
  data.query = search.query;
  data.extension = search.extension;
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
