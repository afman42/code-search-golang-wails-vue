<template>
  <div v-if="show && suggestions.length > 0" class="search-suggestions" ref="suggestionsRef">
    <ul class="suggestions-list">
      <li
        v-for="(suggestion, index) in suggestions"
        :key="`${suggestion.query}-${suggestion.extension}-${index}`"
        class="suggestion-item"
        @mousedown.prevent="selectSuggestion(suggestion)"
        @mouseenter="hoveredIndex = index"
        @mouseleave="hoveredIndex = -1"
        :class="{ hovered: hoveredIndex === index }"
      >
        <div class="suggestion-content">
          <span class="suggestion-query">{{ suggestion.query }}</span>
          <span class="suggestion-meta">
            <span
              v-if="suggestion.extension"
              class="suggestion-extension"
              title="File extension"
            >
              {{ suggestion.extension }}
            </span>
            <span
              v-if="suggestion.directory"
              class="suggestion-directory"
              :title="suggestion.directory"
            >
              {{ shortDirectory(suggestion.directory) }}
            </span>
          </span>
        </div>
        <button
          class="suggestion-delete"
          title="Remove suggestion"
          @mousedown.prevent.stop="deleteSuggestion(suggestion, index)"
        >
          ×
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from "vue";
import {
  loadRecentSearches,
  removeRecentSearch,
} from "@/utils/localStorageUtils";
import type { RecentSearch } from "@/types";

const props = defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (e: "select", search: RecentSearch): void;
  (e: "remove", query: string): void;
  (e: "close"): void;
}>();

const suggestions = ref<RecentSearch[]>([]);
const hoveredIndex = ref(-1);
const suggestionsRef = ref<HTMLElement | null>(null);

const loadSuggestions = () => {
  suggestions.value = loadRecentSearches().slice(0, 10);
};

// Show only the last path segment so long directory paths don't crowd the row.
const shortDirectory = (directory: string): string => {
  const parts = directory.split(/[\\/]/);
  return parts[parts.length - 1] || directory;
};

const selectSuggestion = (search: RecentSearch) => {
  emit("select", search);
};

const deleteSuggestion = (search: RecentSearch, index: number) => {
  removeRecentSearch({
    query: search.query,
    extension: search.extension,
    directory: search.directory,
  });
  suggestions.value.splice(index, 1);
  hoveredIndex.value = -1;
  emit("remove", search.query);
};

// Expose for parent to refresh after a new search is added
defineExpose({
  loadSuggestions,
  refresh: loadSuggestions,
});

// Watch for show changes to refresh suggestions
watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      loadSuggestions();
    }
  },
  { immediate: true }
);

// Close the dropdown when clicking outside it or pressing Escape.
const onDocumentPointerDown = (event: PointerEvent) => {
  const el = suggestionsRef.value;
  if (!el) return;
  const target = event.target as Node | null;
  if (target && el.contains(target)) return;
  emit("close");
};

const onDocumentKeydown = (event: KeyboardEvent) => {
  if (event.key === "Escape") {
    emit("close");
  }
};

onMounted(() => {
  document.addEventListener("pointerdown", onDocumentPointerDown);
  document.addEventListener("keydown", onDocumentKeydown);
});

onUnmounted(() => {
  document.removeEventListener("pointerdown", onDocumentPointerDown);
  document.removeEventListener("keydown", onDocumentKeydown);
});
</script>
<style scoped>
.search-suggestions {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--bg-color, var(--color-surface-dark));
  border: 1px solid var(--border-color, var(--color-border-dark));
  border-top: none;
  border-radius: 0 0 var(--radius-sm) var(--radius-sm);
  max-height: 240px;
  overflow-y: auto;
  z-index: 1000;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
}

.suggestions-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.suggestion-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color, var(--color-border-dark));
  transition: background-color 0.1s;
}

.suggestion-item:last-child {
  border-bottom: none;
}

.suggestion-item.hovered {
  background: var(--hover-bg, var(--color-surface-dark-hover));
}

.suggestion-content {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.suggestion-query {
  font-size: 0.85rem;
  color: var(--text-color, var(--color-text-dark));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.suggestion-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
  margin-left: auto;
}

.suggestion-frequency,
.suggestion-extension {
  font-size: 0.7rem;
  color: var(--accent-color, var(--color-accent));
  background: rgba(var(--color-accent-rgb), 0.15);
  padding: 1px 5px;
  border-radius: 3px;
}

.suggestion-directory {
  font-size: 0.7rem;
  color: var(--muted-text, var(--color-text-dark-muted));
}

.suggestion-delete {
  background: none;
  border: none;
  color: var(--muted-text, var(--color-text-dark-muted));
  font-size: 1rem;
  cursor: pointer;
  padding: 0 var(--space-1);
  line-height: 1;
  flex-shrink: 0;
  border-radius: 3px;
  transition: color 0.1s, background 0.1s;
}

.suggestion-delete:hover {
  color: color-mix(in srgb, var(--color-danger) 15%, var(--color-bg));
  background: color-mix(in srgb, var(--color-danger) 15%, var(--color-surface-dark));
}
</style>
