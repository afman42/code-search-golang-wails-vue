<template>
  <div class="action-buttons-group">
    <!-- Primary search action -->
    <div v-if="!isSearching" class="control-group">
      <button
        class="btn btn-primary btn-search"
        @click="handleSearch"
        :disabled="disabled"
      >
        Search Code
      </button>
    </div>

    <!-- Cancel action -->
    <button
      v-if="isSearching && !disabled"
      class="btn btn-secondary btn-cancel"
      @click="handleCancel"
    >
      Cancel Search
    </button>
  </div>
</template>

<script setup lang="ts">
import { defineOptions } from 'vue';

defineOptions({
  name: 'ActionButtons',
});

interface Props {
  isSearching?: boolean;
  disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
  isSearching: false,
  disabled: false,
});

const emit = defineEmits<{
  search: [];
  cancel: [];
}>();

const handleSearch = () => {
  emit('search');
};

const handleCancel = () => {
  emit('cancel');
};
</script>

<style scoped>
.action-buttons-group {
  display: flex;
  gap: 0.75rem;
  margin-top: 1rem;
}

.control-group {
  margin-bottom: 0.5rem;
}

.btn {
  padding: 0.5rem 1.25rem;
  border-radius: 0.25rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.btn-primary {
  background-color: var(--color-success);
  color: var(--color-text-inverse);
  border: none;
}

.btn-primary:hover:not(:disabled) {
  background-color: var(--color-success);
}

.btn-secondary {
  background-color: var(--color-danger);
  color: var(--color-text-inverse);
  border: none;
}

.btn-secondary:hover:not(:disabled) {
  background-color: var(--color-danger);
}

.btn-search {
  width: 100%;
}

.btn-cancel {
  width: 100%;
}
</style>
