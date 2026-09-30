<template>
  <div class="empty-state-container" role="status" aria-live="polite">
    <div class="empty-state-icon" aria-hidden="true">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="11" cy="11" r="8" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
        <line x1="8" y1="11" x2="14" y2="11" />
      </svg>
    </div>
    <h3 class="empty-state-title">No matches found</h3>
    <p class="empty-state-text">
      No results for <strong>"{{ query }}"</strong><span v-if="extension"> in <code>{{ extension }}</code> files</span>.
      Try adjusting your query, extension filter, or search options.
    </p>
    <p v-if="failedFiles > 0" class="empty-state-hint">
      {{ failedFiles }} file(s) could not be read and were skipped.
    </p>
  </div>
</template>

<script setup lang="ts">
// Zero-result state. Purely presentational: SearchResults decides *when* to
// show it (shouldShowEmptyState) and passes what to show as props.
withDefaults(
  defineProps<{
    query?: string;
    extension?: string;
    failedFiles?: number;
  }>(),
  { query: "", extension: "", failedFiles: 0 }
);
</script>

<style scoped>
/* Empty state – shown when search returned 0 matches */
.empty-state-container {
  max-width: 32rem;
  margin: var(--space-6) auto;
  padding: var(--space-6) var(--space-5);
  text-align: center;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.empty-state-icon {
  color: var(--color-text-muted);
  margin-bottom: var(--space-3);
  display: flex;
  justify-content: center;
}

.empty-state-title {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--color-text-primary);
}

.empty-state-text {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.empty-state-text code {
  font-family: var(--font-mono);
  background: var(--color-bg-tertiary);
  padding: 1px var(--space-1);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
}

.empty-state-hint {
  margin: var(--space-3) 0 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
