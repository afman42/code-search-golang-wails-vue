<template>
  <div v-if="progress" class="replace-progress" role="status" aria-live="polite" aria-atomic="true">
    <div class="replace-progress-info">
      <span class="replace-progress-phase">{{ progress.phase }}… {{ progress.processedFiles }}/{{ progress.totalFiles }} files</span>
      <span v-if="progress.currentFile" class="replace-progress-file" :title="progress.currentFile">{{ formatFilePath(progress.currentFile) }}</span>
    </div>
    <div class="replace-progress-bar" role="progressbar" :aria-valuenow="progress.processedFiles" :aria-valuemin="0" :aria-valuemax="progress.totalFiles || 100">
      <div class="replace-progress-fill" :style="{ width: (progress.totalFiles > 0 ? progress.processedFiles / progress.totalFiles * 100 : 0) + '%' }"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ReplaceProgress } from '@/types';

interface Props {
  progress: ReplaceProgress | null;
  formatFilePath: (p: string) => string;
}

defineProps<Props>();
</script>

<style scoped>
.replace-progress {
  margin-bottom: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-tertiary);
  font-size: var(--font-size-sm);
}
.replace-progress-info {
  display: flex;
  justify-content: space-between;
  gap: var(--space-2);
  color: var(--color-text-muted);
}
.replace-progress-phase {
  text-transform: capitalize;
  font-weight: 500;
}
.replace-progress-file {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 60%;
}
.replace-progress-bar {
  margin-top: var(--space-1);
  height: 4px;
  border-radius: 2px;
  background: var(--color-bg-secondary);
  overflow: hidden;
}
.replace-progress-fill {
  height: 100%;
  background: var(--color-warning);
  transition: width var(--transition-base);
}
</style>
