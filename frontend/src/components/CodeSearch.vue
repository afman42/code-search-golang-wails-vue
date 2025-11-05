<template>
  <main>
    <SearchForm 
      :data="data"
      :searchCode="searchCode"
      :selectDirectory="selectDirectory"
      :cancelSearch="cancelSearch"
    />

    <div id="result" class="result" :class="{ 'error': data.error }">{{ data.resultText }}</div>
    
    <!-- Error display -->
    <div v-if="data.error" class="error-message" id="error-display">
      {{ data.error }}
    </div>

    <!-- Progress Bar -->
    <ProgressIndicator 
      :data="data"
      :formatFilePath="formatFilePath"
    />

    <!-- Search Results -->
    <SearchResults 
      :data="data"
      :formatFilePath="formatFilePath"
      :highlightMatch="highlightMatch"
      :openFileLocation="openFileLocation"
      :copyToClipboard="copyToClipboard"
    />
  </main>
</template>

<script lang="ts" setup>
import SearchForm from './ui/SearchForm.vue';
import ProgressIndicator from './ui/ProgressIndicator.vue';
import SearchResults from './ui/SearchResults.vue';
import { useSearch } from '../composables/useSearch';

// Get all the search functionality from the composable
const { 
  data, 
  searchCode, 
  cancelSearch,
  selectDirectory, 
  formatFilePath, 
  highlightMatch, 
  copyToClipboard, 
  openFileLocation 
} = useSearch();
</script>

<style scoped>
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