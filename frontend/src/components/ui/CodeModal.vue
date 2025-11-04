<template>
  <div v-if="isVisible" class="modal-overlay" @click="closeModal">
    <div class="modal-container" @click.stop>
      <div class="modal-header">
        <h3 class="modal-title">File Preview: {{ truncatePath(filePath) }}</h3>
        <button class="modal-close-button" @click="closeModal">
          <span>&times;</span>
        </button>
      </div>

      <div class="modal-content">
        <div class="code-container" ref="codeContainerRef">
          <pre
            class="code-block"
          ><code ref="codeBlock" v-html="highlightedCode"></code></pre>
        </div>
      </div>

      <div class="modal-footer">
        <div class="modal-footer-info">
          Lines: {{ totalLines }} | Language: {{ detectedLanguage }}
          <span v-if="totalMatches > 0"> | Matches: {{ totalMatches }}</span>
        </div>
        <div class="modal-footer-actions">
          <button
            v-if="totalMatches > 0"
            class="nav-button"
            @click="goToPreviousMatch"
            title="Go to previous match"
          >
            <span>←</span>
          </button>
          <div v-if="totalMatches > 0" class="current-match-indicator">
            {{ currentMatchIndex > 0 ? `${currentMatchIndex}/${totalMatches}` : `0/${totalMatches}` }}
          </div>
          <button
            v-if="totalMatches > 0"
            class="nav-button"
            @click="goToNextMatch"
            title="Go to next match"
          >
            <span>→</span>
          </button>
          <button class="copy-button" @click="copyToClipboard">
            <span v-if="copied">Copied!</span>
            <span v-else>Copy to Clipboard</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, onMounted, nextTick, watch } from "vue";
import hljs from "highlight.js";

