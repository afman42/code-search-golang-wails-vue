<template>
  <div class="modal-footer">
    <div class="modal-footer-info">
      <span class="info-item">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
        </svg>
        Lines: {{ totalLines }}
      </span>
      <span class="info-item">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="2" y1="12" x2="22" y2="12"/>
          <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
        </svg>
        Language: {{ detectedLanguage }}
      </span>
    </div>
    <div class="modal-footer-actions">
      <button v-if="activeTab === 'file' && canJumpToLine" class="action-button" @click="$emit('focusLineInput')">Jump to Line</button>
      <button v-if="activeTab === 'file'" class="action-button" @click="$emit('openFileLocation')">Show in Folder</button>
      <button v-if="!copied" class="copy-button" @click="$emit('copyToClipboard')">Copy to Clipboard</button>
      <button v-else class="copy-button success">Copied!</button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  totalLines: number
  detectedLanguage: string
  activeTab: string
  copied: boolean
  /** False when the modal has no line-jump input to focus (short files hide
   *  the navigation controls), so the button isn't offered as a no-op. */
  canJumpToLine: boolean
}>()

defineEmits<{
  focusLineInput: []
  openFileLocation: []
  copyToClipboard: []
}>()
</script>

<style scoped>
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

@media (max-width: 768px) {
  .modal-footer { flex-direction: column; align-items: flex-start; }
  .modal-footer-actions { width: 100%; }
}
</style>
