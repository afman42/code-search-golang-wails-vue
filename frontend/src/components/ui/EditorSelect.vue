<template>
  <select
    class="editor-select"
    @change="$emit('editorSelect', $event)"
    title="Open in editor"
  >
    <option value="">Editor...</option>
    <option
      v-for="editor in availableCatalogEditors"
      :key="editor.key"
      :value="editor.key"
    >{{ editor.label }}</option>
    <option value="default">System Default</option>
  </select>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { EDITOR_CATALOG } from "@/constants/editors";
import type { EditorAvailability } from "@/types";

const props = defineProps<{
  availableEditors: EditorAvailability;
}>();

defineEmits<{
  editorSelect: [event: Event];
}>();

// Only editors the backend reports as available get an <option>; the fixed
// placeholder and System Default entries are rendered unconditionally above.
const availableCatalogEditors = computed(() =>
  EDITOR_CATALOG.filter((editor) => props.availableEditors[editor.key]),
);
</script>