export default defineComponent({
  name: "CodeModal",
  props: {
    isVisible: {
      type: Boolean,
      required: true,
    },
    filePath: {
      type: String,
      required: true,
    },
    fileContent: {
      type: String,
      required: true,
    },
    query: {
      type: String,
      default: "",
    },
  },
  emits: ["close", "copy"],
  setup(props, { emit }) {
    const codeBlock = ref<HTMLElement | null>(null);
    const codeContainerRef = ref<HTMLElement | null>(null);
    const copied = ref(false);
    const currentMatchIndex = ref(0);

    const closeModal = () => {
      emit("close");
    };

    // Truncate long file paths
    const truncatePath = (path: string): string => {
      if (!path) return "";
      const maxLength = 50;
      if (path.length <= maxLength) return path;
      const parts = path.split("/");
      if (parts.length > 1) {
        return "..." + parts.slice(-2).join("/");
      }
      return path.substring(path.length - maxLength);
    };

    // Detect programming language from file extension
    const detectedLanguage = computed(() => {
      if (!props.filePath) return "text";
      const ext = props.filePath.split(".").pop()?.toLowerCase() || "";
      const languages: Record<string, string> = {
        go: "go",
        js: "javascript",
        ts: "typescript",
        java: "java",
        py: "python",
        rb: "ruby",
        php: "php",
        cpp: "cpp",
        hpp: "cpp",
        h: "c",
        c: "c",
        html: "html",
        htm: "html",
        xml: "xml",
        css: "css",
        scss: "scss",
        sass: "sass",
        json: "json",
        yaml: "yaml",
        yml: "yaml",
        md: "markdown",
        sql: "sql",
        sh: "bash",
        bash: "bash",
        rs: "rust",
        swift: "swift",
        kt: "kotlin",
        scala: "scala",
        dart: "dart",
        lua: "lua",
        pl: "perl",
        r: "r",
        coffee: "coffeescript",
        vue: "vue",
        jsx: "jsx",
        tsx: "tsx",
      };
      return languages[ext] || "text";
    });

    // Get total number of lines in file
    const totalLines = computed(() => {
      if (!props.fileContent) return 0;
      return props.fileContent.split("\n").length;
    });

    // Utility function to escape HTML
    const escapeHtml = (unsafe: string): string => {
      if (!unsafe) return "";
      return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
    };

    // Performance-optimized highlighting for large files
    const highlightedCode = computed(() => {
      if (!props.fileContent) return "";

      const language = detectedLanguage.value;

      // For very large files, we'll process in chunks to improve performance
      const lines = props.fileContent.split(/\r?\n/);

      // If file is very large, apply syntax highlighting line by line to avoid performance issues
      if (lines.length > 1000) {
        // For large files, we'll do a simplified approach to avoid performance issues
        let html = "";
        for (let i = 0; i < lines.length && i < 10000; i++) {
          // Limit to 10k lines to prevent browser crashes
          const lineNumber = i + 1;
          let lineContent = escapeHtml(lines[i]);

          // Apply syntax highlighting to individual lines if possible
          try {
            if (hljs.getLanguage(language)) {
              lineContent = hljs.highlight(lineContent, {
                language: language,
              }).value;
            }
          } catch (e) {
            // If syntax highlighting fails, use plain HTML escaped content
            lineContent = escapeHtml(lines[i]);
          }

          // Highlight query matches if query exists
          if (props.query) {
            try {
              const regex = new RegExp(
                `(${props.query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
                "gi",
              );
              lineContent = lineContent.replace(
                regex,
                '<mark class="highlight-match">$1</mark>',
              );
            } catch (e) {
              // If regex fails, continue without highlighting
            }
          }

          // Add line with number
          html += `<span class="line-number" style="margin-right:5px;margin-left:5px;" data-line="${lineNumber}">${lineNumber}</span><span class="code-line">${lineContent || " "}</span>\n`;
        }

        // Add note if we truncated the file
        if (lines.length > 10000) {
          html += `<span class="line-number" data-line="...">...</span><span class="code-line comment">/* File truncated - showing first 10,000 lines */</span>\n`;
        }

        return html;
      } else {
        // For smaller files, apply syntax highlighting to the whole content
        let highlightedCode = props.fileContent;

        try {
          if (hljs.getLanguage(language)) {
            highlightedCode = hljs.highlight(props.fileContent, {
              language: language,
            }).value;
          }
        } catch (e) {
          // If syntax highlighting fails, use plain HTML escaped content
          highlightedCode = escapeHtml(props.fileContent);
        }

        // Split code into lines
        const codeLines = highlightedCode.split(/\r?\n/);
        let html = "";

        for (let i = 0; i < codeLines.length; i++) {
          const lineNumber = i + 1;
          let lineContent = codeLines[i];

          // Highlight query matches if query exists
          if (props.query) {
            try {
              const regex = new RegExp(
                `(${props.query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
                "gi",
              );
              lineContent = lineContent.replace(
                regex,
                '<mark class="highlight-match">$1</mark>',
              );
            } catch (e) {
              // If regex fails, continue without highlighting
            }
          }

          html += `<span class="line-number" style="margin-right:5px;margin-left:5px;" data-line="${lineNumber}">${lineNumber}</span><span class="code-line">${lineContent || " "}</span>\n`;
        }

        return html;
      }
    });

    // Total number of matches
    const totalMatches = computed(() => {
      if (!props.query || !props.fileContent) return 0;

      try {
        const regex = new RegExp(
          props.query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
          "gi",
        );
        const matches = props.fileContent.match(regex);
        return matches ? matches.length : 0;
      } catch (e) {
        // If regex fails, return 0
        return 0;
      }
    });

    // Copy file content to clipboard
    const copyToClipboard = () => {
      navigator.clipboard
        .writeText(props.fileContent)
        .then(() => {
          copied.value = true;
          // Reset copied status after 2 seconds
          setTimeout(() => {
            copied.value = false;
          }, 2000);

          // Emit copy event
          emit("copy");
        })
        .catch((err) => {
          console.error("Failed to copy:", err);
        });
    };

    // Reset match index when content changes
    watch(
      () => [props.fileContent, props.query],
      () => {
        currentMatchIndex.value = 0;  // Reset to 0 when content or query changes
      }
    );

    // Function to scroll to a specific line
    const scrollToLine = (lineNumber: number) => {
      if (!codeContainerRef.value) return;

      const lineElement = codeContainerRef.value.querySelector(
        `[data-line="${lineNumber}"]`,
      );
      if (lineElement) {
        lineElement.scrollIntoView({ behavior: "smooth", block: "center" });
        // Highlight the line temporarily
        lineElement.classList.add("highlighted-line");
        setTimeout(() => {
          if (lineElement) {
            lineElement.classList.remove("highlighted-line");
          }
        }, 1500);
      }
    };

    // Function to jump to a specific line
    const jumpToLine = (lineNumber: number) => {
      if (lineNumber > 0 && lineNumber <= totalLines.value) {
        scrollToLine(lineNumber);
      }
    };

    // Navigation for highlighted matches
    // Function to calculate all match positions with better precision
    const getAllMatchPositions = () => {
      if (!codeContainerRef.value) return [];
      const matches = codeContainerRef.value.querySelectorAll(".highlight-match");
      const positions: { element: Element; index: number; position: number }[] = [];
      
      matches.forEach((match, i) => {
        const rect = match.getBoundingClientRect();
        const containerRect = codeContainerRef.value!.getBoundingClientRect();
        // Calculate position relative to the scrollable container
        const position = rect.top - containerRect.top + codeContainerRef.value!.scrollTop;
        positions.push({ element: match, index: i, position });
      });
      
      // Sort by position in the document
      positions.sort((a, b) => a.position - b.position);
      return positions;
    };

    const goToNextMatch = () => {
      if (!props.query || !props.fileContent) return;

      if (codeContainerRef.value) {
        const matchPositions = getAllMatchPositions();
        if (matchPositions.length > 0) {
          let nextIndex = 0;
          
          // If we already have a current match, go to the next one
          if (currentMatchIndex.value > 0 && currentMatchIndex.value <= matchPositions.length) {
            nextIndex = currentMatchIndex.value % matchPositions.length;
          } else {
            // Find the first match that's below the current scroll position
            const currentScrollTop = codeContainerRef.value.scrollTop;
            
            for (let i = 0; i < matchPositions.length; i++) {
              if (matchPositions[i].position > currentScrollTop) {
                nextIndex = i;
                break;
              }
            }
          }

          const nextMatch = matchPositions[nextIndex].element;
          if (nextMatch) {
            nextMatch.scrollIntoView({ behavior: "smooth", block: "center" });
            // Update the current match index - 1-indexed for display
            currentMatchIndex.value = nextIndex + 1;
          }
        }
      }
    };

    const goToPreviousMatch = () => {
      if (!props.query || !props.fileContent) return;

      if (codeContainerRef.value) {
        const matchPositions = getAllMatchPositions();
        if (matchPositions.length > 0) {
          let prevIndex = 0;

          if (currentMatchIndex.value > 1) {
            // Go to the previous match in the sequence
            prevIndex = (currentMatchIndex.value - 2 + matchPositions.length) % matchPositions.length;
          } else {
            // If we're at the first match or haven't started, go to the last match
            prevIndex = matchPositions.length - 1;
          }

          const prevMatch = matchPositions[prevIndex].element;
          if (prevMatch) {
            prevMatch.scrollIntoView({ behavior: "smooth", block: "center" });
            // Update the current match index - 1-indexed for display
            currentMatchIndex.value = prevIndex + 1;
          }
        }
      }
    };

    return {
      codeBlock,
      codeContainerRef,
      copied,
      currentMatchIndex,
      closeModal,
      truncatePath,
      detectedLanguage,
      totalLines,
      highlightedCode,
      totalMatches,
      copyToClipboard,
      scrollToLine,
      jumpToLine,
      goToNextMatch,
      goToPreviousMatch,
    };
  },
});
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-container {
  background-color: #333;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  width: 90%;
  max-width: 1200px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #555;
  background-color: #2d2d2d;
}

.modal-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: calc(100% - 40px);
}

.modal-close-button {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #ccc;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background-color 0.2s;
}

.modal-close-button:hover {
  background-color: #555;
  color: #fff;
}

.modal-content {
  flex: 1;
  overflow: auto;
  padding: 0;
  background-color: #333;
}

.code-container {
  overflow: auto;
  max-height: calc(70vh - 60px);
}

.code-block {
  margin: 0;
  padding: 0;
  background-color: #333;
  border-radius: 0;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 14px;
  line-height: 1.4;
}

.code-block code {
  display: block;
  padding: 0;
  background-color: #333 !important;
  color: #fff;
}
/* Line numbers styling */
.line-number {
  display: inline-block;
  width: 50px;
  padding: 0 12px;
  text-align: right;
  color: #888;
  background-color: #222;
  border-right: 1px solid #555;
  user-select: none;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 14px;
  position: relative;
  vertical-align: top;
  line-height: 1.4;
}

.code-line {
  display: inline-block;
  padding: 0 12px;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 14px;
  white-space: pre;
  vertical-align: top;
  line-height: 1.4;
}

/* Highlight matches - ensure they stand out against the Agate theme */
.highlight-match {
  background-color: #ffeb3b;
  color: #000 !important;
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

/* Highlighted line indicator */
.line-number.highlighted-line,
.code-line.highlighted-line {
  background-color: #5a6475 !important;
  transition: background-color 0.3s;
}

.modal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-top: 1px solid #555;
  background-color: #2d2d2d;
  color: #fff;
}

.modal-footer-info {
  color: #ccc;
  font-size: 14px;
}

.modal-footer-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-button {
  background-color: #6c757d;
  color: white;
  border: none;
  width: 32px;
  height: 32px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.2s;
}

.nav-button:hover {
  background-color: #5a6268;
}

/* Additional styling for match counter */
.current-match-indicator {
  margin: 0 10px;
  color: #ccc;
  font-size: 14px;
  min-width: 100px;
  text-align: center;
}

.copy-button {
  background-color: #4caf50;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.copy-button:hover {
  background-color: #45a049;
}

/* Scrollbar styling */
.modal-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.modal-content::-webkit-scrollbar-track {
  background: #222;
}

.modal-content::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.modal-content::-webkit-scrollbar-thumb:hover {
  background: #666;
}
</style>
