<template>
  <div v-if="isVisible" class="modal-overlay" @click="closeModal">
    <div class="modal-container" @click.stop>
      <!-- Header -->
      <div class="modal-header">
        <h3 class="modal-title">File Preview: {{ truncatePath(currentPath) }}</h3>
        <div class="modal-header-actions">
          <button
            v-if="files.length > 0"
            class="tree-view-button"
            :class="{ active: activeTab === 'tree' }"
            @click="toggleTreeView"
            title="Toggle Tree View"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7" rx="1"/>
              <rect x="14" y="3" width="7" height="7" rx="1"/>
              <rect x="3" y="14" width="7" height="7" rx="1"/>
              <rect x="14" y="14" width="7" height="7" rx="1"/>
            </svg>
            Tree View
          </button>
          <button class="modal-close-button" @click="closeModal">&times;</button>
        </div>
      </div>

      <!-- Body -->
      <div class="modal-content">
        <div v-if="activeTab === 'file'" class="code-container" ref="codeContainerRef">
          <!-- Match Navigation Controls -->
          <MatchNavigationControls
            ref="matchNavRef"
            :current-match-index="currentMatchIndex"
            :total-matches="totalMatches"
            :total-lines="totalLines"
            :target-line="targetLine"
            :show-line-numbers="showLineNumbers"
            @go-to-previous-match="goToPreviousMatch"
            @go-to-next-match="goToNextMatch"
            @jump-to-line="jumpToLine"
            @clear-selection="clearSelection"
            @toggle-line-numbers="toggleLineNumbers"
          />

          <!-- Code display -->
          <code v-if="isReady" :key="currentPath" class="code-block" v-html="highlightedCode"></code>
          <code v-else-if="currentContent" class="code-placeholder">{{ currentContent }}</code>
        </div>

        <TreeViewPanel
          v-else-if="activeTab === 'tree'"
          :is-visible="true"
          :current-file-path="currentPath"
          :files="files"
          @file-click="handleFileClick"
        />
      </div>

      <!-- Footer -->
      <ModalFooter
        :total-lines="totalLines"
        :detected-language="detectedLanguage"
        :active-tab="activeTab"
        :copied="copied"
        @jump-to-line-prompt="jumpToLinePrompt"
        @open-file-location="openFileLocation"
        @copy-to-clipboard="copyToClipboard"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { ReadFile, ShowInFolder } from '@wails/go/main/App'
import { TreeViewPanel, MatchNavigationControls, ModalFooter } from '@/components/ui'
import {
  useCodeHighlighting,
  useMatchNavigation,
  toastManager,
} from '@/composables'
import { toErrorMessage } from '@/utils'

interface Props {
  isVisible: boolean
  filePath: string
  fileContent: string
  query?: string
  files?: string[]
  initialLine?: number | null
}

const props = withDefaults(defineProps<Props>(), {
  query: '',
  files: () => [],
  initialLine: null,
})

// Local copy of the displayed file so tree navigation can swap files without
// round-tripping through the parent's props. Resynced whenever the parent
// opens a different file.
const currentPath = ref(props.filePath || '')
const currentContent = ref(props.fileContent || '')

watch(
  () => props.filePath,
  (value) => {
    currentPath.value = value || ''
  },
)
watch(
  () => props.fileContent,
  (value) => {
    currentContent.value = value || ''
  },
)

const emit = defineEmits<{ close: []; copy: [] }>()

const codeContainerRef = ref<HTMLElement | null>(null)
const matchNavRef = ref<InstanceType<typeof MatchNavigationControls> | null>(null)

const copied = ref(false)
const targetLine = ref<number | null>(null)
const activeTab = ref('file')
const showLineNumbers = ref(true)

const { highlightedCodeRef, isReady, detectedLanguage, loadAndHighlight } = useCodeHighlighting(
  () => currentContent.value, () => currentPath.value, () => props.query || '', showLineNumbers
)

const {
  currentMatchIndex, totalMatches: totalMatchesFn, refreshMatchObserver,
  goToNextMatch, goToPreviousMatch,
} = useMatchNavigation(() => codeContainerRef.value, () => currentContent.value, () => props.query || '')

