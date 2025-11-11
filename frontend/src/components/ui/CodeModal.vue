<template>
  <div v-if="isVisible" class="modal-overlay" @click="closeModal">
    <div class="modal-container" @click.stop>
      <div class="modal-header">
        <h3 class="modal-title">File Preview: {{ truncatePath(filePath) }}</h3>
        <div class="modal-header-actions">
          <button
            class="tree-view-button"
            @click="showTreeView = !showTreeView"
            :class="{ active: showTreeView }"
            title="Toggle Tree View"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </button>
          <button class="modal-close-button" @click="closeModal">
            <span>&times;</span>
          </button>
        </div>
      </div>

      <div class="modal-content">
        <!-- Tab navigation -->
        <div class="tab-navigation" v-if="showTreeView">
          <button
            :class="['tab-button', { active: activeTab === 'file' }]"
            @click="activeTab = 'file'"
          >
            File Preview
          </button>
          <button
            :class="['tab-button', { active: activeTab === 'tree' }]"
            @click="activeTab = 'tree'"
          >
            Tree View
          </button>
        </div>

        <!-- Content based on active tab -->
        <div
          v-if="activeTab === 'file'"
          class="code-container"
          ref="codeContainerRef"
        >
          <pre
            class="code-block"
          ><code ref="codeBlock" v-if="isReady" :key="filePath" v-html="highlightedCode"></code><div v-else class="loading">Loading and highlighting code...</div></pre>
        </div>

        <!-- Tree view content -->
        <div v-else-if="activeTab === 'tree'" class="tree-view-container">
          <div class="tree-view-content">
            <div class="tree-controls">
              <h4>File Location in Project</h4>
              <div class="search-bar">
                <input
                  v-model="treeFilter"
                  type="text"
                  placeholder="Filter files..."
                  class="filter-input"
                  @keyup.esc="clearTreeFilter"
                />
                <button
                  v-if="treeFilter"
                  class="clear-filter-btn"
                  @click="clearTreeFilter"
                  title="Clear filter"
                >
                  ×
                </button>
              </div>
            </div>
            <div class="tree-structure">
              <EnhancedTreeItem
                :key="treeRefreshKey"
                :item="treeData"
                :current-file-path="filePath"
                :expanded="shouldExpandAll"
                :filter-text="treeFilter"
                :show-item-count="true"
                @file-click="handleFileClick"
              />
            </div>
            <div class="tree-actions">
              <button
                class="expand-all-btn"
                @click="expandAllTreeItems"
                title="Expand all folders"
              >
                Expand All
              </button>
              <button
                class="collapse-all-btn"
                @click="collapseAllTreeItems"
                title="Collapse all folders"
              >
                Collapse All
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <div class="modal-footer-info">
          Lines: {{ totalLines }} | Language: {{ detectedLanguage }}
          <span v-if="totalMatches > 0"> | Matches: {{ totalMatches }}</span>
        </div>
        <div v-if="activeTab === 'tree'" class="modal-footer-actions">
          <!-- Show in File Explorer button for tree view tab -->
          <button
            class="explorer-button"
            @click="openFileLocation"
            title="Show this file in file explorer"
          >
            Show in File Explorer
          </button>
        </div>
        <div v-else class="modal-footer-actions">
          <button
            v-if="totalMatches > 0"
            class="nav-button"
            @click="goToPreviousMatch"
            title="Go to previous match"
          >
            <span>←</span>
          </button>
          <div v-if="totalMatches > 0" class="current-match-indicator">
            {{
              currentMatchIndex > 0
                ? `${currentMatchIndex}/${totalMatches}`
                : `0/${totalMatches}`
            }}
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
import {
  defineComponent,
  ref,
  computed,
  nextTick,
  watch,
  onUnmounted,
} from "vue";
import DOMPurify from "dompurify";
import { ShowInFolder } from "../../../wailsjs/go/main/App"; // Import the ShowInFolder function and editor detection
import EnhancedTreeItem from "./EnhancedTreeItem.vue"; // Enhanced tree item component with filtering and navigation
import { toastManager } from "../../composables/useToast";
import { highlightCode, detectLanguage, isHighlightingReady } from "../../services/syntaxHighlightingService";

