<template>
  <div v-if="preview && preview.files.length > 0" class="replace-preview">
    <div class="replace-preview-header">
      <span>{{ preview.filesChanged }} file(s), {{ preview.linesChanged }} line(s) to change</span>
      <button class="replace-clear" aria-label="Clear replace preview" @click="$emit('clear')"><span aria-hidden="true">×</span></button>
    </div>
    <ul class="replace-preview-list">
      <li v-for="file in preview.files.slice(0, 20)" :key="file.filePath + file.lineNum" class="replace-preview-item">
        <span class="replace-file">{{ formatFilePath(file.filePath) }}:{{ file.lineNum }}</span>
        <span class="replace-old" title="before">{{ file.oldLine }}</span>
        <span class="replace-arrow" aria-hidden="true">→</span>
        <span class="replace-new" title="after">{{ file.newLine }}</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import type { ReplaceResult } from '@/types';

interface Props {
  preview: ReplaceResult | null;
  formatFilePath: (p: string) => string;
}

defineProps<Props>();
defineEmits<{ (e: 'clear'): void }>();
</script>

<style scoped>
.replace-preview {
  margin-bottom: var(--space-3);
  border: 1px solid var(--color-warning);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.replace-preview-header {
  background: var(--color-bg-tertiary);
  padding: var(--space-1) var(--space-2);
  font-size: var(--font-size-sm);
  font-weight: 500;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.replace-clear {
  background: none;
  border: none;
  font-size: var(--font-size-md);
  cursor: pointer;
  color: var(--color-text-muted);
  line-height: 1;
  padding: var(--space-1);
  border-radius: var(--radius-sm);
}
.replace-clear:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}
.replace-preview-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 12.5rem;
  overflow-y: auto;
}
.replace-preview-item {
  display: flex;
  gap: var(--space-2);
  align-items: baseline;
  padding: var(--space-1) var(--space-2);
  font-size: var(--font-size-xs);
  font-family: var(--font-mono, monospace);
  border-bottom: 1px solid var(--color-border);
}
.replace-preview-item:last-child {
  border-bottom: none;
}
.replace-file {
  color: var(--color-accent);
  flex-shrink: 0;
  min-width: 20%;
}
.replace-old {
  color: var(--color-danger);
  text-decoration: line-through;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.replace-arrow {
  color: var(--color-text-muted);
  flex-shrink: 0;
}
.replace-new {
  color: var(--color-success);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
