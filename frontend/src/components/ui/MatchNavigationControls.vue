<template>
  <div class="navigation-controls" v-if="totalLines > 50">
    <div class="nav-left">
      <button 
        class="nav-button" 
        :disabled="currentMatchIndex <= 0" 
        @click="$emit('goToPreviousMatch')"
        title="Previous match (Ctrl+↑)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
      </button>
      <span class="match-counter">{{ totalMatches > 0 ? Math.max(1, currentMatchIndex) : 0 }} / {{ totalMatches }}</span>
      <button 
        class="nav-button" 
        :disabled="currentMatchIndex >= totalMatches" 
        @click="$emit('goToNextMatch')"
        title="Next match (Ctrl+↓)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6"/>
        </svg>
      </button>
    </div>
    <div class="nav-right">
      <div class="line-jump-group">
        <input 
          ref="lineInputRef" 
          class="line-input" 
          type="number" 
          :value="lineJumpValue"
          @input="handleLineInputChange"
          @keyup.enter="$emit('jumpToLine')"
          @focus="$emit('clearSelection')"
          :min="1" 
          :max="totalLines" 
          placeholder="L"
        />
        <button class="icon-button small" @click="$emit('jumpToLine')" title="Go to line">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M5 12h14"/><path d="M12 5l7 7-7 7"/>
          </svg>
        </button>
      </div>
      <button 
        class="icon-button" 
        :class="{ active: showLineNumbers }" 
        @click="$emit('toggleLineNumbers')"
        title="Toggle line numbers"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="18" height="18" rx="2"/>
          <line x1="8" y1="7" x2="16" y2="7"/>
          <line x1="8" y1="11" x2="16" y2="11"/>
          <line x1="8" y1="15" x2="16" y2="15"/>
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface Props {
  currentMatchIndex: number
  totalMatches: number
  totalLines: number
  targetLine: number | null
  showLineNumbers: boolean
}

const props = defineProps<Props>()

const lineInputRef = ref<HTMLInputElement | null>(null)

// Local state for the line jump input, synced with prop
const lineJumpValue = ref<string>('')

watch(
  () => props.targetLine,
  (val) => {
    lineJumpValue.value = val !== null && val !== undefined ? String(val) : ''
  },
  { immediate: true }
)

const handleLineInputChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  const value = target.value.trim()
  if (value === '') {
    emit('jumpToLine', null)
  } else {
    const num = Number(value)
    if (!isNaN(num)) {
      emit('jumpToLine', num)
    }
  }
}

// Exposed so the parent (CodeModal) can focus the line-jump input when a large
// file finishes highlighting. The input element lives here after the extraction,
// so the parent can no longer reach it via its own ref.
const focusLineInput = () => {
  lineInputRef.value?.focus()
}

defineExpose({ focusLineInput })

const emit = defineEmits<{
  goToNextMatch: []
  goToPreviousMatch: []
  jumpToLine: [lineNumber?: number]
  clearSelection: []
  toggleLineNumbers: []
}>()
</script>

<style>
.navigation-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
}

.nav-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-button {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xs);
  cursor: pointer;
  color: var(--color-text);
  transition: all 0.2s;
}

.nav-button:hover:not(:disabled) {
  background-color: var(--color-bg-secondary);
}

.nav-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.match-counter {
  font-size: 0.85em;
  color: var(--color-text-muted);
  min-width: 80px;
  text-align: center;
}

.line-jump-group {
  display: flex;
  align-items: center;
  gap: 4px;
}

.line-input {
  width: 50px;
  padding: 4px 6px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xs);
  font-size: 0.85em;
  text-align: center;
}

.icon-button {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xs);
  cursor: pointer;
  color: var(--color-text);
  transition: all 0.2s;
}

.icon-button.small {
  padding: 3px;
}

.icon-button:hover:not(.active) {
  background-color: var(--color-bg-secondary);
}

.icon-button.active {
  background-color: var(--color-accent);
  color: white;
  border-color: var(--color-accent);
}
</style>
