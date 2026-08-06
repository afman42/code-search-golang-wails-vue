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
      :contextLines="data.contextLines"
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

    <!-- Extra Directories: additional search roots (one path per line) -->
    <div class="extra-dirs-group">
      <label for="extra-dirs" class="extra-dirs-label">Extra Directories</label>
      <textarea
        id="extra-dirs"
        class="extra-dirs-input"
        :value="data.directories.join('\n')"
        @input="handleExtraDirsChange"
        placeholder="One additional directory path per line (optional)"
        :disabled="data.isSearching"
        rows="2"
      ></textarea>
    </div>

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
import type { SearchState } from "@/types";
import { ref } from "vue";
import {
  ActionButtons,
  DirectoryPicker,
  EditorStatusDisplay,
  PatternSelector,
  QueryInput,
  SearchOptions,
  SearchSuggestions,
  SizeLimitOptions,
} from "@/components/ui";
import { loadRecentSearches } from "@/utils";

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
  contextLines: number;
}) => {
  props.data.minFileSize = limits.minFileSize;
  props.data.maxFileSize = limits.maxFileSize;
  props.data.maxResults = limits.maxResults;
  props.data.contextLines = limits.contextLines;
};

const handleExtraDirsChange = (event: Event) => {
  const target = event.target as HTMLTextAreaElement;
  const lines = target.value.split('\n').map((s) => s.trim()).filter((s) => s.length > 0);
  props.data.directories = lines;
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

const handleSuggestionSelect = (search: {
  query: string;
  extension?: string;
  directory?: string;
}) => {
  props.data.query = search.query;
  props.data.extension = search.extension || "";
  // Restore the directory the suggestion was run against, so re-running from a
  // suggestion behaves identically to the original search.
  if (search.directory) {
    props.data.directory = search.directory;
  }
  showSuggestions.value = false;
  void props.searchCode();
};

const handleSuggestionRemove = () => {
  // SearchSuggestions already removed the entry from localStorage; refresh the
  // sidebar's list so it stays in sync.
  props.data.recentSearches = loadRecentSearches();
};

const showSuggestions = ref(false);
</script>

<style scoped>
.search-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.extra-dirs-group {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.extra-dirs-label {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.extra-dirs-input {
  padding: 0.375rem 0.5rem;
  font-size: 0.85rem;
  font-family: var(--font-mono, monospace);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
  resize: vertical;
  min-height: 2.5em;
}

.extra-dirs-input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px var(--color-accent-light);
}

.extra-dirs-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.query-input-wrap {
  position: relative;
}
</style>
