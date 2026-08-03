<template>
  <div v-if="isVisible" class="modal-overlay" @click="closeModal">
    <div class="modal-container" @click.stop>
      <!-- Header -->
      <div class="modal-header">
        <h3 class="modal-title">File Preview: {{ truncatePath(filePath) }}</h3>
        <div class="modal-header-actions">
          <button
            v-if="showTreeView"
            class="tree-view-button"
            :class="{ active: activeTab === 'tree' }"
            @click="toggleTreeView"
            title="Toggle Tree View"
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
            Tree View
          </button>
          <button class="modal-close-button" @click="closeModal">&times;</button>
        </div>
      </div>

      <!-- Body -->
      <div class="modal-content">
        <div v-if="activeTab === 'file'" class="code-container" ref="codeContainerRef">
          <!-- Navigation controls (large files only) -->
          <div v-if="totalLines > 50" class="navigation-controls">
            <div class="nav-left">
              <button class="nav-button" :disabled="currentMatchIndex <= 0" @click="goToPreviousMatch" title="Previous match (Ctrl+↑)">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
              </button>
              <span class="match-counter">{{ totalMatches > 0 ? Math.max(1, currentMatchIndex) : 0 }} / {{ totalMatches }}</span>
              <button class="nav-button" :disabled="currentMatchIndex >= totalMatches" @click="goToNextMatch" title="Next match (Ctrl+↓)">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>
              </button>
            </div>
            <div class="nav-right">
              <div class="line-jump-group">
                <input ref="lineInputRef" class="line-input" type="number" v-model.number="targetLine" :min="1" :max="totalLines" placeholder="L" @keyup.enter="jumpToLine()" @focus="clearSelection" />
                <button class="icon-button small" @click="jumpToLine()" title="Go to line">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"/><path d="M12 5l7 7-7 7"/></svg>
                </button>
              </div>
              <button class="icon-button" :class="{ active: showLineNumbers }" @click="toggleLineNumbers" title="Toggle line numbers">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="8" y1="7" x2="16" y2="7"/><line x1="8" y1="11" x2="16" y2="11"/><line x1="8" y1="15" x2="16" y2="15"/></svg>
              </button>
            </div>
          </div>

          <!-- Code display -->
          <code v-if="isReady" ref="codeBlock" :key="filePath" class="code-block" v-html="highlightedCode"></code>
          <code v-else-if="fileContent" class="code-placeholder">{{ fileContent }}</code>
        </div>

        <TreeViewPanel
          v-else-if="activeTab === 'tree'"
          :is-visible="true"
          :current-file-path="filePath"
          @file-click="handleFileClick"
        />
      </div>

      <!-- Footer -->
      <div class="modal-footer">
        <div class="modal-footer-info">
          <span class="info-item"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg> Lines: {{ totalLines }}</span>
          <span class="info-item"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg> Language: {{ detectedLanguage }}</span>
        </div>
        <div class="modal-footer-actions">
          <button v-if="activeTab === 'file'" class="action-button" @click="jumpToLinePrompt">Jump to Line</button>
          <button v-if="activeTab === 'file'" class="action-button" @click="openFileLocation">Show in Folder</button>
          <button v-if="!copied" class="copy-button" @click="copyToClipboard">Copy to Clipboard</button>
          <button v-else class="copy-button success">Copied!</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { ShowInFolder } from '../../../wailsjs/go/main/App'
import TreeViewPanel from './TreeViewPanel.vue'
import { toastManager } from '../../composables/useToast'
import { toErrorMessage } from '../../utils/errorUtils'
import { useCodeHighlighting } from '../../composables/useCodeHighlighting'
import { useMatchNavigation } from '../../composables/useMatchNavigation'

interface Props {
  isVisible: boolean
  filePath: string
  fileContent: string
  query?: string
}

const props = withDefaults(defineProps<Props>(), { query: '' })
const emit = defineEmits<{ close: []; copy: [] }>()

const codeBlock = ref<HTMLElement | null>(null)
const codeContainerRef = ref<HTMLElement | null>(null)
const lineInputRef = ref<HTMLInputElement | null>(null)

const copied = ref(false)
const targetLine = ref<number | null>(null)
const showTreeView = ref(false)
const activeTab = ref('file')
const showLineNumbers = ref(true)

const { highlightedCodeRef, isReady, detectedLanguage, loadAndHighlight } = useCodeHighlighting(
  () => props.fileContent, () => props.filePath, () => props.query || '', showLineNumbers
)

const {
  currentMatchIndex, totalMatches: totalMatchesFn, refreshMatchObserver,
  goToNextMatch, goToPreviousMatch,
} = useMatchNavigation(() => codeContainerRef.value, () => props.fileContent, () => props.query || '')

const totalMatches = computed(() => totalMatchesFn())
const highlightedCode = computed(() => highlightedCodeRef.value)
const totalLines = computed(() => (props.fileContent ? props.fileContent.split('\n').length : 0))

const closeModal = () => emit('close')

const truncatePath = (path: string): string => {
  if (!path) return ''
  if (path.length <= 52) return path
  const parts = path.split('/')
  return parts.length > 1 ? '.../' + parts.slice(-2).join('/') : path.slice(-52)
}

const toggleTreeView = () => {
  showTreeView.value = !showTreeView.value
  activeTab.value = showTreeView.value ? 'tree' : 'file'
}

