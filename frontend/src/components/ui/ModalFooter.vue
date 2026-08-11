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
      <button v-if="activeTab === 'file'" class="action-button" @click="$emit('jumpToLinePrompt')">Jump to Line</button>
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
}>()

defineEmits<{
  jumpToLinePrompt: []
  openFileLocation: []
  copyToClipboard: []
}>()
</script>

<style>
.modal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border);
}

.modal-footer-info {
  display: flex;
  gap: 12px;
  font-size: 0.85em;
  color: var(--color-text-muted);
}

.info-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.modal-footer-actions {
  display: flex;
  gap: 8px;
}

.action-button,
.copy-button {
  padding: 6px 12px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.85em;
  transition: all 0.2s;
}

.action-button:hover,
.copy-button:hover:not(.success) {
  background-color: var(--color-bg-secondary);
}

.copy-button.success {
  background-color: var(--color-success);
  border-color: var(--color-success);
  color: white;
}
</style>
