<template>
  <div class="search-form">
    <!-- Editor Detection Status -->
    <EditorStatusDisplay 
      :editor-detection-status="data.editorDetectionStatus" 
    />

    <!-- Directory Selection -->
    <DirectoryPicker
      :directory="data.directory"
      @select="selectDirectory"
      @update="(val) => data.directory = val"
      :disabled="data.isSearching"
    />

    <!-- Search Query Input -->
    <div class="query-input-wrap">
      <QueryInput
        :query="data.query"
        @focus="onSearchFocus"
        @blur="onSearchBlur"
        @search="handleSearch"
        @update="(val) => data.query = val"
        :disabled="data.isSearching"
      />

      <SearchSuggestions
        v-if="showSuggestions"
        :show="showSuggestions"
        @select="handleSuggestionSelect"
        @remove="handleSuggestionRemove"
        @close="showSuggestions = false"
      />
    </div>

    <!-- Search Options (5 checkboxes) -->
    <SearchOptions
      :caseSensitive="data.caseSensitive"
      :useRegex="data.useRegex"
      :includeBinary="data.includeBinary"
      :searchSubdirs="data.searchSubdirs"
      :fuzzySearch="data.fuzzySearch"
      :disabled="data.isSearching"
      @update="handleSearchOptionsUpdate"
    />

    <!-- File Size & Results Limit Options -->
    <SizeLimitOptions
      :minFileSize="data.minFileSize"
      :maxFileSize="data.maxFileSize"
      :maxResults="data.maxResults"
      :disabled="data.isSearching"
      @update="handleSizeLimitsUpdate"
    />

    <!-- Pattern Selector (exclude/allow) -->
    <PatternSelector
      :excludePatterns="data.excludePatterns"
      :allowedFileTypes="data.allowedFileTypes"
      @update="handlePatternPatternsUpdate"
      @remove-pattern="handleRemovePattern"
    />

    <!-- Action Buttons -->
    <ActionButtons
      :is-searching="data.isSearching"
      :disabled="data.isSearching || !data.directory || !data.query"
      @search="handleSearch"
      @cancel="cancelSearch"
    />
  </div>
</template>

<script setup lang="ts">
import type { SearchState } from '../../types/search';
import { ref } from 'vue';
import EditorStatusDisplay from './EditorStatusDisplay.vue';
import DirectoryPicker from './DirectoryPicker.vue';
import QueryInput from './QueryInput.vue';
import SearchOptions from './SearchOptions.vue';
import SizeLimitOptions from './SizeLimitOptions.vue';
import PatternSelector from './PatternSelector.vue';
import ActionButtons from './ActionButtons.vue';
import SearchSuggestions from './SearchSuggestions.vue';
import { loadRecentSearches } from '../../utils/localStorageUtils';

interface Props {
  data: SearchState;
  searchCode: () => Promise<void>;
  selectDirectory: () => Promise<void>;
  cancelSearch: () => Promise<void>;
}

const props = defineProps<Props>();

// Write child updates directly back to the reactive SearchState (props.data).
// The backend reads from this same reactive object, so writes here flow
// immediately to the Go SearchWithProgress call.
const handleSearchOptionsUpdate = (options: {
  caseSensitive: boolean;
  useRegex: boolean;
  includeBinary: boolean;
  searchSubdirs: boolean;
  fuzzySearch: boolean;
}) => {
  props.data.caseSensitive = options.caseSensitive;
  props.data.useRegex = options.useRegex;
  props.data.includeBinary = options.includeBinary;
  props.data.searchSubdirs = options.searchSubdirs;
  props.data.fuzzySearch = options.fuzzySearch;
};

const handleSizeLimitsUpdate = (limits: {
  minFileSize: number;
  maxFileSize: number;
  maxResults: number;
}) => {
  props.data.minFileSize = limits.minFileSize;
  props.data.maxFileSize = limits.maxFileSize;
  props.data.maxResults = limits.maxResults;
};

const handlePatternPatternsUpdate = (patterns: {
  exclude: string[];
  allow: string[];
}) => {
  props.data.excludePatterns = patterns.exclude;
  props.data.allowedFileTypes = patterns.allow;
};

const handleSearch = async () => {
  await props.searchCode();
};

const handleRemovePattern = (type: 'exclude' | 'allow', index: number) => {
  const targetArray = type === 'exclude' 
    ? props.data.excludePatterns 
    : props.data.allowedFileTypes;
  
  if (index >= 0 && index < targetArray.length) {
    targetArray.splice(index, 1);
  }
};

const onSearchFocus = () => {
  showSuggestions.value = true;
};
const onSearchBlur = () => {
  // Keep the dropdown open long enough for a suggestion's mousedown handler to
  // run (items use @mousedown.prevent so selecting one never triggers blur);
  // otherwise close it shortly after the input loses focus.
  setTimeout(() => {
    showSuggestions.value = false;
  }, 150);
};

const handleSuggestionSelect = (query: string) => {
  props.data.query = query;
  showSuggestions.value = false;
  void props.searchCode();
};

const handleSuggestionRemove = () => {
  // SearchSuggestions already removed the entry from localStorage; refresh the
  // sidebar's list so it stays in sync.
  props.data.recentSearches = loadRecentSearches() as Array<{
    query: string;
    extension: string;
  }>;
};

const showSuggestions = ref(false);
</script>

<style scoped>
.search-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.query-input-wrap {
  position: relative;
}
</style>
