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
          ><code ref="codeBlock" :key="filePath" v-html="highlightedCode"></code></pre>
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
          <div class="navigation-controls">
            <input
              v-model.number="targetLine"
              type="number"
              min="1"
              :max="totalLines"
              class="line-input"
              placeholder="Line #"
              title="Jump to line"
              @keyup.enter="jumpToLine()"
            />
            <button
              class="jump-button"
              @click="jumpToLine()"
              title="Jump to line"
            >
              Go
            </button>
          </div>
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

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { ShowInFolder } from "../../../wailsjs/go/main/App";
import EnhancedTreeItem from "./EnhancedTreeItem.vue";
import { toastManager } from "../../composables/useToast";
import { useCodeHighlighting } from "../../composables/useCodeHighlighting";
import { useMatchNavigation } from "../../composables/useMatchNavigation";
import type { TreeItem } from "../../types/search";

interface Props {
  isVisible: boolean;
  filePath: string;
  fileContent: string;
  query?: string;
}
const props = withDefaults(defineProps<Props>(), {
  query: "",
});
const emit = defineEmits<{
  close: [];
  copy: [];
}>();

const codeBlock = ref<HTMLElement | null>(null);
const codeContainerRef = ref<HTMLElement | null>(null);

const fileContentFn = () => props.fileContent;
const filePathFn = () => props.filePath;
const queryFn = () => props.query || "";

const {
  highlightedCodeRef,
  isReady,
  detectedLanguage,
  loadAndHighlight,
} = useCodeHighlighting(fileContentFn, filePathFn, queryFn);

const {
  currentMatchIndex,
  totalMatches: totalMatchesFn,
  refreshMatchObserver,
  goToNextMatch,
  goToPreviousMatch,
} = useMatchNavigation(
  () => codeContainerRef.value,
  fileContentFn,
  queryFn,
);

const totalMatches = computed(() => totalMatchesFn());

const highlightedCode = computed(() => highlightedCodeRef.value);

const totalLines = computed(() => {
  if (!props.fileContent) return 0;
  return props.fileContent.split("\n").length;
});

const copied = ref(false);
const targetLine = ref<number | null>(null);

const showTreeView = ref(false);
const activeTab = ref("file");
const treeFilter = ref("");
const shouldExpandAll = ref(false);
const treeRefreshKey = ref(0);

const treeData = ref<TreeItem>({
  name: "",
  path: "",
  children: [],
  isExpanded: true,
});

const closeModal = () => {
  emit("close");
};

const generateTreeStructure = (filePath: string): TreeItem => {
  if (!filePath) return { name: "", path: "", children: [], isExpanded: true };
  const normalizedPath = filePath.replace(/\\/g, "/");
  const pathParts = normalizedPath.split("/").filter((part) => part !== "");
  const rootName = pathParts[0] || "root";
  const root: TreeItem = {
    name: rootName,
    path: rootName,
    children: [],
    isExpanded: true,
  };
  let currentLevel: TreeItem[] = root.children;
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

watch(
  () => props.filePath,
  (newPath) => {
    if (newPath) {
      treeData.value = generateTreeStructure(newPath);
    }
  },
  { immediate: true },
);

// Initialize highlighting when component is set up
(async () => {
  await loadAndHighlight();
})();

// Set up observer after content is rendered
watch(
  [isReady, highlightedCodeRef],
  async ([ready]) => {
    if (ready) {
      await refreshMatchObserver();
    }
  },
  { immediate: false },
);

const copyToClipboard = () => {
  navigator.clipboard
    .writeText(props.fileContent)
    .then(() => {
      copied.value = true;
      setTimeout(() => {
        copied.value = false;
      }, 2000);
      emit("copy");
    })
    .catch((err) => {
      toastManager.error("Failed to copy:" + err);
      console.error("Failed to copy:", err);
    });
};

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === "Escape" && props.isVisible) {
    closeModal();
  }
};

onMounted(() => {
  document.addEventListener("keydown", handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener("keydown", handleKeydown);
});

const scrollToLine = (lineNumber: number) => {
  if (!codeContainerRef.value) return;
  const lineElement = codeContainerRef.value.querySelector(
    `[data-line="${lineNumber}"]`,
  );
  if (lineElement) {
    lineElement.scrollIntoView({ behavior: "smooth", block: "center" });
    lineElement.classList.add("highlighted-line");
    setTimeout(() => {
      if (lineElement) {
        lineElement.classList.remove("highlighted-line");
      }
    }, 1500);
  }
};

const jumpToLine = (lineNumber?: number) => {
  const line = lineNumber ?? targetLine.value ?? 0;
  if (line > 0 && line <= totalLines.value) {
    scrollToLine(line);
  }
};

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
    const errorMessage = error.message || "Operation failed";
    console.error(`Could not open file location: ${errorMessage}`);
    toastManager.error(`Could not open file location: ${errorMessage}`);
  }
};

const clearTreeFilter = () => {
  treeFilter.value = "";
};

const handleFileClick = (filePath: string) => {
  console.log("Clicked on file:", filePath);
};

const expandAllTreeItems = () => {
  shouldExpandAll.value = true;
  resetTreeExpansion();
  treeRefreshKey.value += 1;
};

const collapseAllTreeItems = () => {
  shouldExpandAll.value = false;
  resetTreeExpansion();
  treeRefreshKey.value += 1;
};

const resetTreeExpansion = () => {
  const resetRecursive = (item: TreeItem) => {
    if (item.children) {
      item.children.forEach((child) => {
        child.isExpanded = shouldExpandAll.value;
        resetRecursive(child);
      });
    }
  };
  resetRecursive(treeData.value);
};
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

.navigation-controls {
  display: flex;
  align-items: center;
  gap: 6px;
}

.line-input {
  width: 80px;
  height: 32px;
  padding: 0 8px;
  border: 1px solid #555;
  border-radius: 4px;
  background-color: #2d2d2d;
  color: #fff;
  font-size: 14px;
}

.line-input:focus {
  outline: none;
  border-color: #4caf50;
}

.jump-button {
  background-color: #6c757d;
  color: white;
  border: none;
  height: 32px;
  padding: 0 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.jump-button:hover {
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
