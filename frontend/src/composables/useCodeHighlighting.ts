import { ref, computed, watch } from "vue";
import {
  highlightCode,
  detectLanguage,
} from "../services/syntaxHighlightingService";

export function useCodeHighlighting(
  fileContent: () => string,
  filePath: () => string,
  query: () => string,
) {
  const highlightedCodeRef = ref("");
  const isReady = ref(false);

  const detectedLanguage = computed(() => {
    return detectLanguage(filePath());
  });

  const escapeHtml = (s: string): string =>
    s
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");

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

    const lines = content.split(/\r?\n/);
    const maxLines = 10000;
    let html = "";
    for (let i = 0; i < lines.length && i < maxLines; i++) {
      let lineContent = escapeHtml(lines[i] || " ");
      if (queryRegex) {
        lineContent = lineContent.replace(
          queryRegex,
          '<mark class="highlight-match">$1</mark>',
        );
      }
      html += `<span class="line-number" style="margin-right:5px;margin-left:5px;" data-line="${i + 1}">${i + 1}</span><span class="code-line">${lineContent || " "}</span>\n`;
    }
    if (lines.length > maxLines) {
      html += `<span class="line-number" data-line="...">...</span><span class="code-line comment">/* File truncated - showing first 10,000 lines */</span>\n`;
    }
    return html;
  };

  const loadAndHighlight = async () => {
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
        addLineNumbers: true,
      });
      if (highlightedCodeResult) {
        highlightedCodeRef.value = highlightedCodeResult;
      }
    } catch (e) {
      console.error("Error highlighting code", e);
    }
  };

  watch(
    () => [fileContent(), query(), detectedLanguage.value],
    async () => {
      isReady.value = false;
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