<template>
  <div class="result-item" :data-index="index">
    <div class="result-header">
      <div class="file-info">
        <input
          type="checkbox"
          class="result-checkbox"
          :checked="isSelected"
          @change="emit('toggle')"
        />
        <span
          class="file-path"
          @click="emit('openLocation', result.filePath)"
          title="Click to show in folder"
        >{{ formatFilePath(result.filePath) }}</span>
        <span class="line-num">Line {{ result.lineNum }}</span>
        <span
          v-if="result.matchedText && result.matchedText !== query"
          class="matched-text"
        >(Matched: "{{ result.matchedText }}")</span>
      </div>
      <div class="result-actions">
        <button class="view-btn" style="margin-right: 5px" @click="emit('openPreview', result.filePath)" title="View full file">View</button>
        <button class="copy-btn" style="margin-right: 5px" @click="emit('copy', result.content)" title="Copy line">Copy</button>
        <EditorSelect
          :available-editors="availableEditors"
          @editor-select="(name) => emit('editorSelect', name, result.filePath)"
        />
      </div>
    </div>

    <!-- Display context before, match line with diff, and context after -->
    <InlineDiffView
      :content="result.content"
      :line-num="result.lineNum"
      :context-before="result.contextBefore"
      :context-after="result.contextAfter"
      :query="query"
      :case-sensitive="caseSensitive"
      :fuzzy-match-score="result.similarityScore"
      @copy="emit('copy', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { SearchResult, EditorAvailability } from "@/types";
import EditorSelect from "./EditorSelect.vue";
import InlineDiffView from "./InlineDiffView.vue";

const props = defineProps<{
  result: SearchResult;
  index: number;
  isSelected: boolean;
  formatFilePath: (filePath: string) => string;
  availableEditors: EditorAvailability;
  query: string;
  caseSensitive: boolean;
}>();

const emit = defineEmits<{
  (e: "toggle"): void;
  (e: "openLocation", filePath: string): void;
  (e: "openPreview", filePath: string): void;
  (e: "copy", text: string): void;
  (e: "editorSelect", name: string, filePath: string): void;
}>();
</script>
