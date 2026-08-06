<template>
  <div class="options-group">
    <div class="control-group checkbox-group">
      <input
        :id="caseSensitiveId"
        v-model="localCaseSensitive"
        type="checkbox"
        :disabled="disabled"
      />
      <label :for="caseSensitiveId">Case Sensitive</label>
    </div>

    <div class="control-group checkbox-group">
      <input
        :id="regexId"
        v-model="localUseRegex"
        type="checkbox"
        :disabled="disabled"
      />
      <label :for="regexId">Regex Search</label>
    </div>

    <div class="control-group checkbox-group">
      <input
        :id="includeBinaryId"
        v-model="localIncludeBinary"
        type="checkbox"
        :disabled="disabled"
      />
      <label :for="includeBinaryId">Include Binary</label>
    </div>

    <div class="control-group checkbox-group">
      <input
        :id="fuzzySearchId"
        v-model="localFuzzySearch"
        type="checkbox"
        :disabled="disabled"
      />
      <label :for="fuzzySearchId">Fuzzy Search</label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

defineOptions({
  name: 'SearchOptions',
});

interface Props {
  caseSensitive?: boolean;
  useRegex?: boolean;
  includeBinary?: boolean;
  fuzzySearch?: boolean;
  disabled?: boolean;
  caseSensitiveId?: string;
  regexId?: string;
  includeBinaryId?: string;
  fuzzySearchId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  caseSensitive: false,
  useRegex: false,
  includeBinary: false,
  fuzzySearch: false,
  disabled: false,
  caseSensitiveId: 'case-sensitive',
  regexId: 'regex-search',
  includeBinaryId: 'include-binary',
  fuzzySearchId: 'fuzzy-search',
});

const emit = defineEmits<{
  update: [options: { 
    caseSensitive: boolean; 
    useRegex: boolean; 
    includeBinary: boolean; 
    fuzzySearch: boolean;
  }];
}>();

const localCaseSensitive = ref(props.caseSensitive);
const localUseRegex = ref(props.useRegex);
const localIncludeBinary = ref(props.includeBinary);
const localFuzzySearch = ref(props.fuzzySearch);

watch(() => props.caseSensitive, (newVal) => {
  localCaseSensitive.value = newVal;
});

watch(() => props.useRegex, (newVal) => {
  localUseRegex.value = newVal;
});

watch(() => props.includeBinary, (newVal) => {
  localIncludeBinary.value = newVal;
});

watch(() => props.fuzzySearch, (newVal) => {
  localFuzzySearch.value = newVal;
});

watch([
  localCaseSensitive, 
  localUseRegex, 
  localIncludeBinary, 
  localFuzzySearch
], ([newCase, newRegex, newBin, newFuzzy]) => {
  emit('update', { 
    caseSensitive: newCase, 
    useRegex: newRegex,
    includeBinary: newBin,
    fuzzySearch: newFuzzy,
  });
});
</script>

<style scoped>
.options-group {
  display: flex;
  gap: 1.5rem;
  margin-bottom: 1rem;
}

.control-group.checkbox-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.control-group.checkbox-group input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.control-group.checkbox-group label {
  margin: 0;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  user-select: none;
}

.control-group.checkbox-group input:disabled + label {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
