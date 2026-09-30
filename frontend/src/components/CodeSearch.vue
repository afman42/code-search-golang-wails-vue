<template>
  <main>
    <a href="#main-content" class="skip-link">Skip to main content</a>

    <div class="app-layout">
      <button
        class="theme-toggle"
        :title="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
        :aria-label="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
        :aria-pressed="isDark ? 'true' : 'false'"
        @click="toggleTheme"
      >
        <svg v-if="isDark" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="4" /><path d="M12 2v2" /><path d="M12 20v2" /><path d="m4.93 4.93 1.41 1.41" /><path d="m17.66 17.66 1.41 1.41" /><path d="M2 12h2" /><path d="M20 12h2" /><path d="m6.34 17.66-1.41 1.41" /><path d="m19.07 4.93-1.41 1.41" /></svg>
        <svg v-else xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" /></svg>
      </button>
      <SearchHistorySidebar
        :recent-searches="data.recentSearches"
        :current-query="data.query"
        :current-extension="data.extension"
        :current-directory="data.directory"
        @re-search="handleReSearch"
        @remove="removeRecentSearch"
        @clear-all="clearAllRecentSearches"
      />
      <div class="main-content" id="main-content" tabindex="-1">
        <h1 class="sr-only">Code Search</h1>
        <!-- Symbol Search Panel -->
        <h2 class="sr-only">Symbol Search</h2>
        <SymbolSearch :directory="data.directory" />

        <SearchForm
          :data="data"
          @update:caseSensitive="(val) => (data.caseSensitive = val)"
          @update:useRegex="(val) => (data.useRegex = val)"
          @update:includeBinary="(val) => (data.includeBinary = val)"
          @update:fuzzySearch="(val) => (data.fuzzySearch = val)"
          @update:respectGitignore="(val) => (data.respectGitignore = val)"
          @update:minFileSize="(val) => (data.minFileSize = val)"
          @update:maxFileSize="(val) => (data.maxFileSize = val)"
          @update:maxResults="(val) => (data.maxResults = val)"
          @update:contextLines="(val) => (data.contextLines = val)"
          @update:directories="(val) => (data.directories = val)"
          @update:excludePatterns="(val) => (data.excludePatterns = val)"
          @update:allowedFileTypes="(val) => (data.allowedFileTypes = val)"
          @update:query="(val) => (data.query = val)"
          @update:extension="(val) => (data.extension = val)"
          @update:directory="(val) => (data.directory = val)"
          @update:recentSearches="(val) => (data.recentSearches = val)"
          :searchCode="searchCode"
          :selectDirectory="selectDirectory"
          :cancelSearch="cancelSearch"
        />

        <div id="result" class="result" :class="{ error: data.error }" role="status" aria-live="polite">
          {{ data.resultText }}
        </div>

        <div v-if="data.error" class="error-message" id="error-display" role="alert" aria-live="assertive">
          {{ data.error }}
        </div>

        <ProgressIndicator :data="data" :formatFilePath="formatFilePath" />

        <SearchResults
          :data="data"
          :formatFilePath="formatFilePath"
          :openFileLocation="openFileLocation"
          :copyToClipboard="copyToClipboard"
          :onSearch="searchCode"
          @update:resultText="(val) => (data.resultText = val)"
          @update:error="(val) => (data.error = val)"
        />
        <div class="section-spacer" aria-hidden="true"></div>
        <Suspense>
          <LogViewer :data="data" />
          <template #fallback><div aria-hidden="true" /></template>
        </Suspense>

        <!-- Top-level file-preview modal (driven by useFilePreview singleton).
             Used by symbol-search navigation and any component that calls
             openFile(). SearchResults keeps its own CodeModal for "View" clicks. -->
        <Suspense v-if="previewState.isVisible">
          <CodeModal
            :is-visible="previewState.isVisible"
            :file-path="previewState.filePath"
            :file-content="previewState.fileContent"
            :query="previewState.query"
            :files="previewState.files"
            :initial-line="previewState.initialLine"
            @close="closePreview"
          />
          <template #fallback><div aria-hidden="true" /></template>
        </Suspense>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { defineAsyncComponent, onMounted, onUnmounted } from "vue";
