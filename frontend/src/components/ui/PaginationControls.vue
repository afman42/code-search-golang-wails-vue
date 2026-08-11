<template>
  <div v-if="totalPages > 1" class="pagination-controls">
    <div class="pagination-info">
      Showing {{ startIndex + 1 }}-{{ Math.min(endIndex, totalResults) }} of
      {{ totalResults }} results
    </div>
    <div class="pagination-actions">
      <button
        class="pagination-btn"
        :disabled="currentPage === 1"
        @click="$emit('goToPage', currentPage - 1)"
      >
        Previous
      </button>
      <span class="page-info">{{ currentPage }} of {{ totalPages }}</span>
      <button
        class="pagination-btn"
        :disabled="currentPage === totalPages"
        @click="$emit('goToPage', currentPage + 1)"
      >
        Next
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  currentPage: number;
  itemsPerPage: number;
  totalResults: number;
  startIndex: number;
  endIndex: number;
}>();

// Total number of pages, derived from the result count and page size. The
// template's visibility guard and page counter both depend on this.
const totalPages = computed(() =>
  Math.max(1, Math.ceil(props.totalResults / props.itemsPerPage))
);

defineEmits<{
  goToPage: [page: number];
}>();
</script>

<style scoped>
.pagination-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  padding: 8px 10px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.pagination-info {
  font-size: 0.9em;
  color: var(--color-text-muted);
}

.pagination-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.page-info {
  font-size: 0.9em;
  min-width: 100px;
  text-align: center;
}

.pagination-btn {
  padding: 6px 12px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.9em;
  transition: all 0.2s;
}

.pagination-btn:hover:not(:disabled) {
  background-color: var(--color-bg-secondary);
}

.pagination-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
