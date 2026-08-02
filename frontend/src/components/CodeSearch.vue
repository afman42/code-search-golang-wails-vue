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
  display: flex;
  min-height: 100vh;
}

.main-content {
  flex: 1;
  overflow-x: auto;
}

.result {
  height: 20px;
  line-height: 20px;
  margin: 1.5rem auto;
  text-align: center;
}

.result.error {
  color: #e74c3c;
}

.error-message {
  max-width: 600px;
  margin: 0.5rem auto;
  padding: 10px;
  background-color: #fadbd8;
  border: 1px solid #e74c3c;
  border-radius: 4px;
  color: #c0392b;
  text-align: center;
  font-size: 0.9em;
}

.searching-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.searching-content {
  text-align: center;
  color: white;
  font-size: 1.2rem;
}

.spinner {
  width: 50px;
  height: 50px;
  border: 5px solid #f3f3f3;
  border-top: 5px solid #3498db;
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
  color: #aaa;
  margin-top: 10px;
}
</style>
