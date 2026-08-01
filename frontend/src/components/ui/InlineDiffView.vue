<template>
  <div class="inline-diff-view">
    <!-- Context before match -->
    <div
      v-for="(line, idx) in contextBefore"
      :key="'before-' + idx"
      class="context-line context-before"
    >
      <span class="line-num">{{ lineNum - (contextBefore.length - idx) }}</span>
      <span class="line-content">
        <span v-html="highlightLine(line, 'before')"></span>
      </span>
    </div>

    <!-- Match line with diff highlight -->
    <div class="result-line matched">
      <span class="line-num">{{ lineNum }}</span>
      <span class="line-content">
        <span
          v-if="fuzzyMatchScore"
          class="fuzzy-badge"
          :title="`Fuzzy match (${Math.round(fuzzyMatchScore * 100)}% similarity)`"
        >
          ~
        </span>
        <span v-html="highlightLine(content)"></span>
      </span>
      <button
        class="copy-line-btn"
        @click="$emit('copy', content)"
        title="Copy this line"
      >
        📋
      </button>
    </div>

    <!-- Context after match -->
    <div
      v-for="(line, idx) in contextAfter"
      :key="'after-' + idx"
      class="context-line context-after"
    >
      <span class="line-num">{{ lineNum + contextBefore.length + 1 + idx }}</span>
      <span class="line-content">
        <span v-html="highlightLine(line, 'after')"></span>
      </span>
    </div>

    <!-- Diff-style indicators -->
    <div v-if="hasDiff" class="diff-indicators">
      <span class="diff-hint">
        Showing {{ contextBefore.length }} lines before &amp; {{ contextAfter.length }} lines after match
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import DOMPurify from "dompurify";

interface Props {
  content: string;
  lineNum: number;
  contextBefore: string[];
  contextAfter: string[];
  query: string;
  fuzzyMatchScore?: number;
}

const props = defineProps<Props>();

defineEmits<{
  (e: "copy", text: string): void;
}>();

const hasDiff = computed(() => {
  return props.contextBefore.length > 0 || props.contextAfter.length > 0;
});

const highlightLine = (line: string, position: 'before' | 'after' | 'match' = 'match'): string => {
  if (!props.query || !line) return DOMPurify.sanitize(line);

  try {
    const escapedQuery = props.query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const regex = new RegExp(`(${escapedQuery})`, "gi");
    
    let highlighted = line.replace(regex, '<mark class="diff-match">$1</mark>');
    
    if (position === 'before' && props.fuzzyMatchScore) {
      highlighted = `<span class="context-hint">… ${highlighted}</span>`;
    } else if (position === 'after' && props.fuzzyMatchScore) {
      highlighted = `<span class="context-hint">${highlighted} …</span>`;
    }
    
    return DOMPurify.sanitize(highlighted, {
      ALLOWED_TAGS: ["mark", "span"],
      ALLOWED_ATTR: ["class"],
    });
  } catch (e) {
    console.warn("Highlight failed:", e);
    return DOMPurify.sanitize(line);
  }
};
</script>

<style scoped>
.inline-diff-view {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 0.9em;
  border-left: 3px solid #3498db;
  padding: 8px 0 8px 12px;
  margin-top: 4px;
}

.result-line {
  display: flex;
  align-items: center;
  background-color: #f8f9fa;
  border-radius: 4px;
  padding: 6px 10px;
  margin: 4px 0;
  transition: background-color 0.2s;
}

.result-line.matched {
  border-left: 3px solid #f1c40f;
  background-color: #fffde7;
}

.line-num {
  color: #888;
  font-size: 0.85em;
  min-width: 40px;
  padding-right: 12px;
  text-align: right;
  user-select: none;
  opacity: 0.7;
}

.line-content {
  flex: 1;
  white-space: pre-wrap;
  word-break: break-word;
  color: #333;
}

.diff-match {
  background-color: #f1c40f;
  color: #000 !important;
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

.copy-line-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  opacity: 0;
  transition: opacity 0.2s;
  padding: 0 4px;
}

.result-line:hover .copy-line-btn {
  opacity: 0.6;
}

.copy-line-btn:hover {
  opacity: 1;
}

.context-before,
.context-after {
  display: flex;
  align-items: center;
  background-color: #f5f5f5;
  border-radius: 3px;
  padding: 4px 10px 4px 12px;
  margin: 3px 0;
  border-left: 2px solid #bdc3c7;
}

.context-before {
  border-left-color: #3498db;
}

.context-after {
  border-left-color: #9b59b6;
}

.fuzzy-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  background-color: #e74c3c;
  color: #fff;
  font-size: 0.7em;
  font-weight: bold;
  border-radius: 50%;
  margin-right: 6px;
  cursor: help;
}

.context-hint {
  color: #7f8c8d;
  font-style: italic;
}

.diff-indicators {
  margin-top: 6px;
  padding: 4px 12px;
  background-color: #f8f9fa;
  border-radius: 3px;
}

.diff-hint {
  color: #888;
  font-size: 0.75em;
}
</style>
