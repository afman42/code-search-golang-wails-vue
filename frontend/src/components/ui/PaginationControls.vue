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
  margin: 15px 0;
  padding: 10px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.pagination-controls.bottom {
  margin-top: 15px;
  margin-bottom: var(--space-5);
}

.pagination-info {
  color: var(--color-text-muted);
  font-size: 0.9em;
}

.pagination-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pagination-btn {
  padding: 6px var(--space-3);
  background-color: var(--color-accent);
  color: var(--color-text-inverse);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.9em;
}

.pagination-btn:hover:not(:disabled) {
  background-color: var(--color-accent);
}

.pagination-btn:disabled {
  background-color: var(--color-border-medium);
  border-color: var(--color-border-medium);
  cursor: not-allowed;
  opacity: 0.6;
}

.page-info {
  color: var(--color-text-muted);
  font-size: 0.9em;
  margin: 0 5px;
}
</style>