export default defineComponent({
  name: "CodeModal",
  components: {
    EnhancedTreeItem,
  },
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
    const observer = ref<IntersectionObserver | null>(null);
    const visibleMatches = ref<Set<Element>>(new Set());
    const matchElements = ref<Element[]>([]);

    // Tree view related reactive variables
    const showTreeView = ref(false);
    const activeTab = ref("file"); // 'file' or 'tree'

    // Tree filtering variables
    const treeFilter = ref("");

    // Tree expansion state
    const isTreeExpanded = ref(false);

    // Whether to expand all nodes
    const shouldExpandAll = ref(false);

    // Key to force tree refresh
    const treeRefreshKey = ref(0);

    // Define the tree structure type
    interface TreeItem {
      name: string;
      path: string;
      children: TreeItem[];
      isFile?: boolean;
      isExpanded?: boolean;
    }

    const treeData = ref<TreeItem>({
      name: "",
      path: "",
      children: [],
      isExpanded: true,
    });

    const closeModal = () => {
      emit("close");
    };

    // Function to generate tree structure from file path
    const generateTreeStructure = (filePath: string): TreeItem => {
      if (!filePath)
        return { name: "", path: "", children: [], isExpanded: true };

      // Handle both Unix (/) and Windows (\) path separators
      const normalizedPath = filePath.replace(/\\/g, "/");
      const pathParts = normalizedPath.split("/").filter((part) => part !== ""); // Remove empty parts to handle absolute paths properly
      // Use the first actual directory name from the path instead of defaulting to 'root'
      const rootName = pathParts[0] || "root";
      const root: TreeItem = {
        name: rootName,
        path: rootName,
        children: [],
        isExpanded: true,
      };

      let currentLevel: TreeItem[] = root.children;

      // Build the tree structure based on the file path starting from the second part
      for (let i = 1; i < pathParts.length; i++) {
        const part = pathParts[i];
        const isLast = i === pathParts.length - 1;

        const pathSoFar = pathParts.slice(0, i + 1).join("/");

        const node: TreeItem = {
          name: part,
          path: pathSoFar,
          children: isLast ? [] : [],
          isFile: isLast,
          isExpanded: true,
        };

        currentLevel.push(node);

        if (!isLast) {
          currentLevel = node.children;
        }
      }

      return root;
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

    // Watch for changes in file path to update tree structure
    watch(
      () => props.filePath,
      (newPath) => {
        if (newPath) {
          treeData.value = generateTreeStructure(newPath);
        }
      },
      { immediate: true },
    );

    // Detect programming language from file extension
    const detectedLanguage = computed(() => {
      return detectLanguage(props.filePath);
    });

    // Get total number of lines in file
    const totalLines = computed(() => {
      if (!props.fileContent) return 0;
      return props.fileContent.split("\n").length;
    });

    // Reactive refs to hold highlighted code and loading state
    const highlightedCodeRef = ref("");
    const isReady = ref(false);

    // Function to load highlight.js and highlight the code
    const loadAndHighlight = async () => {
      if (!props.fileContent) {
        highlightedCodeRef.value = "";
        isReady.value = true;
        return;
      }

      // Use the global syntax highlighting service
      try {
        const highlightedCodeResult = await highlightCode(props.fileContent, {
          language: detectedLanguage.value,
          query: props.query,
          addLineNumbers: true
        });
        highlightedCodeRef.value = highlightedCodeResult;
      } catch (e) {
        console.error("Error highlighting code", e);
        // Simple fallback without highlighting
        highlightedCodeRef.value = props.fileContent
          .split(/\r?\n/)
          .map((line, i) => `<span class="line-number" style="margin-right:5px;margin-left:5px;" data-line="${i + 1}">${i + 1}</span><span class="code-line">${DOMPurify.sanitize(line || " ", { ALLOWED_TAGS: [], ALLOWED_ATTR: [] }) || " "}</span>\n`)
          .join('');
      }

      isReady.value = true;
    };

    // Initialize highlighting when component is set up
    (async () => {
      isReady.value = false;
      await loadAndHighlight();
    })();

    // Watch for changes in file content and run highlighting
    watch(
      () => [props.fileContent, props.query, detectedLanguage.value],
      async () => {
        isReady.value = false;
        await loadAndHighlight();
      },
      { immediate: false },
    ); // Don't run immediately since we already called it above

    // Computed property to return the highlighted code ref
    const highlightedCode = computed(() => highlightedCodeRef.value);

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

    // Initialize Intersection Observer for detecting visible matches
    const initIntersectionObserver = () => {
      if (!codeContainerRef.value) return;

      // Disconnect any existing observer
      if (observer.value) {
        observer.value.disconnect();
      }

      // Create a new Intersection Observer instance
      observer.value = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              visibleMatches.value.add(entry.target);
            } else {
              visibleMatches.value.delete(entry.target);
            }
          });
        },
        {
          root: codeContainerRef.value,
          rootMargin: "100px", // Trigger 100px before element becomes visible
          threshold: 0.1, // Trigger when 10% of element is visible
        },
      );
    };

    // Set up observer after content is rendered
    watch(
      isReady,
      async (ready) => {
        if (ready && codeContainerRef.value) {
          await nextTick(); // Wait for DOM to update

          // Find all highlighted matches and update matchElements
          const matches =
            codeContainerRef.value.querySelectorAll(".highlight-match");
          matchElements.value = Array.from(matches);

          // Initialize the observer
          initIntersectionObserver();

          // Clear previous observations
          if (observer.value) {
            observer.value.disconnect();
          }

          // Observe each match element
          matchElements.value.forEach((match) => {
            observer.value?.observe(match);
          });
        }
      },
      { immediate: false },
    );

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
          toastManager.error("Failed to copy:" + err);
          console.error("Failed to copy:", err);
        });
    };

    // Reset match index when content changes
    watch(
      () => [props.fileContent, props.query],
      () => {
        currentMatchIndex.value = 0; // Reset to 0 when content or query changes

        // Clean up observer when content changes
        if (observer.value) {
          observer.value.disconnect();
          observer.value = null;
        }
        visibleMatches.value.clear();
        matchElements.value = [];
      },
    );

    // Cleanup function to disconnect observer when component unmounts
    onUnmounted(() => {
      if (observer.value) {
        observer.value.disconnect();
        observer.value = null;
      }
      visibleMatches.value.clear();
      matchElements.value = [];
    });

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

    // Navigation for highlighted matches using Intersection Observer
    // Function to calculate all match positions with better precision
    const getAllMatchPositions = () => {
      if (!codeContainerRef.value) return [];

      // Query for matches directly from the DOM to ensure we have current elements
      const matches =
        codeContainerRef.value.querySelectorAll(".highlight-match");
      // Update matchElements for consistency
      matchElements.value = Array.from(matches);

      const positions: { element: Element; index: number; position: number }[] =
        [];

      matchElements.value.forEach((element, i) => {
        const rect = element.getBoundingClientRect();
        const containerRect = codeContainerRef.value!.getBoundingClientRect();
        // Calculate position relative to the scrollable container
        const position =
          rect.top - containerRect.top + codeContainerRef.value!.scrollTop;
        positions.push({ element, index: i, position });
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

          // If we already have a current match, go to the next one (with wraparound)
          if (
            currentMatchIndex.value > 0 &&
            currentMatchIndex.value < matchPositions.length
          ) {
            nextIndex = currentMatchIndex.value; // Go to next match in sequence (0-indexed)
          } else if (currentMatchIndex.value === matchPositions.length) {
            // If we're at the last match, wrap to first (index 0)
            nextIndex = 0;
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
            // Go to the previous match in the sequence (0-indexed)
            prevIndex = currentMatchIndex.value - 2;
          } else {
            // If we're at the first match or haven't started, wrap to the last match
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

    // Function to open the file's containing folder in the system file manager
    const openFileLocation = async () => {
      try {
        if (!props.filePath) {
          console.warn("No file path provided to openFileLocation");
          return;
        }

        await ShowInFolder(props.filePath);
        console.log("Successfully opened file location:", props.filePath);
      } catch (error: any) {
        console.error("Failed to open file location:", error);
        // Show user feedback
        const errorMessage = error.message || "Operation failed";
        console.error(`Could not open file location: ${errorMessage}`);
        toastManager.error(`Could not open file location: ${errorMessage}`);
      }
    };

    // Tree view filtering and navigation methods
    const clearTreeFilter = () => {
      treeFilter.value = "";
    };

    const handleFileClick = (filePath: string) => {
      // Handle clicking on a file in the tree - for now, just emit an event
      // In a future enhancement, this could open the file in a new tab or window
      console.log("Clicked on file:", filePath);
    };

    // Expand all tree items
    const expandAllTreeItems = () => {
      shouldExpandAll.value = true;
      // Reset all individual overrides by updating the tree structure
      resetTreeExpansion();
      // Force a full re-render by updating the key
      treeRefreshKey.value += 1;
    };

    // Collapse all tree items
    const collapseAllTreeItems = () => {
      shouldExpandAll.value = false;
      // Reset all individual overrides by updating the tree structure
      resetTreeExpansion();
      // Force a full re-render by updating the key
      treeRefreshKey.value += 1;
    };

    // Reset individual expansion overrides
    const resetTreeExpansion = () => {
      const resetRecursive = (item: TreeItem) => {
        if (item.children) {
          item.children.forEach((child) => {
            child.isExpanded = shouldExpandAll.value; // Set to the global state
            resetRecursive(child);
          });
        }
      };
      resetRecursive(treeData.value);
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
      isReady,
      visibleMatches,
      matchElements,
      showTreeView,
      activeTab,
      treeData,
      openFileLocation,
      treeFilter,
      isTreeExpanded,
      shouldExpandAll,
      treeRefreshKey,
      clearTreeFilter,
      handleFileClick,
      expandAllTreeItems,
      collapseAllTreeItems,
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

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #555;
  background-color: #2d2d2d;
}

.modal-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tree-view-button {
  background: none;
  border: 1px solid #555;
  color: #ccc;
  padding: 6px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tree-view-button:hover {
  background-color: #555;
  color: #fff;
}

.tree-view-button.active {
  background-color: #6c757d;
  color: #fff;
}

.tab-navigation {
  display: flex;
  border-bottom: 1px solid #555;
  background-color: #333;
  padding: 0 16px;
}

.tab-button {
  padding: 8px 16px;
  border: none;
  background: none;
  color: #ccc;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-button.active {
  color: #fff;
  border-bottom: 2px solid #4caf50;
  font-weight: bold;
}

.tab-button:hover {
  color: #fff;
  background-color: #444;
}

.tree-view-container {
  padding: 16px;
  height: 100%;
  overflow: auto;
  background-color: #333;
}

.tree-view-content h4 {
  margin: 0 0 16px 0;
  color: #fff;
  font-size: 16px;
  border-bottom: 1px solid #555;
  padding-bottom: 8px;
}

.tree-structure {
  padding-left: 16px;
}

.loading {
  padding: 20px;
  color: #fff;
  font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
  font-size: 14px;
  line-height: 1.4;
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

.explorer-button {
  background-color: #2196f3;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
  margin-right: 8px;
}

.explorer-button:hover {
  background-color: #1976d2;
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

/* Tree View Filtering Styles */
.tree-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  gap: 10px;
}

.tree-controls h4 {
  margin: 0;
  flex: 1;
}

.search-bar {
  display: flex;
  align-items: center;
  position: relative;
}

.filter-input {
  padding: 6px 28px 6px 8px; /* Extra padding on right for the clear button */
  border: 1px solid #444;
  border-radius: 4px;
  background-color: #222;
  color: white;
  font-size: 14px;
  width: 200px;
}

.filter-input:focus {
  outline: none;
  border-color: #2196f3;
}

.clear-filter-btn {
  position: absolute;
  right: 6px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: #999;
  cursor: pointer;
  font-size: 16px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.clear-filter-btn:hover {
  background-color: #444;
  color: white;
}

.tree-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #333;
}

.tree-actions button {
  padding: 6px 12px;
  background-color: #555;
  color: white;
  border: 1px solid #666;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}

.tree-actions button:hover {
  background-color: #666;
}

.tree-structure {
  max-height: 500px;
  overflow-y: auto;
  background-color: #2c2c2c;
  border-radius: 4px;
  padding: 8px;
}

.tree-structure::-webkit-scrollbar {
  width: 8px;
}

.tree-structure::-webkit-scrollbar-track {
  background: #222;
}

.tree-structure::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.tree-structure::-webkit-scrollbar-thumb:hover {
  background: #666;
}
</style>
