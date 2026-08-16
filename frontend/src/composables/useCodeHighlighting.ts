import { ref, computed, watch } from "vue";
import type { Ref } from "vue";
import {
  highlightCode,
  detectLanguage,
} from "@/services";
import { escapeHtml } from "@/utils/htmlUtils";

const MAX_LINES_FOR_RENDERING = 10000;

export function useCodeHighlighting(
  fileContent: () => string,
  filePath: () => string,
  query: () => string,
  addLineNumbers: Ref<boolean> = ref(true),
) {
  const highlightedCodeRef = ref("");
  const isReady = ref(false);
  // Generation counter: each loadAndHighlight run captures its own token so a
  // slow async highlight (e.g. first-time hljs load) cannot overwrite the
  // display after a newer file/query has already re-rendered.
  let generation = 0;

  const detectedLanguage = computed(() => {
    return detectLanguage(filePath());
  });


  const renderPlainText = (): string => {
    const content = fileContent();
    if (!content) return "";

    const q = query();
    let queryRegex: RegExp | null = null;
    if (q) {
      try {
        queryRegex = new RegExp(
          `(${q.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
          "gi",
        );
      } catch {
        queryRegex = null;
      }
    }

    const showNumbers = addLineNumbers.value;
    const lines = content.split(/\r?\n/);
    let html = "";
    for (let i = 0; i < lines.length && i < MAX_LINES_FOR_RENDERING; i++) {
      let lineContent = escapeHtml(lines[i] || " ");
      if (queryRegex) {
        lineContent = lineContent.replace(
          queryRegex,
          '<mark class="highlight-match">$1</mark>',
        );
      }
      if (showNumbers) {
        html += `<span class="line-number" style="margin-right:5px;margin-left:5px;" data-line="${i + 1}">${i + 1}</span>`;
      }
      html += `<span class="code-line">${lineContent || " "}</span>\n`;
    }
    if (lines.length > MAX_LINES_FOR_RENDERING) {
      html += `<span class="line-number" data-line="...">...</span><span class="code-line comment">/* File truncated - showing first 10,000 lines */</span>\n`;
    }
    return html;
  };

  const loadAndHighlight = async () => {
    const myGeneration = ++generation;
    const content = fileContent();
    if (!content) {
      highlightedCodeRef.value = "";
      isReady.value = true;
      return;
    }

    highlightedCodeRef.value = renderPlainText();
    isReady.value = true;

    try {
      const highlightedCodeResult = await highlightCode(content, {
        language: detectedLanguage.value,
        query: query(),
        addLineNumbers: addLineNumbers.value,
      });
      // Discard stale results: a newer loadAndHighlight already superseded
      // this run (rapid file switching), so committing here would flash the
      // previous file's HTML over the current one.
      if (myGeneration !== generation) return;
      if (highlightedCodeResult) {
        highlightedCodeRef.value = highlightedCodeResult;
      }
    } catch (e) {
      console.error("Error highlighting code", e);
    }
  };

  watch(
    () => [fileContent(), query(), detectedLanguage.value, addLineNumbers.value],
    async () => {
      // Don't blank isReady when we already have content — the old highlighted
      // HTML (with its data-line spans) stays in the DOM until the new pass
      // replaces it. Flipping isReady false here renders the code-placeholder
      // branch (plain text, no data-line attrs), which makes any pending
      // waitForHighlightReady poll miss the target element and give up.
      // Only clear when content is truly gone (empty string).
      if (!fileContent()) {
        isReady.value = false;
      }
      await loadAndHighlight();
    },
    { immediate: false },
  );

  return {
    highlightedCodeRef,
    isReady,
    detectedLanguage,
    loadAndHighlight,
    renderPlainText,
  };
}