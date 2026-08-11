<template>
  <div v-if="totalResults > 0" class="batch-actions">
    <label class="select-all-label">
      <input
        type="checkbox"
        :checked="allVisibleSelected"
        @change="$emit('toggleSelectAll')"
      />
      Select All (page)
    </label>
    <span class="selected-count" v-if="selectedCount > 0">{{ selectedCount }} selected</span>
    <button class="batch-btn" :disabled="selectedCount === 0" @click="$emit('copySelected')">Copy Selected</button>
    <button class="batch-btn" :disabled="totalResults === 0" @click="$emit('exportResults', 'csv')">Export CSV</button>
    <button class="batch-btn" :disabled="totalResults === 0" @click="$emit('exportResults', 'json')">Export JSON</button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  totalResults: number;
  selectedCount: number;
  allVisibleSelected: boolean;
}>();

defineEmits<{
  toggleSelectAll: [];
  copySelected: [];
  exportResults: [format: string];
}>();
</script>

<style scoped>
.batch-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: 10px;
  padding: 8px 10px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  flex-wrap: wrap;
}

.select-all-label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 0.85em;
  cursor: pointer;
  user-select: none;
}

.selected-count {
  font-size: 0.85em;
  color: var(--color-text-muted);
}

.batch-btn {
  padding: 6px 12px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.9em;
  transition: all 0.2s;
}

.batch-btn:hover:not(:disabled) {
  background-color: var(--color-bg-secondary);
}

.batch-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