import {
  ProgressIndicator,
  SearchForm,
  SearchHistorySidebar,
  SearchResults,
  SymbolSearch,
} from "@/components/ui";
// Non-critical below-fold / on-demand panels: split into separate chunks so
// first paint only ships the search form + results. Direct paths (not the
// barrel) keep the dynamic imports chunkable.
const CodeModal = defineAsyncComponent(() => import("@/components/ui/CodeModal.vue"));
const LogViewer = defineAsyncComponent(() => import("@/components/ui/LogViewer.vue"));
import {
  useFilePreview,
  useKeyboardShortcuts,
  useSearch,
  useTheme,
} from "@/composables";
import type { SymbolInfo } from "@/types";

const { isDark, toggleTheme } = useTheme();

const {
  data,
  searchCode,
  cancelSearch,
  selectDirectory,
  formatFilePath,
  copyToClipboard,
  openFileLocation,
  cleanup,
  focusSearch,
  executeSearch,
  clearSearch,
} = useSearch();

const { previewState, openFile, closePreview } = useFilePreview();

// Symbol-search → code preview: SymbolSearch dispatches a 'symbol-selected'
// CustomEvent. Listen for it and open the preview modal at the symbol's line.
const handleSymbolSelected = (event: Event) => {
  const detail = (event as CustomEvent).detail as SymbolInfo | null | undefined;
  if (!detail?.file || detail.line === undefined) return;
  openFile(detail.file, { initialLine: detail.line });
};

onMounted(() => {
  window.addEventListener("symbol-selected", handleSymbolSelected as EventListener);
});
onUnmounted(() => {
  window.removeEventListener("symbol-selected", handleSymbolSelected as EventListener);
  cleanup();
});

useKeyboardShortcuts(() => ({
  onFocusSearch: focusSearch,
  onExecuteSearch: executeSearch,
  // ESC cancels the in-flight backend search when one is running
  // (cancelSearch bumps the generation token + calls GoCancelSearch);
  // otherwise it clears the form. Wiring ESC to clearSearch unconditionally
  // cleared the query/results while the backend kept running.
  onClearSearch: () => {
    if (data.isSearching) {
      void cancelSearch();
    } else {
      clearSearch();
    }
  },
}));

const handleReSearch = (search: {
  query: string;
  extension: string;
  directory?: string;
}) => {
  data.query = search.query;
  data.extension = search.extension;
  // Restore the directory the search was originally run against so history
  // entries stay genuinely re-runnable.
  if (search.directory) {
    data.directory = search.directory;
  }
  searchCode();
};

const removeRecentSearch = (index: number) => {
  data.recentSearches.splice(index, 1);
};

const clearAllRecentSearches = () => {
  data.recentSearches = [];
};

</script>

<style scoped>
.app-layout {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  grid-template-areas: "sidebar main";
  min-height: 100vh;
  width: 100%;
}

.search-history-sidebar {
  grid-area: sidebar;
  height: 100vh;
  position: sticky;
  top: 0;
}

.theme-toggle {
  position: fixed;
  top: var(--space-2);
  right: var(--space-3);
  z-index: 900;
  width: 44px;
  height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-medium);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  cursor: pointer;
  opacity: 0.75;
  transition: opacity var(--transition-fast), background var(--transition-fast);
}

.theme-toggle:hover {
  opacity: 1;
  background: var(--color-bg-hover);
}

.main-content {
  grid-area: main;
  min-width: 0;
  overflow-x: auto;
  padding: var(--space-3) var(--space-5) 0;
}

/* Narrow screens: stack sidebar above the main content */
@media (max-width: 768px) {
  .app-layout {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas:
      "sidebar"
      "main";
  }

  .search-history-sidebar {
    height: auto;
    position: static;
    width: 100%;
  }

  .main-content {
    padding: var(--space-3) var(--space-4);
  }
}

.section-spacer {
  margin-top: var(--space-6);
  height: var(--space-2);
}

.result {
  min-height: var(--space-5);
  line-height: var(--space-5);
  margin: var(--space-5) auto;
  text-align: center;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.result.error {
  color: var(--color-danger);
}

.error-message {
  max-width: 37.5rem;
  margin: var(--space-2) auto;
  padding: var(--space-3);
  background-color: color-mix(in srgb, var(--color-danger) 15%, var(--color-bg));
  border: 1px solid var(--color-danger);
  border-radius: var(--radius-sm);
  color: var(--color-danger-dark);
  text-align: center;
  font-size: var(--font-size-sm);
}

</style>
