<template>
  <div class="search-controls">
    <!-- Editor detection status display -->
    <div
      v-if="data.editorDetectionStatus.detectingEditors"
      class="editor-detection-status"
    >
      <div class="detection-animation">
        <div class="spinner"></div>
        <span>{{ data.editorDetectionStatus.message }}</span>
      </div>
      <div class="detection-progress">
        <div class="progress-bar">
          <div
            class="progress-fill"
            :style="{ width: data.editorDetectionStatus.detectionProgress + '%' }"
          ></div>
        </div>
        <span class="progress-text">
          {{ Math.round(data.editorDetectionStatus.detectionProgress) }}%
        </span>
      </div>
    </div>

    <!-- Editor detection complete message -->
    <div
      v-else-if="data.editorDetectionStatus.detectionComplete"
      class="editor-detection-status completed"
    >
      <div class="detection-result">
        <span class="status-icon">✓</span>
        <span>{{ data.editorDetectionStatus.message }}</span>
      </div>
      <div
        v-if="data.editorDetectionStatus.detectedEditors.length > 0"
        class="detected-editors-list"
      >
        <span>
          Found editors: {{ data.editorDetectionStatus.detectedEditors.join(", ") }}
        </span>
      </div>
    </div>

    <div class="control-group">
      <label for="directory">Directory:</label>
      <div class="directory-input">
        <input
          id="directory"
          v-model="data.directory"
          class="input directory"
          type="text"
          placeholder="Enter directory to search"
          :disabled="data.isSearching"
        />
        <button
          class="btn select-dir"
          :disabled="data.isSearching"
          @click="selectDirectory"
        >
          Browse
        </button>
      </div>
    </div>

    <div class="control-group" style="width: auto">
      <label for="query">Search Query:</label>
      <input
        id="query"
        style="width: 100%; height: 1.5rem; padding: 2px"
        v-model="data.query"
        class="input"
        type="text"
        placeholder="Enter search term"
        @keyup.enter="searchCode"
        :disabled="data.isSearching"
      />
    </div>

    <!-- Search Options Group -->
    <div class="options-group">
      <div class="control-group checkbox-group">
        <input
          id="case-sensitive"
          v-model="data.caseSensitive"
          type="checkbox"
          :disabled="data.isSearching"
        />
        <label for="case-sensitive">Case Sensitive</label>
      </div>

      <div class="control-group checkbox-group">
        <input
          id="regex-search"
          v-model="data.useRegex"
          type="checkbox"
          :disabled="data.isSearching"
        />
        <label for="regex-search">Regex Search</label>
      </div>

      <div class="control-group checkbox-group">
        <input
          id="include-binary"
          v-model="data.includeBinary"
          type="checkbox"
          :disabled="data.isSearching"
        />
        <label for="include-binary">Include Binary</label>
      </div>

      <div class="control-group checkbox-group">
        <input
          id="search-subdirs"
          v-model="data.searchSubdirs"
          type="checkbox"
          :disabled="data.isSearching"
        />
        <label for="search-subdirs">Search Subdirs</label>
      </div>
    </div>

    <!-- File Size and Results Limit Options Group -->
    <div class="options-group">
      <div class="control-group">
        <label for="min-filesize">Min File Size (bytes):</label>
        <input
          id="min-filesize"
          v-model.number="data.minFileSize"
          class="input"
          type="number"
          placeholder="0"
          :disabled="data.isSearching"
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
          :disabled="data.isSearching"
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
          :disabled="data.isSearching"
        />
      </div>
    </div>

    <!-- Exclude Patterns Section -->
    <div class="control-group">
      <label for="exclude-patterns">Exclude Patterns:</label>
      <div class="exclude-patterns-container">
        <!-- Display selected exclude patterns -->
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
              :disabled="data.isSearching"
            >
              ×
            </button>
          </span>
        </div>

        <!-- Input for adding new exclude patterns -->
        <div class="exclude-patterns-input-wrapper">
          <select
            id="exclude-patterns"
            class="input exclude-select"
            @change="addPatternFromSelect"
            :disabled="data.isSearching"
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

          <div>
            <form
              @submit.prevent="addCustomPattern"
              class="custom-pattern-input"
            >
              <input
                v-model="customPattern"
                class="input"
                type="text"
                placeholder="Or add custom pattern..."
                required
                :disabled="data.isSearching"
              />
              <button
                type="submit"
                class="add-custom-pattern"
                :disabled="data.isSearching"
              >
                Add
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>

    <!-- Allowed File Types Selection -->
    <div class="control-group">
      <label for="allowed-filetypes">
        Allowed File Types (optional - acts as allow-list):
      </label>
      <div class="allowed-filetypes-container">
        <!-- Display selected allowed file types -->
        <div class="allowed-filetypes-selected">
          <span
            v-for="(type, index) in selectedAllowedTypes"
            :key="index"
            class="allowed-type-tag"
          >
            {{ type }}
            <button
              type="button"
              class="remove-allowed-type"
              @click="removeAllowedType(index)"
              :aria-label="`Remove ${type} allowed type`"
              :disabled="data.isSearching"
            >
              ×
            </button>
          </span>
        </div>

        <!-- Input for adding new allowed file types -->
        <div class="allowed-filetypes-input-wrapper">
          <select
            id="allowed-filetypes"
            class="input allowed-select"
            @change="addAllowedTypeFromSelect"
            :disabled="data.isSearching"
          >
            <option value="">Add common type...</option>
            <option value="js">js</option>
            <option value="ts">ts</option>
            <option value="jsx">jsx</option>
            <option value="tsx">tsx</option>
            <option value="go">go</option>
            <option value="py">py</option>
            <option value="java">java</option>
            <option value="cpp">cpp</option>
            <option value="c">c</option>
            <option value="h">h</option>
            <option value="cs">cs</option>
            <option value="rb">rb</option>
            <option value="php">php</option>
            <option value="html">html</option>
            <option value="css">css</option>
            <option value="json">json</option>
            <option value="yaml">yaml</option>
            <option value="yml">yml</option>
            <option value="xml">xml</option>
            <option value="md">md</option>
            <option value="min.js">min.js</option>
            <option value="tar.gz">tar.gz</option>
            <option value="backup.txt">backup.txt</option>
          </select>

          <div>
            <form
              @submit.prevent="addCustomAllowedType"
              class="custom-allowed-input"
            >
              <input
                v-model="customAllowedType"
                class="input"
                type="text"
                required
                placeholder="Or add custom type (e.g. min.js, tar.gz)..."
                :disabled="data.isSearching"
              />
              <button
                type="submit"
                class="add-custom-allowed-type"
                :disabled="data.isSearching"
              >
                Add
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>

    <!-- Search/Cancel Button -->
    <div class="control-group">
      <!-- Show search button when not searching -->
      <button
        v-if="!data.isSearching"
        class="btn search-btn"
        @click="searchCode"
        :disabled="data.isSearching"
      >
        <span v-if="data.isSearching" class="spinner"></span>
        Search Code
      </button>
      <!-- Show cancel button when searching is active -->
      <button
        v-else
        class="btn cancel-btn"
        @click="cancelSearch"
        :disabled="!data.isSearching"
      >
        Cancel Search
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { SearchState } from "../../types/search";

