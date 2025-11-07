<template>
  <div class="tree-item">
    <div
      class="tree-item-header"
      :class="{ 'is-file': item.isFile, 'is-folder': !item.isFile }"
      @click="toggleExpand"
    >
      <span
        class="tree-item-toggle"
        v-if="!item.isFile && item.children && item.children.length > 0"
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
        :class="{ 'current-file': currentFilePath === item.path }"
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
          v-else-if="!item.isFile && item.children && item.children.length > 0"
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
      </span>
    </div>

    <div
      class="tree-item-children"
      v-if="
        !item.isFile && item.children && item.children.length > 0 && isExpanded
      "
    >
      <TreeItem
        v-for="child in item.children"
        :key="child.path || child.name"
        :item="child"
        :current-file-path="currentFilePath"
        :expanded="expanded"
      />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, PropType } from "vue";

interface TreeItem {
  name: string;
  path: string;
  children?: TreeItem[];
  isFile?: boolean;
  isExpanded?: boolean;
}

export default defineComponent({
  name: "TreeItem",
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
  },
  setup(props) {
    const isExpanded = ref(props.expanded || false);

    const toggleExpand = () => {
      if (
        !props.item.isFile &&
        props.item.children &&
        props.item.children.length > 0
      ) {
        isExpanded.value = !isExpanded.value;
      }
    };

    return {
      isExpanded,
      toggleExpand,
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
}

.tree-item-toggle.rotated {
  transform: rotate(90deg);
}

.tree-item-name {
  display: flex;
  align-items: center;
  gap: 4px;
}

.icon {
  flex-shrink: 0;
  color: white;
}

.tree-item-children {
  padding-left: 20px;
  margin-top: 2px;
}
</style>
