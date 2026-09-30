<template>
  <div class="replace-row">
    <label for="replace-input" class="sr-only">Replace text</label>
    <input
      id="replace-input"
      v-model="replacement"
      class="replace-input"
      placeholder="Replace matches with…"
      aria-label="Replace matches with"
      :disabled="isPreviewing || isApplying || disabled"
    />
    <button
      class="replace-btn"
      aria-label="Preview replace"
      :disabled="isPreviewing || isApplying || disabled || !replacement"
      @click="emit('preview')"
    >
      <span v-if="isPreviewing" class="replace-spinner" aria-hidden="true"></span>
      {{ isPreviewing ? "Previewing…" : "Preview Replace" }}
    </button>
    <button
      class="replace-btn apply"
      :aria-label="canApply ? applyLabel : 'Apply replace'"
      :disabled="isApplying || isPreviewing || disabled || !canApply"
      @click="emit('apply')"
    >
      <span v-if="isApplying" class="replace-spinner" aria-hidden="true"></span>
      {{ isApplying ? "Applying…" : applyLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
// Find & Replace controls. Presentational only: SearchResults owns useReplace
// and passes the live flags + derived apply state down as props.
const replacement = defineModel<string>({ required: true });

withDefaults(
  defineProps<{
    isPreviewing?: boolean;
    isApplying?: boolean;
    canApply?: boolean;
    applyLabel?: string;
    disabled?: boolean;
  }>(),
  { isPreviewing: false, isApplying: false, canApply: false, applyLabel: "Apply ", disabled: false }
);

const emit = defineEmits<{
  preview: [];
  apply: [];
}>();
</script>

<style scoped>
/* Replace row */
.replace-row {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  margin-bottom: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-sm);
}
.replace-input {
  flex: 1;
  padding: 0.375rem 0.5rem;
  font-size: 0.85rem;
  font-family: var(--font-mono, monospace);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
}
.replace-input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px var(--color-accent-light);
}
.replace-btn {
  padding: 0.375rem 0.75rem;
  font-size: 0.85rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
  cursor: pointer;
  white-space: nowrap;
}
.replace-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.replace-btn.apply {
  background: var(--color-accent);
  color: var(--color-text-inverse);
  border-color: var(--color-accent);
}
.replace-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}
.replace-spinner {
  width: 0.8em;
  height: 0.8em;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: replace-spin 0.7s linear infinite;
}
@keyframes replace-spin {
  to { transform: rotate(360deg); }
}
</style>