// Define props with TypeScript
interface Props {
  data: SearchState;
  searchCode: () => Promise<void>;
  selectDirectory: () => Promise<void>;
  cancelSearch: () => Promise<void>;
}
const props = defineProps<Props>();

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
  }
);

// Initialize allowed file types from the data property
const selectedAllowedTypes = ref<string[]>(props.data.allowedFileTypes || []);

// Watch for changes in the data.allowedFileTypes from outside
watch(
  () => props.data.allowedFileTypes,
  (newVal: string | string[] | undefined) => {
    if (Array.isArray(newVal)) {
      selectedAllowedTypes.value = newVal;
    } else {
      selectedAllowedTypes.value = newVal ? [newVal] : [];
    }
  }
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

// Add allowed type from the select dropdown
const addAllowedTypeFromSelect = (event: Event) => {
  const selectElement = event.target as HTMLSelectElement;
  const type = selectElement.value;

  if (type && !selectedAllowedTypes.value.includes(type)) {
    selectedAllowedTypes.value.push(type);
    updateAllowedFileTypes();
  }

  // Reset select to placeholder option
  selectElement.selectedIndex = 0;
};

// Remove allowed type by index
const removeAllowedType = (index: number) => {
  selectedAllowedTypes.value.splice(index, 1);
  updateAllowedFileTypes();
};

// Add custom allowed type
const customAllowedType = ref("");
const addCustomAllowedType = () => {
  if (
    customAllowedType.value &&
    !selectedAllowedTypes.value.includes(customAllowedType.value)
  ) {
    selectedAllowedTypes.value.push(customAllowedType.value);
    updateAllowedFileTypes();
    customAllowedType.value = ""; // Clear the input
  }
};

// Update the parent data property to reflect selected allowed types
const updateAllowedFileTypes = () => {
  props.data.allowedFileTypes = selectedAllowedTypes.value;
};
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

.tooltip {
  cursor: help;
  margin-left: 5px;
  color: #7f8c8d;
}

.allowed-filetypes-container {
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 10px;
  background-color: #f8f9fa;
  min-height: 60px;
}

.allowed-filetypes-selected {
  min-height: 25px;
  margin-bottom: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.allowed-type-tag {
  display: inline-flex;
  align-items: center;
  background-color: #2ecc71;
  color: white;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 0.8em;
  margin-right: 5px;
  margin-bottom: 5px;
}

.remove-allowed-type {
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

.remove-allowed-type:hover {
  background-color: rgba(255, 255, 255, 0.2);
}

.allowed-select {
  width: 100%;
  margin-bottom: 10px;
  padding: 5px;
  border: 1px solid #ccc;
  border-radius: 3px;
}

.custom-allowed-input {
  display: flex;
  gap: 5px;
}

.custom-allowed-input input {
  flex: 1;
  padding: 5px;
  border: 1px solid #ccc;
  border-radius: 3px;
}

.add-custom-allowed-type {
  padding: 5px 10px;
  background-color: #27ae60;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
}

.add-custom-allowed-type:hover {
  background-color: #219653;
}

.cancel-btn {
  width: 100%;
  padding: 10px;
  background-color: #e74c3c; /* Red color for cancel button */
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.cancel-btn:hover {
  background-color: #c0392b;
}

.cancel-btn:disabled {
  background-color: #bdc3c7;
  cursor: not-allowed;
}

.editor-detection-status {
  background-color: #f8f9fa;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 10px;
  margin-bottom: 15px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.editor-detection-status.completed {
  background-color: #d4edda;
  border-color: #c3e6cb;
}

.detection-animation {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.9em;
  color: #495057;
}

.detection-result {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.9em;
  color: #155724;
}

.status-icon {
  font-weight: bold;
  color: #28a745;
}

.detection-progress {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  max-width: 400px;
}

.progress-bar {
  flex: 1;
  height: 10px;
  background-color: #e9ecef;
  border-radius: 5px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background-color: #28a745;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.8em;
  color: #6c757d;
  min-width: 40px;
  text-align: right;
}

.detected-editors-list {
  font-size: 0.85em;
  color: #495057;
  margin-top: 5px;
  text-align: center;
  width: 100%;
}

.spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid rgba(0, 0, 0, 0.1);
  border-radius: 50%;
  border-top-color: #28a745;
  animation: spin 1s ease-in-out infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
