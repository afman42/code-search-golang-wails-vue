<template>
  <div class="search-controls">
    <div class="control-group">
      <label for="directory">Directory:</label>
      <div class="directory-input">
        <input
          id="directory"
          v-model="data.directory"
          class="input directory"
          type="text"
          placeholder="Enter directory to search"
        />
        <button class="btn select-dir" @click="selectDirectory">Browse</button>
      </div>
    </div>
    <div class="options-group" style="width: auto">
      <div class="control-group" style="width: 65%">
        <label for="query">Search Query:</label>
        <input
          id="query"
          style="width: 100%; height: 1.5rem; padding: 2px"
          v-model="data.query"
          class="input"
          type="text"
          placeholder="Enter search term"
          @keyup.enter="searchCode"
        />
      </div>

      <div class="control-group" style="width: 30%">
        <label for="extension" style="font-size: 0.9rem; margin-bottom: 0.5rem"
          >File Extension (optional):</label
        >
        <input
          id="extension"
          style="width: 100%; height: 1.5rem; padding: 2px"
          v-model="data.extension"
          class="input"
          type="text"
          placeholder="e.g., go, js, ts"
        />
      </div>
    </div>

    <div class="options-group">
      <div class="control-group checkbox-group">
        <input
          id="case-sensitive"
          v-model="data.caseSensitive"
          type="checkbox"
        />
        <label for="case-sensitive">Case Sensitive</label>
      </div>

      <div class="control-group checkbox-group">
        <input id="regex-search" v-model="data.useRegex" type="checkbox" />
        <label for="regex-search">Regex Search</label>
      </div>

      <div class="control-group checkbox-group">
        <input
          id="include-binary"
          v-model="data.includeBinary"
          type="checkbox"
        />
        <label for="include-binary">Include Binary</label>
      </div>

      <div class="control-group checkbox-group">
        <input
          id="search-subdirs"
          v-model="data.searchSubdirs"
          type="checkbox"
        />
        <label for="search-subdirs">Search Subdirs</label>
      </div>
    </div>

    <div class="options-group">
      <div class="control-group">
        <label for="min-filesize">Min File Size (bytes):</label>
        <input
          id="min-filesize"
          v-model.number="data.minFileSize"
          class="input"
          type="number"
          placeholder="0"
        />
      </div>

      <div class="control-group">
        <label for="max-filesize">Max File Size (bytes):</label>
        <input
          id="max-filesize"
          v-model.number="data.maxFileSize"
          class="input"
          type="number"
          placeholder="10485760 (10MB)"
        />
      </div>

      <div class="control-group">
        <label for="max-results">Max Results:</label>
        <input
          id="max-results"
          v-model.number="data.maxResults"
          class="input"
          type="number"
          placeholder="1000"
        />
      </div>
    </div>

    <div class="control-group">
      <label for="exclude-patterns">Exclude Patterns:</label>
      <div class="exclude-patterns-container">
        <div class="exclude-patterns-selected">
          <span
            v-for="(pattern, index) in selectedPatterns"
            :key="index"
            class="pattern-tag"
          >
            {{ pattern }}
            <button
              type="button"
              class="remove-pattern"
              @click="removePattern(index)"
              :aria-label="`Remove ${pattern} pattern`"
            >
              ×
            </button>
          </span>
        </div>

        <div class="exclude-patterns-input-wrapper">
          <select
            id="exclude-patterns"
            class="input exclude-select"
            @change="addPatternFromSelect"
          >
            <option value="">Add common pattern...</option>
            <option value="node_modules">node_modules</option>
            <option value=".git">.git</option>
            <option value=".svn">.svn</option>
            <option value=".hg">.hg</option>
            <option value="build">build</option>
            <option value="dist">dist</option>
            <option value="target">target</option>
            <option value=".DS_Store">.DS_Store</option>
            <option value=".idea">.idea</option>
            <option value=".vscode">.vscode</option>
            <option value="*.log">*.log</option>
            <option value="*.tmp">*.tmp</option>
            <option value="*.temp">*.temp</option>
          </select>

          <div class="custom-pattern-input">
            <input
              v-model="customPattern"
              class="input"
              type="text"
              placeholder="Or add custom pattern..."
            />
            <button
              type="button"
              class="add-custom-pattern"
              @click="addCustomPattern"
            >
              Add
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="control-group">
      <button
        class="btn search-btn"
        @click="searchCode"
        :disabled="data.isSearching"
      >
        <span v-if="data.isSearching" class="spinner"></span>
        {{ data.isSearching ? "Searching..." : "Search Code" }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, watch } from "vue";
import type { PropType } from "vue";
import { SearchState } from "../../types/search";

