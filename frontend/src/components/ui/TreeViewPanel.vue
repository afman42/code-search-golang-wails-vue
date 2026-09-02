 <template>
  <div class="tree-view-panel">
    <div class="tree-header">
      <h4>File Explorer</h4>
      <span v-if="fileCount > 0" class="tree-summary">
        {{ fileCount }} file{{ fileCount === 1 ? '' : 's' }} · {{ directoryCount }} folder{{ directoryCount === 1 ? '' : 's' }}
      </span>
    </div>

    <!-- type="search" for the browser's native clear affordance. Uncontrolled
         value + explicit handler so the debounced commit is the only thing
         that drives filterText. -->
    <input
      v-if="fileCount > 0"
      class="tree-filter"
      type="search"
      placeholder="Filter files…"
      aria-label="Filter files"
      @input="onFilterInput"
    />

    <div v-if="visibleRoots.length > 0" class="tree-scroll">
      <EnhancedTreeItem
        v-for="item in visibleRoots"
        :key="item.path || item.name"
        :item="item"
        :current-file-path="currentFilePath || ''"
        :expanded="true"
        :filter-text="filterText"
        :show-item-count="true"
        @file-click="handleFileClick"
      />
    </div>

    <div v-else class="tree-empty">
      <p class="no-selection" v-if="fileCount > 0">
        No files match “{{ filterText }}”.
      </p>
      <p class="no-selection" v-else>No files found for this search.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import EnhancedTreeItem from './EnhancedTreeItem.vue';
import { debounce } from '@/utils';
import type { TreeItem } from '@/types';

interface Props {
  currentFilePath: string | null;
  files?: string[];
}

const props = withDefaults(defineProps<Props>(), {
  currentFilePath: null,
  files: () => [],
});

const emit = defineEmits<{(e: 'fileClick', path: string): void}>();

// Filtering walks the whole tree, so keystrokes are debounced instead of
// re-filtering per character.
const FILTER_DEBOUNCE_MS = 150;

// Trimmed at commit so this value and EnhancedTreeItem's own matching (which
// lowercases but doesn't trim) can't disagree about leading whitespace.
const filterText = ref('');

// `unknown` param: debounce()'s generic is constrained to
// (...args: unknown[]) => void.
const commitFilter = debounce((value: unknown) => {
  filterText.value = typeof value === 'string' ? value.trim() : '';
}, FILTER_DEBOUNCE_MS);

const onFilterInput = (event: Event) => {
  commitFilter((event.target as HTMLInputElement).value);
};

// Handles both '/' (POSIX) and '\\' (Windows) separators.
const SEPARATOR = /[\/\\]+/;

const sortNodes = (nodes: TreeItem[]): void => {
  nodes.sort((a, b) => {
    const aIsDir = !a.isFile;
    const bIsDir = !b.isFile;
    if (aIsDir !== bIsDir) return aIsDir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
  nodes.forEach((node) => sortNodes(node.children));
};

const buildTree = (filePaths: string[]): TreeItem[] => {
  const roots: TreeItem[] = [];
  const nodeByPath = new Map<string, TreeItem>();

  filePaths.forEach((filePath) => {
    if (!filePath || typeof filePath !== 'string') return;

    const parts = filePath.split(SEPARATOR).filter(Boolean);
    if (parts.length === 0) return;

    const absolute = filePath.startsWith('/') || filePath.startsWith('\\');
    let accumulatedPath = '';

    parts.forEach((part, index) => {
      const isFile = index === parts.length - 1;
      const currentPath =
        accumulatedPath === ''
          ? (absolute ? '/' : '') + part
          : `${accumulatedPath}/${part}`;
      accumulatedPath = currentPath;

      let node = nodeByPath.get(currentPath);
      if (!node) {
        node = { name: part, path: currentPath, isFile, children: [] };
        nodeByPath.set(currentPath, node);

        if (index === 0) {
          roots.push(node);
        } else {
          const parentPath = accumulatedPath.slice(
            0,
            accumulatedPath.lastIndexOf('/'),
          );
          const parent = nodeByPath.get(parentPath);
          parent?.children.push(node);
        }
      }
    });
  });

  sortNodes(roots);
  return roots;
};

const tree = computed<TreeItem[]>(() => buildTree(props.files || []));

const countNodes = (nodes: TreeItem[]): { files: number; dirs: number } => {
  let files = 0;
  let dirs = 0;
  const walk = (list: TreeItem[]): void => {
    list.forEach((node) => {
      if (node.isFile) {
        files++;
      } else {
        dirs++;
        walk(node.children);
      }
    });
  };
  walk(nodes);
  return { files, dirs };
};

const fileCount = computed(() => countNodes(tree.value).files);
const directoryCount = computed(() => countNodes(tree.value).dirs);

// EnhancedTreeItem filters its own children but never itself, so the roots are
// filtered here — otherwise a root with no match still renders with an empty
// body. Same rule as the child filter: case-insensitive substring on names.
const nameMatchesAnywhere = (nodes: TreeItem[], needle: string): boolean =>
  nodes.some(
    (node) =>
      node.name.toLowerCase().includes(needle) ||
      nameMatchesAnywhere(node.children, needle),
  );

const visibleRoots = computed<TreeItem[]>(() => {
  const needle = filterText.value.toLowerCase();
  if (needle === '') return tree.value;
  return tree.value.filter(
    (root) =>
      root.name.toLowerCase().includes(needle) ||
      nameMatchesAnywhere(root.children, needle),
  );
});

const handleFileClick = (path: string) => {
  emit('fileClick', path);
};
</script>

<style scoped>
.tree-view-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 0.5rem;
  border-top: 1px solid var(--color-border);
  overflow: hidden;
}

.tree-header {
  margin-bottom: 0.5rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
}

h4 {
  margin: 0;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}

.tree-summary {
  font-size: 0.7rem;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.tree-filter {
  width: 100%;
  margin-bottom: 0.5rem;
  padding: 0.3rem 0.5rem;
  font-size: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg);
  color: var(--color-text-primary);
}

.tree-filter:focus {
  outline: none;
  border-color: var(--color-accent);
}

.tree-scroll {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.tree-scroll::-webkit-scrollbar {
  width: 6px;
}

.tree-scroll::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}

.tree-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
}

.no-selection {
  color: var(--color-text-muted);
  font-size: 0.8rem;
  text-align: center;
  padding: 1rem 0;
  margin: 0;
}
</style>
