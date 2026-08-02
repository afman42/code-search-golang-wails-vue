<template>
  <div v-if="isVisible" class="modal-overlay" @click="closeModal">
    <div class="modal-container" @click.stop>
      <div class="modal-header">
        <h3 class="modal-title">File Preview: {{ truncatePath(filePath) }}</h3>
        <div class="modal-header-actions">
          <button
            v-if="showTreeView"
            class="tree-view-button"
            :class="{ active: showTreeView }"
            @click="toggleTreeView"
            title="Toggle Tree View"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
            Tree View
          </button>
          <button class="modal-close-button" @click="closeModal"><span>&times;</span></button>
        </div>
      </div>

      <div class="modal-content">
        <div v-if="activeTab === 'file'" class="code-container" ref="codeContainerRef">
          <div v-if="totalLines > 50" class="navigation-controls">
            <button class="action-button" @click="goToPreviousMatch">‹ Prev</button>
            <span class="match-info">{{ currentMatchIndex + 1 }}/{{ totalMatches }}</span>
            <button class="action-button" @click="goToNextMatch">Next ›</button>
          <input class="line-input" type="number" v-model.number="targetLine" placeholder="Line #" @keyup.enter="jumpToLine()" />
          </div>
          <pre class="code-block"><code ref="codeBlock" :key="filePath" v-html="highlightedCode"></code></pre>
        </div>
        <TreeViewPanel
          v-else-if="activeTab === 'tree'"
          :is-visible="true"
          :current-file-path="filePath"
          @file-click="handleFileClick"
        />
      </div>

      <div class="modal-footer">
        <div class="modal-footer-info">
          <span>Lines: {{ totalLines }}</span>
          <span>Language: {{ detectedLanguage }}</span>
        </div>
        <div v-if="activeTab === 'tree'" class="modal-footer-actions">
          <button class="action-button" @click="expandAllTreeItems">Expand All</button>
          <button class="action-button" @click="collapseAllTreeItems">Collapse All</button>
        </div>
        <div v-else class="modal-footer-actions">
          <button v-if="!copied" class="copy-button" @click="copyToClipboard">Copy to Clipboard</button>
          <button v-else class="copy-button success">Copied!</button>
          <button class="action-button" @click="jumpToLinePrompt">Jump to Line</button>
          <button class="action-button" @click="openFileLocation">Show in Folder</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { ShowInFolder } from '../../../wailsjs/go/main/App';
import TreeViewPanel from './TreeViewPanel.vue';
import { toastManager } from '../../composables/useToast';
import { toErrorMessage } from '../../utils/errorUtils';
import { useCodeHighlighting } from '../../composables/useCodeHighlighting';
import { useMatchNavigation } from '../../composables/useMatchNavigation';

interface Props {
  isVisible: boolean;
  filePath: string;
  fileContent: string;
  query?: string;
}

const props = withDefaults(defineProps<Props>(), { query: '' });
const emit = defineEmits<{ close: []; copy: [] }>();

const codeBlock = ref<HTMLElement | null>(null);
const codeContainerRef = ref<HTMLElement | null>(null);

const fileContentFn = () => props.fileContent;
const filePathFn = () => props.filePath;
const queryFn = () => props.query || '';

const { highlightedCodeRef, isReady, detectedLanguage, loadAndHighlight } = useCodeHighlighting(
  fileContentFn,
  filePathFn,
  queryFn
);

const {
  currentMatchIndex,
  totalMatches: totalMatchesFn,
  refreshMatchObserver,
  goToNextMatch,
  goToPreviousMatch,
} = useMatchNavigation(() => codeContainerRef.value, fileContentFn, queryFn);

const totalMatches = computed(() => totalMatchesFn());
const highlightedCode = computed(() => highlightedCodeRef.value);
const totalLines = computed(() => (props.fileContent ? props.fileContent.split('\n').length : 0));
const copied = ref(false);
const targetLine = ref<number | null>(null);
const showTreeView = ref(false);
const activeTab = ref('file');

const closeModal = () => emit('close');

const truncatePath = (path: string): string => {
  if (!path) return '';
  const maxLength = 50;
  if (path.length <= maxLength) return path;
  const parts = path.split('/');
  if (parts.length > 1) return '...' + parts.slice(-2).join('/');
  return path.substring(path.length - maxLength);
};