const totalMatches = computed(() => totalMatchesFn())
const highlightedCode = computed(() => highlightedCodeRef.value)
const totalLines = computed(() => (currentContent.value ? currentContent.value.split('\n').length : 0))

const closeModal = () => emit('close')

const truncatePath = (path: string): string => {
  if (!path) return ''
  if (path.length <= 52) return path
  const parts = path.split('/')
  return parts.length > 1 ? '.../' + parts.slice(-2).join('/') : path.slice(-52)
}

const toggleTreeView = () => {
  activeTab.value = activeTab.value === 'tree' ? 'file' : 'tree'
}

const focusGuardedFor = ref<string | null>(null)
watch([isReady, highlightedCodeRef], async ([ready]) => {
  if (ready) {
    await refreshMatchObserver()
    // Focus the line-jump input once per opened file (large files only). Guarded
    // so re-highlights (query change, line-number toggle) don't steal focus.
    if (totalLines.value > 50 && activeTab.value !== 'tree' && focusGuardedFor.value !== currentPath.value) {
      focusGuardedFor.value = currentPath.value
      setTimeout(() => matchNavRef.value?.focusLineInput(), 120)
    }
  }
})

; (async () => { await loadAndHighlight() })()

// Jump to an initial line (e.g. from symbol-search navigation).
watch(
  () => props.initialLine,
  async (line) => {
    if (!line || line <= 0 || !props.isVisible) return
    if (activeTab.value !== 'file') activeTab.value = 'file'
    try {
      await waitForHighlightReady()
      scrollToLine(line)
    } catch {
      // Timeout — target line element never appeared. Don't attempt a
      // scrollIntoView against a missing element (silent no-op).
    }
  },
  { immediate: true },
)

// Helper: resolve when content is loaded, highlight is ready, AND the target
// line element exists in the DOM. Retries up to 100 times with 50ms delay
// (5 seconds total) to handle async ReadFile (Go IPC) + highlight.js
// rendering. Rejects on timeout so the caller knows the element is missing
// and can skip the scroll instead of running scrollToLine against nothing.
function waitForHighlightReady(): Promise<void> {
  return new Promise((resolve, reject) => {
    let attempts = 0
    const check = () => {
      attempts++
      const line = props.initialLine
      if (!line || line <= 0) return resolve()

      const ready =
        totalLines.value > 0 &&
        isReady.value &&
        !!codeContainerRef.value?.querySelector(`[data-line="${line}"]`)

      if (ready) {
        resolve()
        return
      }
      if (attempts >= 100) {
        reject(new Error('waitForHighlightReady: timed out waiting for DOM'))
        return
      }
      setTimeout(check, 50)
    }
    // Small initial delay so the v-if render + highlight pass can start.
    setTimeout(check, 100)
  })
}

const copyToClipboard = () => {
  navigator.clipboard.writeText(props.fileContent).then(() => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
    emit('copy')
  }).catch((err: unknown) => { toastManager.error(`Failed to copy: ${err}`) })
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.isVisible) closeModal()
  if (e.ctrlKey && props.isVisible) {
    if (e.key === 'ArrowUp') { e.preventDefault(); goToPreviousMatch() }
    if (e.key === 'ArrowDown') { e.preventDefault(); goToNextMatch() }
  }
}
onMounted(() => document.addEventListener('keydown', handleKeydown))
onUnmounted(() => document.removeEventListener('keydown', handleKeydown))

const scrollToLine = (n: number) => {
  const el = codeContainerRef.value?.querySelector(`[data-line="${n}"]`)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  el.classList.add('highlighted-line')
  setTimeout(() => el.classList.remove('highlighted-line'), 1600)
}

const jumpToLine = (lineNumber?: number) => {
  const line = lineNumber ?? targetLine.value ?? 0
  if (line > 0 && line <= totalLines.value) {
    scrollToLine(line)
    targetLine.value = line
  } else {
    toastManager.warning(`Line must be 1–${totalLines.value}`)
  }
}

const jumpToLinePrompt = () => {
  const raw = prompt('Enter line number:')
  if (raw) { const n = parseInt(raw, 10); if (!isNaN(n)) jumpToLine(n) }
}

const openFileLocation = async () => {
  try { if (currentPath.value) await ShowInFolder(currentPath.value) }
  catch (e: unknown) { toastManager.error(toErrorMessage(e, 'Could not open location')) }
}

