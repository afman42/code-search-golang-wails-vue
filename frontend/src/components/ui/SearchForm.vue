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
      @update="(val: string) => $emit('update:directory', val)"
      :disabled="data.isSearching"
    />

    <!-- Search Query Input -->
    <div class="query-input-wrap">
      <QueryInput
        :query="data.query"
        @focus="onSearchFocus"
        @blur="onSearchBlur"
        @search="handleSearch"
        @update="(val: string) => $emit('update:query', val)"
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

    <!-- Search Options (4 checkboxes) -->
    <SearchOptions
      :caseSensitive="data.caseSensitive"
      :useRegex="data.useRegex"
      :includeBinary="data.includeBinary"
      :fuzzySearch="data.fuzzySearch"
      :respectGitignore="data.respectGitignore"
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
import type {
  PatternKind,
  PatternSelectionUpdate,
  RecentSearch,
  SearchOptionsUpdate,
  SearchState,
  SizeLimitsUpdate,
} from "@/types";
import { ref } from "vue";
import ActionButtons from "./ActionButtons.vue";
import DirectoryPicker from "./DirectoryPicker.vue";
import EditorStatusDisplay from "./EditorStatusDisplay.vue";
import PatternSelector from "./PatternSelector.vue";
import QueryInput from "./QueryInput.vue";
import SearchOptions from "./SearchOptions.vue";
import SearchSuggestions from "./SearchSuggestions.vue";
import SizeLimitOptions from "./SizeLimitOptions.vue";
import { loadRecentSearches } from "@/utils";

interface Props {
  data: SearchState;
  searchCode: () => Promise<void>;
  selectDirectory: () => Promise<void>;
  cancelSearch: () => Promise<void>;
}

const props = defineProps<Props>();

// All field writes flow up as update:xxx events; the parent (CodeSearch.vue)
// owns the reactive SearchState and applies them. The component never mutates
// props.data directly.
const emit = defineEmits<{
  (e: "update:caseSensitive", value: boolean): void;
  (e: "update:useRegex", value: boolean): void;
  (e: "update:includeBinary", value: boolean): void;
  (e: "update:fuzzySearch", value: boolean): void;
  (e: "update:respectGitignore", value: boolean): void;
  (e: "update:minFileSize", value: number): void;
  (e: "update:maxFileSize", value: number): void;
  (e: "update:maxResults", value: number): void;
  (e: "update:contextLines", value: number): void;
  (e: "update:directories", value: string[]): void;
  (e: "update:excludePatterns", value: string[]): void;
  (e: "update:allowedFileTypes", value: string[]): void;
  (e: "update:query", value: string): void;
  (e: "update:extension", value: string): void;
  (e: "update:directory", value: string): void;
  (e: "update:recentSearches", value: RecentSearch[]): void;
}>();

const handleSearchOptionsUpdate = (options: SearchOptionsUpdate) => {
  emit("update:caseSensitive", options.caseSensitive);
  emit("update:useRegex", options.useRegex);
  emit("update:includeBinary", options.includeBinary);
  emit("update:fuzzySearch", options.fuzzySearch);
  emit("update:respectGitignore", options.respectGitignore);
};

const handleSizeLimitsUpdate = (limits: SizeLimitsUpdate) => {
  emit("update:minFileSize", limits.minFileSize);
  emit("update:maxFileSize", limits.maxFileSize);
  emit("update:maxResults", limits.maxResults);
  emit("update:contextLines", limits.contextLines);
};

const handleExtraDirsChange = (event: Event) => {
  const target = event.target as HTMLTextAreaElement;
  const lines = target.value.split('\n').map((s) => s.trim()).filter((s) => s.length > 0);
  emit("update:directories", lines);
};

const handlePatternPatternsUpdate = (patterns: PatternSelectionUpdate) => {
  emit("update:excludePatterns", patterns.exclude);
  emit("update:allowedFileTypes", patterns.allow);
};

const handleSearch = async () => {
  await props.searchCode();
};

const handleRemovePattern = (type: PatternKind, index: number) => {
  const targetArray = type === 'exclude'
    ? props.data.excludePatterns
    : props.data.allowedFileTypes;

  if (index >= 0 && index < targetArray.length) {
    const updated = [...targetArray];
    updated.splice(index, 1);
    if (type === 'exclude') {
      emit("update:excludePatterns", updated);
    } else {
      emit("update:allowedFileTypes", updated);
    }
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

const handleSuggestionSelect = (search: RecentSearch) => {
  emit("update:query", search.query);
  emit("update:extension", search.extension || "");
  // Restore the directory the suggestion was run against, so re-running from a
  // suggestion behaves identically to the original search.
  if (search.directory) {
    emit("update:directory", search.directory);
  }
  showSuggestions.value = false;
  void props.searchCode();
};

const handleSuggestionRemove = () => {
  // SearchSuggestions already removed the entry from localStorage; refresh the
  // sidebar's list so it stays in sync.
  emit("update:recentSearches", loadRecentSearches());
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
