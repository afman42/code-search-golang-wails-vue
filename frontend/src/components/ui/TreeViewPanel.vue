<template>
  <div class="tree-view-panel" ref="containerRef">
    <div v-if="currentFilePath" class="tree-header">
      <h4>File Explorer</h4>
      <span class="active-file">{{ currentFilePath }}</span>
    </div>
    <p v-else class="no-selection">Select a file to view the tree structure</p>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';

interface Props {
  isVisible: boolean;
  currentFilePath: string | null;
}

defineProps<Props>();
const emit = defineEmits<{(e: 'fileClick', path: string): void}>();
const containerRef = ref<HTMLElement | null>(null);

// Placeholder for tree structure - full implementation pending
onMounted(() => {
  console.log('TreeViewPanel mounted');
});
</script>

<style scoped>
.tree-view-panel {
  padding: 0.5rem;
  border-top: 1px solid var(--color-border);
}

.tree-header {
  margin-bottom: 0.5rem;
}

h4 {
  margin: 0 0 0.25rem 0;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}

.active-file {
  display: block;
  font-size: 0.75rem;
  color: var(--color-text-primary);
  word-break: break-all;
}

.no-selection {
  color: var(--color-text-muted);
  font-size: 0.75rem;
  text-align: center;
  padding: 1rem 0;
}
</style>
