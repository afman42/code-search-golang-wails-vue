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
    <div
      v-for="(result, index) in data.searchResults &&
      Array.isArray(data.searchResults)
        ? data.searchResults
        : []"
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
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
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