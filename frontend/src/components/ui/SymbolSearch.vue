<template>
  <div class="symbol-search">
    <h3>Symbol Search</h3>

    <!-- Search Input -->
    <div class="search-container">
      <input
        type="text"
        v-model="searchQuery"
        placeholder="Enter symbol name to search..."
        @keyup.enter="handleSymbolSearch"
        @focus="($event.target as HTMLInputElement).select()"
        class="symbol-input"
        :disabled="isSearching || isFetchingAll"
      />
      <button
        @click="handleSymbolSearch"
        :disabled="!searchQuery.trim() || isSearching || isFetchingAll"
        class="search-btn"
        title="Search symbols (Ctrl+Enter)"
      >
        Search
      </button>
      <button
        @click="fetchAllSymbols"
        :disabled="isFetchingAll || isSearching"
        class="fetch-all-btn"
        title="Load all symbols from indexed files (Ctrl+K)"
      >
        {{ isFetchingAll ? 'Loading...' : 'Load All Symbols' }}
      </button>
    </div>

    <!-- Status Messages -->
    <div v-if="statusMessage" class="status-message" :class="statusType">
      {{ statusMessage }}
    </div>

    <!-- Symbol Results List -->
    <div v-if="symbolResults.length > 0" class="results-container">
      <div class="results-header">
        Found {{ symbolResults.length }} symbol{{ symbolResults.length !== 1 ? 's' : '' }}
      </div>

      <div
        v-for="(symbol, index) in symbolResults"
        :key="index"
        class="symbol-result"
        :class="{ selected: selectedIndex === index }"
        @click="selectSymbol(symbol)"
        @mouseenter="selectedIndex = index"
      >
        <div class="symbol-header">
          <span class="symbol-name">{{ symbol.name }}</span>
          <span class="symbol-type">{{ symbol.type }}</span>
          <span v-if="symbol.signature" class="symbol-signature">{{ symbol.signature }}</span>
        </div>
        <div class="symbol-location">
          <span class="file-path">{{ formatFilePath(symbol.file) }}</span>
          <span class="line-num">Line {{ symbol.line }}</span>
          <span v-if="symbol.endLine && symbol.endLine !== symbol.line" class="range">
            - {{ symbol.endLine }}
          </span>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else-if="!isSearching && !isFetchingAll && hasSearched" class="empty-state">
      No symbols found matching "{{ searchQuery }}"
    </div>

    <div
      v-else-if="!isFetchingAll && allSymbols.length === 0 && !hasSearched"
      class="empty-state"
    >
      <p>No symbols loaded yet.</p>
      <p class="hint">Click "Load All Symbols" to index symbols from your project files.</p>
    </div>

    <!-- Loading State -->
    <div v-if="isSearching" class="loading-state">
      <div class="spinner"></div>
      <span>Searching symbols...</span>
    </div>

    <div v-if="isFetchingAll" class="loading-state">
      <div class="spinner"></div>
      <span>Loading all symbols... {{ fetchProgress }}%</span>
    </div>

    <!-- Load All Progress (when fetching) -->
    <div v-if="isFetchingAll && showFetchProgress" class="fetch-progress">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: `${fetchProgress}%` }"></div>
      </div>
    </div>

    <!-- Quick Access - Recently Indexed Symbols -->
    <div
      v-if="allSymbols.length > 0 && !isFetchingAll && !isSearching && !hasSearched"
      class="quick-access"
    >
      <h4>Quick Access</h4>
      <div class="quick-items">
        <span
          v-for="symbol in recentlySeenSymbols"
          :key="symbol.name"
          @click="prefillSearchAndNavigate(symbol)"
          class="quick-item"
        >
          {{ symbol.name }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue';
import { GetAllSymbols, SearchSymbols as GoSearchSymbols } from '../../../wailsjs/go/main/App';
import { EventsOn } from '../../../wailsjs/runtime';
import { toastManager } from '../../composables/useToast';
import { formatFilePath } from '../../utils/fileUtils';
import type { SymbolInfo } from '../../types/search';
import { toErrorMessage } from '../../utils/errorUtils';

// The directory to scan for symbols — supplied by the parent (search form's
// selected directory). The Go bindings require it; without a directory the
// backend returns no symbols.
const props = defineProps<{
  directory?: string;
}>();

// Reactive state
const searchQuery = ref('');
const symbolResults = ref<SymbolInfo[]>([]);
const allSymbols = ref<SymbolInfo[]>([]);
const isSearching = ref(false);
const isFetchingAll = ref(false);
const selectedIndex = ref(-1);
const hasSearched = ref(false);
const statusMessage = ref('');
const statusType = ref('');
const fetchProgress = ref(0);
const showFetchProgress = ref(false);

// Computed property for recently seen symbols (last 5 indexed)
const recentlySeenSymbols = computed(() => {
  if (!allSymbols.value.length) return [];
  return allSymbols.value.slice(-5);
});
const handleSymbolSearch = async () => {
  if (!searchQuery.value.trim()) return;

  if (!props.directory) {
    statusMessage.value = 'Select a directory in the search form first';
    statusType.value = 'info';
    return;
  }

  const query = searchQuery.value.trim();
  isSearching.value = true;
  hasSearched.value = true;
  symbolResults.value = [];
  selectedIndex.value = -1;
  statusMessage.value = '';
  statusType.value = '';

  try {
    const results = (await GoSearchSymbols(query, props.directory, 50)) as SymbolInfo[];
    symbolResults.value = results;

    if (results.length === 0) {
      statusMessage.value = `No symbols found matching "${query}"`;
      statusType.value = 'info';
    } else {
      const suffix = results.length === 1 ? '' : 's';
      statusMessage.value = `Found ${results.length} symbol${suffix}`;
      statusType.value = 'success';
    }
  } catch (error: unknown) {
    const msg = toErrorMessage(error, 'Could not search symbols');
    statusMessage.value = `Error searching symbols: ${msg}`;
    statusType.value = 'error';
    toastManager.error(msg, 'Symbol Search Failed');
  } finally {
    isSearching.value = false;
  }
};

// Fetch all symbols from indexed files
const fetchAllSymbols = async () => {
  if (allSymbols.value.length > 0) {
    // If already fetched, just show them
    hasSearched.value = false;
    symbolResults.value = [];
    statusMessage.value = 'All symbols loaded. Start typing to search.';
    statusType.value = 'info';
    return;
  }

  if (!props.directory) {
    statusMessage.value = 'Select a directory in the search form first';
    statusType.value = 'info';
    return;
  }

  isFetchingAll.value = true;
  showFetchProgress.value = true;
  fetchProgress.value = 0;
  symbolResults.value = [];
  hasSearched.value = false;

  // Subscribe to real per-file scan progress emitted by the Go backend
  // (symbol-progress). This replaces the previous synthetic 0->100 jump.
  const stopProgress = EventsOn('symbol-progress', (payload: unknown) => {
    const p = (payload && typeof payload === 'object' ? payload : {}) as Record<string, unknown>;
    const processed = typeof p.processed === 'number' ? p.processed : 0;
    const total = typeof p.total === 'number' ? p.total : 0;
    fetchProgress.value = total > 0 ? Math.round((processed / total) * 100) : 0;
  });

  try {
    // Call GetAllSymbols which processes files under the selected directory.
    const results = (await GetAllSymbols(props.directory, 2000)) as SymbolInfo[];

    allSymbols.value = results;
    fetchProgress.value = 100;

    setTimeout(() => {
      showFetchProgress.value = false;
      fetchProgress.value = 0;
    }, 1000);

    statusMessage.value = `Indexed ${results.length.toLocaleString()} symbols from your project files`;
    statusType.value = 'success';

    toastManager.success(
      `Found ${results.length.toLocaleString()} symbols`,
      'Symbols Indexed'
    );
  } catch (error: unknown) {
    const msg = toErrorMessage(error, 'Could not index symbols');
    statusMessage.value = `Error loading symbols: ${msg}`;
    statusType.value = 'error';
    toastManager.error(msg, 'Failed to Load Symbols');
    showFetchProgress.value = false;
    fetchProgress.value = 0;
  } finally {
    stopProgress();
    isFetchingAll.value = false;
  }
};

// Select a symbol result and show toast with file location
const selectSymbol = (symbol: SymbolInfo) => {
  // Show toast with file location info
  toastManager.info(
    `${symbol.name} at ${formatFilePath(symbol.file)}:${symbol.line}`,
    'Symbol Selected'
  );

  // Emit an event that could be used to navigate to the symbol
  // This could be expanded to open the file in an external editor or internal view
  window.dispatchEvent(new CustomEvent('symbol-selected', { detail: symbol }));
};

// Prefill search and navigate to symbol
const prefillSearchAndNavigate = async (symbol: SymbolInfo) => {
  searchQuery.value = symbol.name;
  hasSearched.value = false;
  symbolResults.value = [];

  // Trigger search for this symbol
  await handleSymbolSearch();
};

// Expose methods for keyboard shortcut integration
defineExpose({
  searchQuery,
  handleSymbolSearch,
  fetchAllSymbols,
  setFocus: () => {
    const input = document.querySelector('.symbol-input') as HTMLInputElement;
    if (input) input.focus();
  },
});
</script>

<style scoped>
.symbol-search {
  padding: 1rem;
  border-bottom: 1px solid #e0e0e0;
  background-color: #fafbfc;
}

h3 {
  margin: 0 0 0.75rem 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: #24292f;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

h3::before {
  content: "#";
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background-color: #0366d6;
  color: white;
  border-radius: 4px;
  font-family: monospace;
  font-size: 1rem;
}

.search-container {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  flex-wrap: wrap;
}

.symbol-input {
  flex: 1;
  min-width: 200px;
  padding: 0.625rem 0.875rem;
  font-size: 0.9rem;
  border: 1px solid #e1e4e8;
  border-radius: 6px;
  background-color: #ffffff;
  color: #24292f;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.symbol-input:focus {
  outline: none;
  border-color: #0366d6;
  box-shadow: 0 0 0 3px rgba(3, 102, 214, 0.3);
}

.symbol-input:disabled {
  background-color: #f6f8fa;
  cursor: not-allowed;
}

.search-btn,
.fetch-all-btn {
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.search-btn {
  background-color: #2da44e;
  color: white;
}

.search-btn:hover:not(:disabled) {
  background-color: #2c974b;
}

.search-btn:disabled {
  background-color: #85909b;
  opacity: 0.6;
  cursor: not-allowed;
}

.fetch-all-btn {
  background-color: #0366d6;
  color: white;
}

.fetch-all-btn:hover:not(:disabled) {
  background-color: #0256be;
}

.fetch-all-btn:disabled {
  background-color: #85909b;
  opacity: 0.6;
  cursor: not-allowed;
}

/* Status Messages */
.status-message {
  padding: 0.75rem 1rem;
  border-radius: 6px;
  font-size: 0.875rem;
  margin-bottom: 0.75rem;
}

.status-message.success {
  background-color: #dafbe1;
  color: #1a7f37;
  border: 1px solid #a2d0a8;
}

.status-message.error {
  background-color: #ffeef0;
  color: #cf222e;
  border: 1px solid #ffa1a7;
}

.status-message.info {
  background-color: #dbedff;
  color: #0969da;
  border: 1px solid #79c0ff;
}

/* Results Container */
.results-container {
  margin-top: 0.75rem;
}

.results-header {
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
  font-weight: 600;
  color: #57606a;
  background-color: #f6f8fa;
  border-radius: 6px;
  margin-bottom: 0.5rem;
}

.symbol-result {
  padding: 0.875rem 1rem;
  margin-bottom: 0.5rem;
  background-color: white;
  border: 1px solid #e1e4e8;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.symbol-result:hover {
  border-color: #0366d6;
  box-shadow: 0 0 0 3px rgba(3, 102, 214, 0.1);
}

.symbol-result.selected {
  border-color: #0366d6;
  background-color: #f1faff;
  box-shadow: 0 0 0 3px rgba(3, 102, 214, 0.2);
}

.symbol-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.375rem;
  flex-wrap: wrap;
}

.symbol-name {
  font-family: SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 1rem;
  font-weight: 600;
  color: #1a7f37;
}

.symbol-type {
  font-size: 0.75rem;
  font-weight: 500;
  color: #656d76;
  background-color: #f6f8fa;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  text-transform: uppercase;
}

.symbol-signature {
  font-family: SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 0.85rem;
  color: #57606a;
  font-style: italic;
}

.symbol-location {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: #656d76;
}

.file-path {
  color: #0969da;
}

.line-num {
  font-family: SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
}

.range {
  color: #656d76;
}

/* Empty States */
.empty-state {
  padding: 2rem 1rem;
  text-align: center;
  color: #656d76;
}

.empty-state p {
  margin: 0.5rem 0;
}

.empty-state .hint {
  font-size: 0.8rem;
  color: #85909b;
  font-style: italic;
}

/* Loading State */
.loading-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.5rem;
  color: #57606a;
  font-size: 0.9rem;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid #e1e4e8;
  border-top-color: #0366d6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Fetch Progress */
.fetch-progress {
  margin-top: 1rem;
}

.progress-bar {
  height: 8px;
  background-color: #e1e4e8;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background-color: #0366d6;
  transition: width 0.3s ease;
}

/* Quick Access */
.quick-access {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px dashed #e1e4e8;
}

.quick-access h4 {
  margin: 0 0 0.75rem 0;
  font-size: 0.8rem;
  font-weight: 600;
  color: #57606a;
}

.quick-items {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.quick-item {
  padding: 0.375rem 0.75rem;
  background-color: #f6f8fa;
  border: 1px solid #e1e4e8;
  border-radius: 6px;
  font-family: SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 0.85rem;
  color: #0969da;
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-item:hover {
  background-color: #dbedff;
  border-color: #0366d6;
}
</style>
