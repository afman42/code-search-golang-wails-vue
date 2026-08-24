<template>
  <div class="options-group">
    <div class="control-group">
      <label :for="minFileSizeId">Min File Size (bytes):</label>
      <input
        :id="minFileSizeId"
        v-model.number="localMinFileSize"
        class="input"
        type="number"
        placeholder="0"
        :disabled="disabled"
      />
    </div>

    <div class="control-group">
      <label :for="maxFileSizeId">Max File Size (bytes):</label>
      <input
        :id="maxFileSizeId"
        v-model.number="localMaxFileSize"
        class="input"
        type="number"
        placeholder="10485760 (10MB)"
        :disabled="disabled"
      />
    </div>

    <div class="control-group">
      <label :for="maxResultsId">Max Results:</label>
      <input
        :id="maxResultsId"
        v-model.number="localMaxResults"
        class="input"
        type="number"
        placeholder="1000"
        :disabled="disabled"
      />
    </div>

    <div class="control-group">
      <label :for="contextLinesId">Context Lines:</label>
      <input
        :id="contextLinesId"
        v-model.number="localContextLines"
        class="input"
        type="number"
        min="1"
        max="10"
        placeholder="3"
        :title="'Lines of context before/after each match (1–10)'"
        :disabled="disabled"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import type { SizeLimitsUpdate } from '@/types';
import {
  DEFAULT_MAX_FILE_SIZE,
  DEFAULT_MAX_RESULTS,
  DEFAULT_MIN_FILE_SIZE,
} from '@/constants/appConstants';

defineOptions({
  name: 'SizeLimitOptions',
});

interface Props {
  minFileSize?: number;
  maxFileSize?: number;
  maxResults?: number;
  contextLines?: number;
  disabled?: boolean;
  minFileSizeId?: string;
  maxFileSizeId?: string;
  maxResultsId?: string;
  contextLinesId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  minFileSize: DEFAULT_MIN_FILE_SIZE,
  maxFileSize: DEFAULT_MAX_FILE_SIZE,
  maxResults: DEFAULT_MAX_RESULTS,
  contextLines: 3,
  disabled: false,
  minFileSizeId: 'min-filesize',
  maxFileSizeId: 'max-filesize',
  maxResultsId: 'max-results',
  contextLinesId: 'context-lines',
});

const emit = defineEmits<{
  update: [options: SizeLimitsUpdate];
}>();

const localMinFileSize = ref(props.minFileSize || DEFAULT_MIN_FILE_SIZE);
const localMaxFileSize = ref(props.maxFileSize || DEFAULT_MAX_FILE_SIZE);
const localMaxResults = ref(props.maxResults || DEFAULT_MAX_RESULTS);
const localContextLines = ref(props.contextLines || 3);

watch(() => props.minFileSize, (newVal) => {
  localMinFileSize.value = newVal || DEFAULT_MIN_FILE_SIZE;
});

watch(() => props.maxFileSize, (newVal) => {
  localMaxFileSize.value = newVal || DEFAULT_MAX_FILE_SIZE;
});

watch(() => props.maxResults, (newVal) => {
  localMaxResults.value = newVal || DEFAULT_MAX_RESULTS;
});

watch(() => props.contextLines, (newVal) => {
  localContextLines.value = newVal && newVal > 0 ? newVal : 3;
});

watch([localMinFileSize, localMaxFileSize, localMaxResults, localContextLines], ([newMin, newMax, newLimit, newCtx]) => {
  emit('update', { 
    minFileSize: newMin, 
    maxFileSize: newMax, 
    maxResults: newLimit, 
    contextLines: newCtx, 
  });
});
</script>

<style scoped>
.options-group {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.control-group {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.control-group label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.input {
  padding: 0.5rem;
  border: 1px solid var(--color-border-medium);
  border-radius: 0.25rem;
  font-size: 0.875rem;
  min-width: 120px;
}

.input:focus {
  outline: none;
  border-color: var(--color-accent-light);
  box-shadow: 0 0 0 0.2rem rgba(0, 123, 255, 0.25);
}

.input:disabled {
  background-color: var(--color-bg-tertiary);
  cursor: not-allowed;
}
</style>
