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

    <!-- Pagination controls -->
    <div v-if="totalPages > 1" class="pagination-controls">
      <div class="pagination-info">
        Showing {{ startIndex + 1 }}-{{ Math.min(endIndex, totalResults) }} of {{ totalResults }} results
      </div>
      <div class="pagination-actions">
        <button 
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
      :key="index"
      class="result-item"
    >
      <div class="result-header">
        <div class="file-info">
          <span
            class="file-path"
            @click="openFileLocation(result.filePath)"
            title="Click to show in folder"
          >
            {{ formatFilePath(result.filePath) }}
          </span>
          <span class="line-num">Line {{ result.lineNum }}</span>
          <span class="matched-text" v-if="result.matchedText && result.matchedText !== data.query">
            (Matched: "{{ result.matchedText }}")
          </span>
        </div>
        <button
          class="copy-btn"
          @click="copyToClipboard(result.content)"
          title="Copy line"
        >
          Copy
        </button>
      </div>
      
      <!-- Display context lines before match -->
      <div 
        v-for="(contextLine, ctxIndex) in result.contextBefore"
        :key="'before-' + index + '-' + ctxIndex"
        class="context-line context-before"
        v-html="highlightMatch(contextLine, data.query || '')"
      ></div>
      
      <!-- Display the matched line -->
      <div
        class="result-content"
        v-html="highlightMatch(result.content || '', data.query || '')"
      ></div>
      
      <!-- Display context lines after match -->
      <div 
        v-for="(contextLine, ctxIndex) in result.contextAfter"
        :key="'after-' + index + '-' + ctxIndex"
        class="context-line context-after"
        v-html="highlightMatch(contextLine, data.query || '')"
      ></div>
    </div>

    <!-- Pagination controls at the bottom -->
    <div v-if="totalPages > 1" class="pagination-controls bottom">
      <div class="pagination-info">
        Showing {{ startIndex + 1 }}-{{ Math.min(endIndex, totalResults) }} of {{ totalResults }} results
      </div>
      <div class="pagination-actions">
        <button 
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
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, watch } from 'vue';
import type { PropType } from 'vue';
import { SearchResult, SearchState } from '../../types/search';

export default defineComponent({
  name: 'SearchResults',
  props: {
    data: {
      type: Object as () => SearchState,
      required: true
    },
    formatFilePath: {
      type: Function as PropType<(filePath: string) => string>,
      required: true
    },
    highlightMatch: {
      type: Function as PropType<(text: string, query: string) => string>,
      required: true
    },
    openFileLocation: {
      type: Function as PropType<(filePath: string) => Promise<void>>,
      required: true
    },
    copyToClipboard: {
      type: Function as PropType<(text: string) => Promise<void>>,
      required: true
    }
  },
  setup(props) {
    // Pagination state
    const currentPage = ref(1);
    const itemsPerPage = ref(10); // Default to 10 items per page

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
      return Math.min(startIndex.value + itemsPerPage.value, totalResults.value);
    });

    const paginatedResults = computed(() => {
      if (!props.data.searchResults || !Array.isArray(props.data.searchResults)) {
        return [];
      }
      return props.data.searchResults.slice(startIndex.value, endIndex.value);
    });

    // Method to change page
    const goToPage = (page: number) => {
      if (page >= 1 && page <= totalPages.value) {
        currentPage.value = page;
      }
    };

    // Reset to first page when results change
    // Using a watcher to detect when search results change
    watch(() => props.data.searchResults, () => {
      currentPage.value = 1; // Reset to first page when new results come in
    });

    return {
      currentPage,
      itemsPerPage,
      totalResults,
      totalPages,
      startIndex,
      endIndex,
      paginatedResults,
      goToPage
    };
  }
});
</script>

<style scoped>
.results-container {
  max-width: 800px;
  margin: 20px auto;
  padding: 0 20px;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.results-summary {
  color: #7f8c8d;
  font-size: 0.9em;
}

/* Pagination styles */
.pagination-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 15px 0;
  padding: 10px;
  background-color: #f8f9fa;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.pagination-controls.bottom {
  margin-top: 15px;
  margin-bottom: 20px;
}

.pagination-info {
  color: #7f8c8d;
  font-size: 0.9em;
}

.pagination-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pagination-btn {
  padding: 6px 12px;
  background-color: #3498db;
  color: white;
  border: 1px solid #3498db;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9em;
}

.pagination-btn:hover:not(:disabled) {
  background-color: #2980b9;
}

.pagination-btn:disabled {
  background-color: #bdc3c7;
  border-color: #bdc3c7;
  cursor: not-allowed;
  opacity: 0.6;
}

.page-info {
  color: #7f8c8d;
  font-size: 0.9em;
  margin: 0 5px;
}

.result-item {
  border: 1px solid #ddd;
  border-radius: 4px;
  margin-bottom: 10px;
  padding: 10px;
  background-color: #fafafa;
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
  color: #2980b9;
  cursor: pointer;
  text-decoration: underline;
}

.file-path:hover {
  color: #3498db;
}

.line-num {
  color: #7f8c8d;
  font-size: 0.9em;
  background-color: #ecf0f1;
  padding: 2px 6px;
  border-radius: 3px;
}

.matched-text {
  color: #27ae60;
  font-size: 0.85em;
  font-style: italic;
  margin-left: 10px;
}

.copy-btn {
  background-color: #95a5a6;
  color: white;
  border: none;
  padding: 4px 8px;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8em;
}

.copy-btn:hover {
  background-color: #7f8c8d;
}

.result-content {
  font-family: monospace;
  padding: 8px;
  background-color: #f8f9fa;
  border-left: 3px solid #3498db;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
}

.highlight {
  background-color: #f1c40f;
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

.context-line {
  font-family: monospace;
  padding: 4px 8px;
  background-color: #f0f0f0;
  border-left: 2px solid #bdc3c7;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
  font-size: 0.9em;
  color: #7f8c8d;
}

.context-before {
  border-left-color: #3498db;
}

.context-after {
  border-left-color: #9b59b6;
}
</style>