const focusGuardedFor = ref<string | null>(null)
watch([isReady, highlightedCodeRef], async ([ready]) => {
  if (ready) {
    await refreshMatchObserver()
    // Focus the line-jump input once per opened file (large files only). Guarded
    // so re-highlights (query change, line-number toggle) don't steal focus.
    if (totalLines.value > 50 && !showTreeView.value && focusGuardedFor.value !== props.filePath) {
      focusGuardedFor.value = props.filePath
      setTimeout(() => lineInputRef.value?.focus(), 120)
    }
  }
})

;(async () => { await loadAndHighlight() })()

const copyToClipboard = () => {
  navigator.clipboard.writeText(props.fileContent).then(() => {
    copied.value = true; setTimeout(() => { copied.value = false }, 2000); emit('copy')
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
    scrollToLine(line); targetLine.value = line
  } else {
    toastManager.warning(`Line must be 1–${totalLines.value}`)
  }
}

const jumpToLinePrompt = () => {
  const raw = prompt('Enter line number:')
  if (raw) { const n = parseInt(raw, 10); if (!isNaN(n)) jumpToLine(n) }
}

const openFileLocation = async () => {
  try { if (props.filePath) await ShowInFolder(props.filePath) }
  catch (e: unknown) { toastManager.error(toErrorMessage(e, 'Could not open location')) }
}

const handleFileClick = (p: string) => { void p }
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

/* Navigation controls */
.navigation-controls {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--space-2) var(--space-4);
  background: var(--color-bg-tertiary);
  border-bottom: 1px solid var(--color-border);
  gap: var(--space-3); flex-shrink: 0; flex-wrap: wrap;
}

.nav-left, .nav-right { display: flex; align-items: center; gap: var(--space-2); }

.nav-button {
  display: flex; align-items: center; justify-content: center;
  width: 30px; height: 30px; padding: 0;
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-secondary); color: var(--color-text-primary);
  cursor: pointer; transition: all var(--transition-fast);
}
.nav-button:hover:not(:disabled) { background: var(--color-bg-hover); border-color: var(--color-accent); transform: translateY(-1px); }
.nav-button:disabled { opacity: 0.3; cursor: default; }

.match-counter {
  font-family: var(--font-mono); font-size: var(--font-size-xs); font-weight: 600;
  color: var(--color-text-secondary); background: var(--color-bg);
  padding: 2px var(--space-2); border-radius: var(--radius-sm);
  min-width: 48px; text-align: center;
}

.line-jump-group { display: flex; align-items: center; gap: 3px; }

.line-input {
  width: 54px; padding: 4px var(--space-2);
  font-size: var(--font-size-xs); font-family: var(--font-mono);
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg); color: var(--color-text-primary);
  text-align: center;
}
.line-input:focus { outline: none; border-color: var(--color-accent); box-shadow: 0 0 0 2px var(--color-accent-light); }

.icon-button {
  display: flex; align-items: center; justify-content: center;
  width: 30px; height: 30px; padding: 0;
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-secondary); color: var(--color-text-primary);
  cursor: pointer; transition: all var(--transition-fast);
}
.icon-button:hover { background: var(--color-bg-hover); border-color: var(--color-accent); transform: translateY(-1px); }
.icon-button.active { background: var(--color-accent); color: var(--color-text-inverse); border-color: var(--color-accent); }
.icon-button.small { width: 30px; height: 30px; }

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

/* Footer */
.modal-footer {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--space-2) var(--space-4);
  border-top: 1px solid var(--color-border);
  background: var(--color-bg-tertiary); flex-shrink: 0;
  gap: var(--space-2); flex-wrap: wrap;
}

.modal-footer-info {
  display: flex; gap: var(--space-4);
  font-family: var(--font-mono); font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.info-item { display: flex; align-items: center; gap: 4px; }

.modal-footer-actions { display: flex; gap: var(--space-2); flex-wrap: wrap; }

.action-button {
  padding: 7px 12px; font-size: var(--font-size-xs); font-weight: 500;
  border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-secondary); color: var(--color-text-primary);
  cursor: pointer; transition: all var(--transition-fast);
}
.action-button:hover { background: var(--color-bg-hover); border-color: var(--color-accent); transform: translateY(-1px); box-shadow: var(--shadow-sm); }

.copy-button {
  padding: 7px 12px; font-size: var(--font-size-xs); font-weight: 500;
  border: 1px solid var(--color-accent); border-radius: var(--radius-md);
  background: linear-gradient(135deg, var(--color-accent), var(--color-accent-dark));
  color: var(--color-text-inverse); cursor: pointer; transition: all var(--transition-fast);
}
.copy-button:hover { transform: translateY(-1px); box-shadow: 0 4px 12px rgba(var(--color-accent-rgb), 0.3); }
.copy-button.success { background: var(--color-success); border-color: var(--color-success); }

/* Responsive */
@media (max-width: 768px) {
  .modal-overlay { padding: 0; }
  .modal-container { width: 100%; max-height: 100vh; border-radius: 0; }
  .modal-footer { flex-direction: column; align-items: flex-start; }
  .modal-footer-actions { width: 100%; }
  .navigation-controls { flex-direction: column; align-items: stretch; }
  .nav-left, .nav-right { justify-content: space-between; }
  .modal-title { max-width: 65%; }
}
</style>
