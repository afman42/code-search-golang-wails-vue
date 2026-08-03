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
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

defineOptions({
  name: 'SizeLimitOptions',
});

interface Props {
  minFileSize?: number;
  maxFileSize?: number;
  maxResults?: number;
  disabled?: boolean;
  minFileSizeId?: string;
  maxFileSizeId?: string;
  maxResultsId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  minFileSize: 0,
  maxFileSize: 10485760, // 10MB default
  maxResults: 1000,
  disabled: false,
  minFileSizeId: 'min-filesize',
  maxFileSizeId: 'max-filesize',
  maxResultsId: 'max-results',
});

const emit = defineEmits<{
  update: [options: { 
    minFileSize: number; 
    maxFileSize: number; 
    maxResults: number; 
  }];
}>();

const localMinFileSize = ref(props.minFileSize || 0);
const localMaxFileSize = ref(props.maxFileSize || 10485760);
const localMaxResults = ref(props.maxResults || 1000);

watch(() => props.minFileSize, (newVal) => {
  localMinFileSize.value = newVal || 0;
});

watch(() => props.maxFileSize, (newVal) => {
  localMaxFileSize.value = newVal || 10485760;
});

watch(() => props.maxResults, (newVal) => {
  localMaxResults.value = newVal || 1000;
});

watch([localMinFileSize, localMaxFileSize, localMaxResults], ([newMin, newMax, newLimit]) => {
  emit('update', { 
    minFileSize: newMin, 
    maxFileSize: newMax, 
    maxResults: newLimit, 
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