const toggleTreeView = () => {
  showTreeView.value = !showTreeView.value;
  activeTab.value = showTreeView.value ? 'tree' : 'file';
};

watch(
  () => props.filePath,
  (newPath) => {
    if (newPath && showTreeView.value) {
      // Tree data handled by child component
    }
  },
  { immediate: true }
);

(async () => {
  await loadAndHighlight();
})();

watch([isReady, highlightedCodeRef], async ([ready]) => {
  if (ready) await refreshMatchObserver();
});

const copyToClipboard = () => {
  navigator.clipboard
    .writeText(props.fileContent)
    .then(() => {
      copied.value = true;
      setTimeout(() => { copied.value = false; }, 2000);
      emit('copy');
    })
    .catch((err) => {
      toastManager.error(`Failed to copy: ${err}`);
      console.error('Failed to copy:', err);
    });
};

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.isVisible) closeModal();
};

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

const scrollToLine = (lineNumber: number) => {
  if (!codeContainerRef.value) return;
  const lineElement = codeContainerRef.value.querySelector(`[data-line="${lineNumber}"]`);
  if (lineElement) {
    lineElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
    lineElement.classList.add('highlighted-line');
    setTimeout(() => { if (lineElement) lineElement.classList.remove('highlighted-line'); }, 1500);
  }
};

const jumpToLine = (lineNumber?: number) => {
  const line = lineNumber ?? targetLine.value ?? 0;
  if (line > 0 && line <= totalLines.value) scrollToLine(line);
};

const jumpToLinePrompt = () => {
  const input = prompt('Enter line number:');
  if (input) {
    const line = parseInt(input, 10);
    if (!isNaN(line)) jumpToLine(line);
  }
};

const openFileLocation = async () => {
  try {
    if (!props.filePath) {
      console.warn('No file path provided to openFileLocation');
      return;
    }
    await ShowInFolder(props.filePath);
    console.log('Successfully opened file location:', props.filePath);
  } catch (error: unknown) {
    console.error('Failed to open file location:', error);
    const errorMessage = toErrorMessage(error, 'Operation failed');
    console.error(`Could not open file location: ${errorMessage}`);
    toastManager.error(`Could not open file location: ${errorMessage}`);
  }
};

const handleFileClick = (filePath: string) => {
  console.log('Clicked on file:', filePath);
};

const expandAllTreeItems = () => {
  // Handled by TreeViewPanel
};

const collapseAllTreeItems = () => {
  // Handled by TreeViewPanel
};
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-container {
  background-color: var(--color-bg-primary);
  border-radius: 8px;
  width: 90%;
  max-width: 1200px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-medium);
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.modal-header-actions {
  display: flex;
  gap: 8px;
}

.tree-view-button {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  font-size: 13px;
  border: 1px solid var(--color-border-medium);
  border-radius: 4px;
  background-color: var(--color-bg-secondary);
  color: var(--color-text-primary);
  cursor: pointer;
  transition: background-color 0.2s;
}

.tree-view-button:hover {
  background-color: var(--color-bg-hover);
}

.tree-view-button.active {
  background-color: var(--color-accent);
  border-color: var(--color-accent);
  color: white;
}

.modal-close-button {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.modal-close-button:hover {
  color: var(--color-text-primary);
}

.modal-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.code-container {
  flex: 1;
  overflow: auto;
  padding: 16px;
  background-color: var(--color-bg-secondary);
}

.code-block {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-primary);
}

.code-block code {
  white-space: pre;
  word-wrap: normal;
  display: block;
}

.highlighted-line {
  background-color: rgba(255, 215, 0, 0.3) !important;
  border-left: 3px solid gold;
}

.modal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-top: 1px solid var(--color-border-medium);
  background-color: var(--color-bg-tertiary);
}

.modal-footer-info {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.modal-footer-actions {
  display: flex;
  gap: 8px;
}

.action-button {
  padding: 6px 12px;
  font-size: 13px;
  border: 1px solid var(--color-border-medium);
  border-radius: 4px;
  background-color: var(--color-bg-secondary);
  color: var(--color-text-primary);
  cursor: pointer;
  transition: all 0.2s;
}

.action-button:hover {
  background-color: var(--color-bg-hover);
  border-color: var(--color-accent);
}

.action-button.success {
  background-color: #22c55e;
  border-color: #22c55e;
  color: white;
}
</style>
