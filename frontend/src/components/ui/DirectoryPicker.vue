<template>
  <div class="control-group">
    <label :for="id">{{ label }}</label>
    <div class="directory-input">
      <input
        :id="id"
        v-model="localDirectory"
        class="input directory"
        type="text"
        :placeholder="placeholder"
        :disabled="disabled"
      />
      <button
        class="btn select-dir"
        :disabled="disabled"
        @click="handleBrowse"
      >
        {{ buttonText }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

defineOptions({
  name: 'DirectoryPicker',
});


interface Props {
  id?: string;
  label?: string;
  placeholder?: string;
  buttonText?: string;
  directory: string;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  id: 'directory',
  label: 'Directory:',
  placeholder: 'Enter directory to search',
  buttonText: 'Browse',
  disabled: false,
});

const emit = defineEmits<{
  select: [directory: string];
  update: [directory: string];
}>();

const localDirectory = ref(props.directory);

watch(() => props.directory, (newVal) => {
  localDirectory.value = newVal;
});

watch(localDirectory, (newVal) => {
  emit('update', newVal);
});

const handleBrowse = () => {
  emit('select', localDirectory.value);
};
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

.directory-input {
  display: flex;
  gap: 0.5rem;
}

.input.directory {
  flex: 1;
  padding: 0.375rem 0.75rem;
  border: 1px solid var(--color-border-medium);
  border-radius: 0.25rem;
  font-size: 0.875rem;
  min-width: 0;
}

.input.directory:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(77, 171, 247, 0.1);
}

.btn.select-dir {
  padding: 0.375rem 1rem;
  background-color: var(--color-text-secondary);
  color: var(--color-text-inverse);
  border: none;
  border-radius: 0.25rem;
  cursor: pointer;
  white-space: nowrap;
}

.btn.select-dir:hover:not(:disabled) {
  background-color: #5a6268;
}

.btn.select-dir:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}
</style>
