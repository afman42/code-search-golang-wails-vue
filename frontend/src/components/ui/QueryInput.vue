<template>
  <div class="control-group">
    <label :for="id">{{ label }}</label>
    <input
      :id="id"
      v-model="query"
      ref="inputRef"
      class="input"
      type="text"
      :placeholder="placeholder"
      :disabled="disabled"
      @keyup.enter="emit('search')"
      @focus="onFocus"
      @blur="onBlur"
    />
  </div>
</template>

<script setup lang="ts">
import { useTemplateRef } from 'vue';

defineOptions({
  name: 'QueryInput',
});

interface Props {
  id?: string;
  label?: string;
  placeholder?: string;
  disabled?: boolean;
}

withDefaults(defineProps<Props>(), {
  id: 'query',
  label: 'Search Query:',
  placeholder: 'Enter search term',
  disabled: false,
});

const query = defineModel<string>('query', { required: true });

const emit = defineEmits<{
  search: [];
  focus: [];
  blur: [];
}>();

const inputRef = useTemplateRef<HTMLInputElement>('inputRef');

const onFocus = () => emit('focus');
const onBlur = () => emit('blur');

const focusInput = () => {
  inputRef.value?.focus();
};

defineExpose({ focusInput });
</script>

<style scoped>
.control-group {
  margin-bottom: 1rem;
}

.control-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
  color: var(--color-text-secondary);
}

.input {
  width: 100%;
  padding: 0.375rem 0.75rem;
  border: 1px solid var(--color-border-medium);
  border-radius: 0.25rem;
  font-size: 0.875rem;
  box-sizing: border-box;
}

.input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(77, 171, 247, 0.1);
}

.input:disabled {
  background-color: var(--color-bg-tertiary);
  cursor: not-allowed;
}
</style>
