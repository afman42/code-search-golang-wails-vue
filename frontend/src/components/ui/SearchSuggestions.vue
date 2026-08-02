<template>
  <div v-if="show && suggestions.length > 0" class="search-suggestions" ref="suggestionsRef">
    <ul class="suggestions-list">
      <li
        v-for="(suggestion, index) in suggestions"
        :key="suggestion.query"
        class="suggestion-item"
        @mousedown.prevent="selectSuggestion(suggestion.query)"
        @mouseenter="hoveredIndex = index"
        @mouseleave="hoveredIndex = -1"
        :class="{ hovered: hoveredIndex === index }"
      >
        <div class="suggestion-content">
          <span class="suggestion-query">{{ suggestion.query }}</span>
          <span class="suggestion-meta">
            <span class="suggestion-frequency" title="Search frequency">
              ×{{ suggestion.frequency }}
            </span>
            <span class="suggestion-timestamp" :title="formatFullDate(suggestion.timestamp)">
              {{ formatRelativeTime(suggestion.timestamp) }}
            </span>
          </span>
        </div>
        <button
          class="suggestion-delete"
          title="Remove suggestion"
          @mousedown.prevent.stop="deleteSuggestion(suggestion.query, index)"
        >
          ×
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import {
  getRecentSuggestions,
  removeRecentSearch,
  type RecentSearch,
} from "../../utils/localStorageUtils";

const props = defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (e: "select", query: string): void;
  (e: "remove", query: string): void;
}>();

const suggestions = ref<RecentSearch[]>([]);
const hoveredIndex = ref(-1);
const suggestionsRef = ref<HTMLElement | null>(null);

const loadSuggestions = () => {
  suggestions.value = getRecentSuggestions(10);
};

const selectSuggestion = (query: string) => {
  emit("select", query);
};

const deleteSuggestion = (query: string, index: number) => {
  removeRecentSearch(query);
  suggestions.value.splice(index, 1);
  hoveredIndex.value = -1;
  emit("remove", query);
};

const formatRelativeTime = (timestamp: number): string => {
  const now = Date.now();
  const diff = now - timestamp;
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  if (hours < 24) return `${hours}h ago`;
  if (days < 7) return `${days}d ago`;
  return formatFullDate(timestamp);
};

const formatFullDate = (timestamp: number): string => {
  const date = new Date(timestamp);
  return date.toLocaleString();
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
  }
);
</script>
<style scoped>
.search-suggestions {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--bg-color, #1e1e1e);
  border: 1px solid var(--border-color, #3c3c3c);
  border-top: none;
  border-radius: 0 0 4px 4px;
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
  border-bottom: 1px solid var(--border-color, #2a2a2a);
  transition: background-color 0.1s;
}

.suggestion-item:last-child {
  border-bottom: none;
}

.suggestion-item.hovered {
  background: var(--hover-bg, #2a2d2e);
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
  color: var(--text-color, #d4d4d4);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.suggestion-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  margin-left: auto;
}

.suggestion-frequency {
  font-size: 0.7rem;
  color: var(--accent-color, #569cd6);
  background: rgba(86, 156, 214, 0.15);
  padding: 1px 5px;
  border-radius: 3px;
}

.suggestion-timestamp {
  font-size: 0.7rem;
  color: var(--muted-text, #6a6a6a);
}

.suggestion-delete {
  background: none;
  border: none;
  color: var(--muted-text, #6a6a6a);
  font-size: 1rem;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
  flex-shrink: 0;
  border-radius: 3px;
  transition: color 0.1s, background 0.1s;
}

.suggestion-delete:hover {
  color: #f48771;
  background: rgba(244, 135, 113, 0.15);
}
</style>