// Load a file selected in the tree view and show it in the file tab.
const handleFileClick = async (path: string) => {
  if (!path || path === currentPath.value) return
  try {
    const content = await ReadFile(path)
    currentPath.value = path
    currentContent.value = content
    targetLine.value = null
    copied.value = false
    activeTab.value = 'file'
    toastManager.success(`Loaded ${path}`)
  } catch (error) {
    toastManager.error(toErrorMessage(error, 'Could not open file'))
  }
}

const clearSelection = () => { targetLine.value = null }
const toggleLineNumbers = () => { showLineNumbers.value = !showLineNumbers.value }
</script>

<style scoped>
/* === Layout === */
.modal-overlay {
  position: fixed; inset: 0; z-index: 1000;
  display: flex; align-items: center; justify-content: center;
  background: var(--color-bg-overlay); backdrop-filter: blur(4px);
  padding: var(--space-4);
}

.modal-container {
  width: min(90vw, var(--modal-max-width)); max-height: var(--modal-max-height);
  display: flex; flex-direction: column;
  background: var(--color-bg); border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg); border: 1px solid var(--color-border-medium);
  overflow: hidden;
}

/* Header */
.modal-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--space-3) var(--space-5);
  border-bottom: 1px solid var(--color-border);
  background: linear-gradient(180deg, var(--color-bg-secondary), var(--color-bg));
  flex-shrink: 0;
  position: sticky; top: 0; z-index: 10;
}

.modal-title {
  font-size: var(--font-size-sm); font-weight: 600;
  color: var(--color-text-primary); margin: 0;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 75%;
}

.modal-header-actions { display: flex; gap: var(--space-2); align-items: center; }

.tree-view-button {
  display: flex; align-items: center; gap: 5px;
  padding: 5px 10px; font-size: var(--font-size-xs); font-weight: 500;
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-secondary); color: var(--color-text-primary);
  cursor: pointer; transition: all var(--transition-fast);
}
.tree-view-button:hover { background: var(--color-bg-hover); transform: translateY(-1px); box-shadow: var(--shadow-sm); }
.tree-view-button.active { background: var(--color-accent); color: var(--color-text-inverse); border-color: var(--color-accent); }

.modal-close-button {
  display: flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border: none; background: transparent;
  font-size: 24px; color: var(--color-text-muted); cursor: pointer;
  border-radius: var(--radius-md); transition: all var(--transition-fast);
}
.modal-close-button:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }

/* Body */
.modal-content { flex: 1; overflow: hidden; display: flex; flex-direction: column; }

.code-container {
  flex: 1; overflow: auto; background: var(--color-code-bg);
  position: relative; display: flex; flex-direction: column;
}

/* Code */
.code-block {
  display: block; margin: 0;
  font-family: var(--font-mono); font-size: var(--font-size-sm); line-height: 1.65;
  color: var(--color-text-primary); padding: var(--space-4);
  white-space: pre; tab-size: 4; flex: 1;
}

.code-placeholder {
  font-family: var(--font-mono); font-size: var(--font-size-sm);
  padding: var(--space-4); color: var(--color-text-muted);
  white-space: pre-wrap;
}

.code-block :deep(.line-number) {
  display: inline-block; width: 3.2em; text-align: right;
  margin-right: 1.2em; color: var(--color-code-line-num);
  border-right: 1px solid var(--color-code-border-ln);
  padding-right: 0.8em; user-select: none; font-size: 0.92em;
}

.code-block :deep(.highlight-match) {
  background: var(--color-highlight-match); border-radius: 2px; padding: 0 1px;
}

.code-block :deep(.code-line) { display: inline; }

.code-block :deep(.highlighted-line) {
  animation: flash 1.6s ease-out;
}

@keyframes flash {
  0%   { background: rgba(var(--color-accent-rgb), 0.25); border-left: 3px solid var(--color-accent); }
  100% { background: transparent; border-left: 3px solid transparent; }
}

/* Responsive */
@media (max-width: 768px) {
  .modal-overlay { padding: 0; }
  .modal-container { width: 100%; max-height: 100vh; border-radius: 0; }
  .modal-title { max-width: 65%; }
}
</style>
