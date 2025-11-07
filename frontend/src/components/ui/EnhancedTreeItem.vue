<template>
  <div class="tree-item">
    <div
      class="tree-item-header"
      :class="{ 'is-file': item.isFile, 'is-folder': !item.isFile, 'current-file': item.path === currentFilePath }"
      @click="onItemClick"
    >
      <span
        class="tree-item-toggle"
        v-if="!item.isFile && hasChildren"
        @click.stop="toggleExpand"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          :class="{ rotated: isExpanded }"
        >
          <polyline points="6 9 12 15 18 9"></polyline>
        </svg>
      </span>
      <span
        class="tree-item-name"
        :class="{ 'current-file': item.path === currentFilePath }"
      >
        <svg
          v-if="item.isFile"
          class="icon"
          xmlns="http://www.w3.org/2000/svg"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path
            d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"
          ></path>
          <polyline points="14 2 14 8 20 8"></polyline>
        </svg>
        <svg
          v-else-if="!item.isFile && hasChildren"
          class="icon"
          xmlns="http://www.w3.org/2000/svg"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path
            d="M13 10H2v8a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V7.5L17.5 2H13z"
          ></path>
        </svg>
        <svg
          v-else
          class="icon"
          xmlns="http://www.w3.org/2000/svg"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path
            d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2z"
          ></path>
        </svg>
        {{ item.name }}
        <span v-if="showItemCount && item.children && !item.isFile" class="item-count">
          ({{ visibleChildCount }})
        </span>
      </span>
    </div>

    <div
      class="tree-item-children"
      v-if="!item.isFile && hasChildren && isExpanded"
    >
      <div v-if="filterText && filteredChildren.length === 0" class="no-results">
        No matching items
      </div>
      <EnhancedTreeItem
        v-for="child in filteredChildren"
        :key="child.path || child.name"
        :item="child"
        :current-file-path="currentFilePath"
        :expanded="expanded"
        :filter-text="filterText"
        :show-item-count="showItemCount"
        @file-click="$emit('file-click', $event)"
      />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, PropType, watch } from "vue";

interface TreeItem {
  name: string;
  path: string;
  children?: TreeItem[];
  isFile?: boolean;
  isExpanded?: boolean;
}

export default defineComponent({
  name: "EnhancedTreeItem",
  props: {
    item: {
      type: Object as PropType<TreeItem>,
      required: true,
    },
    currentFilePath: {
      type: String,
      default: "",
    },
    expanded: {
      type: Boolean,
      default: false,
    },
    filterText: {
      type: String,
      default: "",
    },
    showItemCount: {
      type: Boolean,
      default: true,
    }
  },
  emits: ["file-click"],
  setup(props, { emit }) {
    const localExpanded = ref<boolean>(props.item.isExpanded || false);
    const hasIndividualOverride = ref<boolean>(false);
    // Cache for matching descendants to avoid recomputing
    const descendantMatchCache = new Map<string, boolean>();

    // Check if item has children (considering potential filtering)
    const hasChildren = computed(() => {
      return props.item.children && props.item.children.length > 0;
    });

    // Watch for filter changes to clear the cache
    watch(() => props.filterText, () => {
      descendantMatchCache.clear(); // Clear cache when filter changes
    });

    // Computed property that prioritizes individual expansion state after user interaction
    const effectiveExpanded = computed(() => {
      // If the user has interacted with this specific item, respect their choice
      if (hasIndividualOverride.value) {
        return localExpanded.value;
      }
      // Otherwise, use the global expanded state if it's set
      if (props.expanded !== undefined && props.expanded !== null) {
        return props.expanded;
      }
      // Finally, fall back to the local state or item's isExpanded property
      return localExpanded.value;
    });

    // Filter children based on filter text
    const filteredChildren = computed(() => {
      if (!props.filterText || props.filterText.trim() === '') {
        return props.item.children || [];
      }

      const filter = props.filterText.toLowerCase();
      return (props.item.children || []).filter(child => {
        // Include if name matches
        if (child.name.toLowerCase().includes(filter)) {
          return true;
        }
        
        // Include if it's a folder with matching children
        if (!child.isFile && child.children && child.children.length > 0) {
          return child.children.some(descendant => 
            descendant.name.toLowerCase().includes(filter) ||
            (!descendant.isFile && hasMatchingDescendant(descendant, filter))
          );
        }
        
        return false;
      });
    });

    // Helper function to check if a folder has matching descendants
    const hasMatchingDescendant = (item: TreeItem, filter: string): boolean => {
      // Create a unique cache key for this item and filter combination
      const cacheKey = `${item.path || item.name}-${filter}`;
      
      // Return cached result if available
      if (descendantMatchCache.has(cacheKey)) {
        return descendantMatchCache.get(cacheKey)!;
      }

      if (!item.children || item.children.length === 0) {
        descendantMatchCache.set(cacheKey, false);
        return false;
      }

      let result = false;
      for (const child of item.children) {
        if (child.name.toLowerCase().includes(filter)) {
          result = true;
          break;
        }
        
        if (!child.isFile && hasMatchingDescendant(child, filter)) {
          result = true;
          break;
        }
      }

      // Cache the result
      descendantMatchCache.set(cacheKey, result);
      return result;
    };

    // Count visible children (either filtered or all)
    const visibleChildCount = computed<number>(() => {
      if (props.filterText && props.filterText.trim() !== '') {
        return filteredChildren.value.length;
      }
      return props.item.children ? props.item.children.length : 0;
    });

    const toggleExpand = () => {
      if (!props.item.isFile && hasChildren.value) {
        localExpanded.value = !localExpanded.value;
        hasIndividualOverride.value = true;  // Mark that this node has an individual override
      }
    };

    // Click handler - emit event if it's a file
    const onItemClick = () => {
      if (props.item.isFile) {
        emit('file-click', props.item.path);
      } else if (hasChildren.value) {
        toggleExpand();
      }
    };

    return {
      isExpanded: effectiveExpanded,
      hasChildren,
      filteredChildren,
      visibleChildCount,
      toggleExpand,
      onItemClick,
    };
  },
});
</script>

<style scoped>
.tree-item {
  user-select: none;
}

.tree-item-header {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  cursor: pointer;
  border-radius: 4px;
  margin: 2px 0;
  color: white;
}

.tree-item-header:hover {
  background-color: #444;
}

.tree-item-header.is-file {
  padding-left: 24px;
}

.tree-item-header.current-file {
  background-color: #5a6475;
  font-weight: bold;
}

.tree-item-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  margin-right: 4px;
  transition: transform 0.2s;
  color: white;
  cursor: pointer;
}

.tree-item-toggle.rotated {
  transform: rotate(90deg);
}

.tree-item-name {
  display: flex;
  align-items: center;
  gap: 4px;
}

.item-count {
  color: #aaa;
  font-size: 0.8em;
}

.icon {
  flex-shrink: 0;
  color: white;
}

.tree-item-children {
  padding-left: 20px;
  margin-top: 2px;
}

.no-results {
  padding: 4px 8px;
  color: #aaa;
  font-style: italic;
}
</style>