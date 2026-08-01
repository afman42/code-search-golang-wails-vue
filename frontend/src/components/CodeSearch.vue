<template>
  <main>
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
import { useSearch } from "../composables/useSearch";
import { onUnmounted } from "vue";

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
} = useSearch();

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
</style>
