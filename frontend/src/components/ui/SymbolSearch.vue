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

// Select a symbol result: open the code preview modal at the symbol's line
// (via the useFilePreview singleton) and show a toast with the location.
const selectSymbol = (symbol: SymbolInfo) => {
  toastManager.info(
    `${symbol.name} at ${formatFilePath(symbol.file)}:${symbol.line}`,
    'Symbol Selected'
  );

  // Dispatch an event that CodeSearch.vue listens for to open the preview
  // modal at the symbol's file:line with a flash highlight.
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
  border-bottom: 1px solid var(--color-border);
  background-color: var(--color-bg-secondary);
}

h3 {
  margin: 0 0 0.75rem 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--color-text-primary);
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
  background-color: var(--color-accent);
  color: var(--color-text-inverse);
  border-radius: var(--radius-sm);
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
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background-color: var(--color-bg);
  color: var(--color-text-primary);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.symbol-input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(var(--color-accent-rgb), 0.3);
}

.symbol-input:disabled {
  background-color: var(--color-bg-secondary);
  cursor: not-allowed;
}

.search-btn,
.fetch-all-btn {
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.search-btn {
  background-color: var(--color-success);
  color: var(--color-text-inverse);
}

.search-btn:hover:not(:disabled) {
  background-color: var(--color-success);
}

.search-btn:disabled {
  background-color: var(--color-text-muted);
  opacity: 0.6;
  cursor: not-allowed;
}

.fetch-all-btn {
  background-color: var(--color-accent);
  color: var(--color-text-inverse);
}

.fetch-all-btn:hover:not(:disabled) {
  background-color: var(--color-accent-dark);
}

.fetch-all-btn:disabled {
  background-color: var(--color-text-muted);
  opacity: 0.6;
  cursor: not-allowed;
}

/* Status Messages */
.status-message {
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  margin-bottom: 0.75rem;
}

.status-message.success {
  background-color: color-mix(in srgb, var(--color-success) 15%, var(--color-bg));
  color: var(--color-success);
  border: 1px solid color-mix(in srgb, var(--color-success) 15%, var(--color-bg));
}

.status-message.error {
  background-color: color-mix(in srgb, var(--color-danger) 15%, var(--color-bg));
  color: var(--color-danger);
  border: 1px solid color-mix(in srgb, var(--color-danger) 15%, var(--color-bg));
}

.status-message.info {
  background-color: var(--color-accent-light);
  color: var(--color-accent);
  border: 1px solid var(--color-accent-light);
}

/* Results Container */
.results-container {
  margin-top: 0.75rem;
}

.results-header {
  padding: 0.5rem 0.75rem;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  background-color: var(--color-bg-secondary);
  border-radius: var(--radius-md);
  margin-bottom: 0.5rem;
}

.symbol-result {
  padding: 0.875rem 1rem;
  margin-bottom: 0.5rem;
  background-color: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.symbol-result:hover {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(var(--color-accent-rgb), 0.1);
}

.symbol-result.selected {
  border-color: var(--color-accent);
  background-color: var(--color-bg-hover);
  box-shadow: 0 0 0 3px rgba(var(--color-accent-rgb), 0.2);
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
  color: var(--color-success);
}

.symbol-type {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background-color: var(--color-bg-secondary);
  padding: 0.125rem 0.5rem;
  border-radius: var(--radius-sm);
  text-transform: uppercase;
}

.symbol-signature {
  font-family: SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  font-style: italic;
}

.symbol-location {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: var(--color-text-secondary);
}

.file-path {
  color: var(--color-accent);
}

.line-num {
  font-family: SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
}

.range {
  color: var(--color-text-secondary);
}

/* Empty States */
.empty-state {
  padding: 2rem 1rem;
  text-align: center;
  color: var(--color-text-secondary);
}

.empty-state p {
  margin: 0.5rem 0;
}

.empty-state .hint {
  font-size: 0.8rem;
  color: var(--color-text-muted);
  font-style: italic;
}

/* Loading State */
.loading-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1.5rem;
  color: var(--color-text-secondary);
  font-size: 0.9rem;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-accent);
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
  background-color: var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background-color: var(--color-accent);
  transition: width 0.3s ease;
}

/* Quick Access */
.quick-access {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px dashed var(--color-border);
}

.quick-access h4 {
  margin: 0 0 0.75rem 0;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.quick-items {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.quick-item {
  padding: 0.375rem 0.75rem;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-family: SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 0.85rem;
  color: var(--color-accent);
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-item:hover {
  background-color: var(--color-accent-light);
  border-color: var(--color-accent);
}
</style>
