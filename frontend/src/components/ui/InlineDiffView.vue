<template>
  <div class="inline-diff-view">
    <!-- Context before match -->
    <div
      v-for="(line, idx) in contextBefore"
      :key="'before-' + idx"
      class="context-line context-before"
    >
      <span class="line-num">{{ lineNum - (contextBefore.length - idx) }}</span>
      <span class="diff-marker diff-minus">-</span>
      <span class="line-content">
        <span v-html="highlightLine(line)"></span>
      </span>
    </div>

    <!-- Match line with diff highlight -->
    <div class="result-line matched">
      <span class="line-num">{{ lineNum }}</span>
      <span class="diff-marker diff-plus">+</span>
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
      <span class="diff-marker diff-minus">-</span>
      <span class="line-content">
        <span v-html="highlightLine(line)"></span>
      </span>
    </div>

    <!-- Diff-style indicators -->
    <div v-if="hasDiff || matchCount > 1 || isTruncated" class="diff-indicators">
      <span class="diff-hint">
        <template v-if="hasDiff">
          Showing {{ contextBefore.length }} lines before &amp; {{ contextAfter.length }} lines after match
        </template>
        <template v-if="matchCount > 1"> · {{ matchCount }} matches on this line</template>
        <template v-if="isTruncated"> · line truncated</template>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  findMatchRanges,
  buildDiffSegments,
  renderDiffHtml,
} from "../../utils/diffUtils";

interface Props {
  content: string;
  lineNum: number;
  contextBefore: string[];
  contextAfter: string[];
  query: string;
  caseSensitive?: boolean;
  fuzzyMatchScore?: number;
}

const props = withDefaults(defineProps<Props>(), {
  caseSensitive: false,
  fuzzyMatchScore: undefined,
});

defineEmits<{
  (e: "copy", text: string): void;
}>();

const hasDiff = computed(() => {
  return props.contextBefore.length > 0 || props.contextAfter.length > 0;
});

// Count how many matches are on the matched line (for the diff hint).
const matchCount = computed(() => {
  if (!props.query || !props.content) return 0;
  return findMatchRanges(props.content, props.query, props.caseSensitive).length;
});

const isTruncated = computed(() => {
  return props.content.length > 200;
});

const highlightLine = (line: string): string => {
  if (!props.query || !line) return renderDiffHtml([{ text: line, type: "normal" }]);
  const ranges = findMatchRanges(line, props.query, props.caseSensitive);
  const segments = buildDiffSegments(line, ranges);
  return renderDiffHtml(segments);
};
</script>

<style scoped>
.inline-diff-view {
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 0.9em;
  border-left: 3px solid var(--color-accent);
  padding: var(--space-2) 0 var(--space-2) var(--space-3);
  margin-top: var(--space-1);
}

.result-line {
  display: flex;
  align-items: center;
  background-color: var(--color-bg-secondary);
  border-radius: var(--radius-sm);
  padding: 6px 10px;
  margin: var(--space-1) 0;
  transition: background-color 0.2s;
}

.result-line.matched {
  border-left: 3px solid var(--color-warning);
  background-color: var(--color-accent-light);
}

.line-num {
  color: var(--color-text-muted);
  font-size: 0.85em;
  min-width: 40px;
  padding-right: var(--space-3);
  text-align: right;
  user-select: none;
  opacity: 0.7;
}

.line-content {
  flex: 1;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--color-text-primary);
}

.diff-marker {
  display: inline-block;
  width: 1.2em;
  text-align: center;
  font-weight: bold;
  user-select: none;
  flex-shrink: 0;
}

.diff-plus {
  color: var(--color-success);
}

.diff-minus {
  color: var(--color-danger);
  opacity: 0.6;
}

.diff-truncation {
  color: var(--color-text-muted);
  opacity: 0.5;
  font-style: italic;
}

.diff-match {
  background-color: var(--color-warning);
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
  padding: 0 var(--space-1);
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
  background-color: var(--color-bg-secondary);
  border-radius: 3px;
  padding: var(--space-1) 10px var(--space-1) var(--space-3);
  margin: 3px 0;
  border-left: 2px solid var(--color-border-medium);
}

.context-before {
  border-left-color: var(--color-accent);
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
  background-color: var(--color-danger);
  color: var(--color-text-inverse);
  font-size: 0.7em;
  font-weight: bold;
  border-radius: 50%;
  margin-right: 6px;
  cursor: help;
}

.context-hint {
  color: var(--color-text-muted);
  font-style: italic;
}

.diff-indicators {
  margin-top: 6px;
  padding: var(--space-1) var(--space-3);
  background-color: var(--color-bg-secondary);
  border-radius: 3px;
}

.diff-hint {
  color: var(--color-text-muted);
  font-size: 0.75em;
}
</style>