export default defineComponent({
  name: "SearchForm",
  props: {
    data: {
      type: Object as () => SearchState,
      required: true,
    },
    searchCode: {
      type: Function as PropType<() => Promise<void>>,
      required: true,
    },
    selectDirectory: {
      type: Function as PropType<() => Promise<void>>,
      required: true,
    },
  },
  setup(props) {
    // Initialize selected patterns from the data property
    const selectedPatterns = ref<string[]>(props.data.excludePatterns || []);

    // Watch for changes in the data.excludePatterns from outside (e.g., when it's updated by composable)
    watch(
      () => props.data.excludePatterns,
      (newVal: string | string[] | undefined) => {
        if (Array.isArray(newVal)) {
          selectedPatterns.value = newVal;
        } else if (typeof newVal === "string") {
          // newVal is string from old format or string type, convert it
          selectedPatterns.value = newVal
            ? newVal
                .split(",")
                .map((s: string) => s.trim())
                .filter((s: string) => s.length > 0)
            : [];
        } else {
          selectedPatterns.value = [];
        }
      },
    );

    // Add pattern from the select dropdown
    const addPatternFromSelect = (event: Event) => {
      const selectElement = event.target as HTMLSelectElement;
      const pattern = selectElement.value;

      if (pattern && !selectedPatterns.value.includes(pattern)) {
        selectedPatterns.value.push(pattern);
        updateExcludePatterns();
      }

      // Reset select to placeholder option
      selectElement.selectedIndex = 0;
    };

    // Remove pattern by index
    const removePattern = (index: number) => {
      selectedPatterns.value.splice(index, 1);
      updateExcludePatterns();
    };

    // Add custom pattern
    const customPattern = ref("");
    const addCustomPattern = () => {
      if (
        customPattern.value &&
        !selectedPatterns.value.includes(customPattern.value)
      ) {
        selectedPatterns.value.push(customPattern.value);
        updateExcludePatterns();
        customPattern.value = ""; // Clear the input
      }
    };

    // Update the parent data property to reflect selected patterns
    const updateExcludePatterns = () => {
      props.data.excludePatterns = selectedPatterns.value;
    };

    return {
      selectedPatterns,
      addPatternFromSelect,
      removePattern,
      customPattern,
      addCustomPattern,
    };
  },
});
</script>

<style scoped>
.search-controls {
  max-width: 600px;
  margin: 0 auto;
  padding: 20px;
}

.search-controls .control-group {
  margin-bottom: 15px;
}

.search-controls .options-group {
  display: flex;
  gap: 15px;
  margin-bottom: 15px;
  flex-wrap: wrap;
}

.search-controls label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
}

.directory-input {
  display: flex;
  gap: 10px;
}

.directory-input .input {
  flex: 1;
}

.select-dir {
  width: auto;
  padding: 0 15px;
}

.checkbox-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.checkbox-group label {
  margin-bottom: 0;
  font-weight: normal;
}

.search-btn {
  width: 100%;
  padding: 10px;
  background-color: #27ae60;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.search-btn:hover {
  background-color: #219653;
}

.search-btn:disabled {
  background-color: #bdc3c7;
  cursor: not-allowed;
}

.spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-radius: 50%;
  border-top-color: #fff;
  animation: spin 1s ease-in-out infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.select-dir:hover {
  background-image: linear-gradient(to top, #cfd9df 0%, #e2ebf0 100%);
  color: #333333;
}

.directory-input .input {
  border: none;
  border-radius: 3px;
  outline: none;
  height: 30px;
  line-height: 30px;
  padding: 0 10px;
  background-color: rgba(240, 240, 240, 1);
  -webkit-font-smoothing: antialiased;
}

.directory-input .input:hover {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}

.directory-input .input:focus {
  border: none;
  background-color: rgba(255, 255, 255, 1);
}

.exclude-patterns-container {
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 10px;
  background-color: #fafafa;
  min-height: 60px;
}

.exclude-patterns-selected {
  min-height: 25px;
  margin-bottom: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.pattern-tag {
  display: inline-flex;
  align-items: center;
  background-color: #3498db;
  color: white;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 0.8em;
  margin-right: 5px;
  margin-bottom: 5px;
}

.remove-pattern {
  background: none;
  border: none;
  color: white;
  margin-left: 6px;
  cursor: pointer;
  font-weight: bold;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.remove-pattern:hover {
  background-color: rgba(255, 255, 255, 0.2);
}

.exclude-select {
  width: 100%;
  margin-bottom: 10px;
  padding: 5px;
  border: 1px solid #ccc;
  border-radius: 3px;
}

.custom-pattern-input {
  display: flex;
  gap: 5px;
}

.custom-pattern-input input {
  flex: 1;
  padding: 5px;
  border: 1px solid #ccc;
  border-radius: 3px;
}

.add-custom-pattern {
  padding: 5px 10px;
  background-color: #95a5a6;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
}

.add-custom-pattern:hover {
  background-color: #7f8c8d;
}
</style>
