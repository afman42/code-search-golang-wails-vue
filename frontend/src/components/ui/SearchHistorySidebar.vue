<template>
  <aside class="search-history-sidebar" :class="{ collapsed: !isVisible }">
    <div class="sidebar-header">
      <h3>Recent Searches</h3>
      <button
        class="toggle-btn"
        @click="isVisible = !isVisible"
        :title="isVisible ? 'Collapse' : 'Expand'"
      >
        {{ isVisible ? '◀' : '▶' }}
      </button>
    </div>

    <div v-if="isVisible" class="sidebar-content">
      <p v-if="recentSearchesList.length === 0" class="empty-state">
        No recent searches yet.
      </p>

      <ul v-else class="history-list">
        <li
          v-for="(search, index) in recentSearchesList"
          :key="`${search.query}-${search.extension}-${index}`"
          class="history-item"
          :class="{ active: isActiveSearch(search) }"
          @click="$emit('re-search', search)"
          :title="`Query: ${search.query}${search.extension ? ' · Ext: ' + search.extension : ''}`"
        >
          <div class="history-query">{{ search.query }}</div>
          <div class="history-meta" v-if="search.extension">
            <span class="history-ext">{{ search.extension }}</span>
          </div>
          <button
            class="remove-history"
            @click.stop="$emit('remove', index)"
            title="Remove this search"
          >
            ×
          </button>
        </li>
      </ul>

      <button
        v-if="recentSearchesList.length > 0"
        class="clear-all-btn"
        @click="$emit('clear-all')"
      >
        Clear All
      </button>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";

export interface RecentSearch {
  query: string;
  extension: string;
}

const props = defineProps<{
  recentSearches?: RecentSearch[];
  currentQuery?: string;
  currentExtension?: string;
}>();

// Provide a default empty array if undefined
const recentSearchesList = computed(() => props.recentSearches || []);

defineEmits<{
  (e: "re-search", search: RecentSearch): void;
  (e: "remove", index: number): void;
  (e: "clear-all"): void;
}>();

const isVisible = ref(true);

const isActiveSearch = (search: RecentSearch): boolean => {
  return (
    search.query === props.currentQuery &&
    search.extension === (props.currentExtension || "")
  );
};
</script>

<style scoped>
.search-history-sidebar {
  width: 240px;
  background-color: #2d2d2d;
  border-right: 1px solid #555;
  display: flex;
  flex-direction: column;
  transition: width 0.2s ease;
  overflow: hidden;
  flex-shrink: 0;
}

.search-history-sidebar.collapsed {
  width: 40px;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #555;
  background-color: #2d2d2d;
}

.sidebar-header h3 {
  margin: 0;
  font-size: 14px;
  color: #fff;
  white-space: nowrap;
}

.collapsed .sidebar-header h3 {
  display: none;
}

.toggle-btn {
  background: none;
  border: 1px solid #555;
  color: #ccc;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.2s;
}

.toggle-btn:hover {
  background-color: #555;
  color: #fff;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
}

.empty-state {
  color: #888;
  font-size: 0.85em;
  text-align: center;
  padding: 20px 8px;
  margin: 0;
}

.history-list {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
}

.history-item {
  position: relative;
  padding: 8px 28px 8px 12px;
  margin-bottom: 4px;
  background-color: #333;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
  border: 1px solid transparent;
}

.history-item:hover {
  background-color: #3a3a3a;
  border-color: #555;
}

.history-item.active {
  background-color: #2c4a2c;
  border-color: #4caf50;
}

.history-query {
  color: #fff;
  font-size: 0.85em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: "Monaco", "Menlo", monospace;
}

.history-meta {
  margin-top: 4px;
}

.history-ext {
  color: #4caf50;
  font-size: 0.75em;
  background-color: #1a1a1a;
  padding: 1px 6px;
  border-radius: 3px;
}

.remove-history {
  position: absolute;
  right: 6px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: #888;
  font-size: 16px;
  cursor: pointer;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
}

.history-item:hover .remove-history {
  opacity: 1;
}

.remove-history:hover {
  background-color: #e74c3c;
  color: #fff;
}

.clear-all-btn {
  margin-top: 8px;
  padding: 6px 12px;
  background-color: #e74c3c;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8em;
  transition: background-color 0.2s;
}

.clear-all-btn:hover {
  background-color: #c0392b;
}

.sidebar-content::-webkit-scrollbar {
  width: 6px;
}

.sidebar-content::-webkit-scrollbar-track {
  background: #222;
}

.sidebar-content::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 3px;
}
</style>
