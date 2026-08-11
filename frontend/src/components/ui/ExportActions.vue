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
  font-size: 0.8em;
  color: var(--color-accent);
  font-weight: 600;
}

.batch-btn {
  padding: 4px 10px;
  font-size: 0.8em;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.batch-btn:hover:not(:disabled) {
  background: var(--color-bg-hover);
  border-color: var(--color-accent);
}

.batch-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
