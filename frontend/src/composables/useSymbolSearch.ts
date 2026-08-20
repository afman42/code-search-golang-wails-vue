import { ref, computed, watch, onUnmounted, getCurrentInstance } from "vue";
import { GetAllSymbols, SearchSymbols as GoSearchSymbols } from "@wails/go/main/App";
import { EventsOn } from "@wails/runtime";
import { formatFilePath, toErrorMessage } from "@/utils";
import type { SymbolInfo } from "@/types";
import { toastManager } from "./useToast";
import { coerceProgress } from "./searchProgress";

export function useSymbolSearch(directory: () => string | undefined) {
  // Reactive state
  const searchQuery = ref("");
  const symbolResults = ref<SymbolInfo[]>([]);
  const allSymbols = ref<SymbolInfo[]>([]);
  const isSearching = ref(false);
  const isFetchingAll = ref(false);
  const selectedIndex = ref(-1);
  const hasSearched = ref(false);
  const statusMessage = ref("");
  const statusType = ref("");
  const fetchProgress = ref(0);
  const showFetchProgress = ref(false);

  // Generation counter: each handleSymbolSearch run captures its own token so
  // a slow response cannot overwrite the results of a newer search.
  let searchGeneration = 0;

  // Computed property for recently seen symbols (last 5 indexed)
  const recentlySeenSymbols = computed(() => {
    if (!allSymbols.value.length) return [];
    return allSymbols.value.slice(-5);
  });
  const handleSymbolSearch = async () => {
    if (!searchQuery.value.trim()) return;

    if (!directory()) {
      statusMessage.value = "Select a directory in the search form first";
      statusType.value = "info";
      return;
    }

    const query = searchQuery.value.trim();
    const myGeneration = ++searchGeneration;
    isSearching.value = true;
    hasSearched.value = true;
    symbolResults.value = [];
    selectedIndex.value = -1;
    statusMessage.value = "";
    statusType.value = "";

    try {
      const results = (await GoSearchSymbols(query, directory() as string, 50)) as SymbolInfo[];
      // Discard stale responses: a newer search superseded this one.
      if (myGeneration !== searchGeneration) return;
      symbolResults.value = results;

      if (results.length === 0) {
        statusMessage.value = `No symbols found matching "${query}"`;
        statusType.value = "info";
      } else {
        const suffix = results.length === 1 ? "" : "s";
        statusMessage.value = `Found ${results.length} symbol${suffix}`;
        statusType.value = "success";
      }
    } catch (error: unknown) {
      if (myGeneration !== searchGeneration) return;
      const msg = toErrorMessage(error, "Could not search symbols");
      statusMessage.value = `Error searching symbols: ${msg}`;
      statusType.value = "error";
      toastManager.error(msg, "Symbol Search Failed");
    } finally {
      if (myGeneration === searchGeneration) {
        isSearching.value = false;
      }
    }
  };

  // The allSymbols cache is keyed to the directory: when the directory
  // changes, the cache is cleared so a fetch re-indexes the new folder
  // instead of showing another directory's symbols.
  let hideProgressTimer: number | null = null;
  // Handles for onUnmounted cleanup: in-flight symbol-progress subscription
  // and the hide-progress timeout.
  let progressStop: (() => void) | null = null;

  // Re-index when the scanned directory changes.
  watch(directory, () => {
    allSymbols.value = [];
    hasSearched.value = false;
  });

  // Fetch all symbols from indexed files
  const fetchAllSymbols = async () => {
    // Re-entry guard: a second call while a fetch is in flight would arm a
    // duplicate progress subscription and double-fetch.
    if (isFetchingAll.value) return;

    if (!directory()) {
      statusMessage.value = "Select a directory in the search form first";
      statusType.value = "info";
      return;
    }

    if (allSymbols.value.length > 0) {
      // If already fetched, just show them
      hasSearched.value = false;
      symbolResults.value = [];
      statusMessage.value = "All symbols loaded. Start typing to search.";
      statusType.value = "info";
      return;
    }

    isFetchingAll.value = true;
    showFetchProgress.value = true;
    fetchProgress.value = 0;
    symbolResults.value = [];
    hasSearched.value = false;

    // Subscribe to real per-file scan progress emitted by the Go backend
    // (symbol-progress). This replaces the previous synthetic 0->100 jump.
    const stopProgress = EventsOn("symbol-progress", (payload: unknown) => {
      const p = coerceProgress(payload);
      fetchProgress.value =
        p.totalFiles > 0 ? Math.round((p.processedFiles / p.totalFiles) * 100) : 0;
    });
    progressStop = stopProgress;

    try {
      // Call GetAllSymbols which processes files under the selected directory.
      const results = (await GetAllSymbols(directory() as string, 2000)) as SymbolInfo[];

      allSymbols.value = results;
      fetchProgress.value = 100;

      hideProgressTimer = window.setTimeout(() => {
        showFetchProgress.value = false;
        fetchProgress.value = 0;
      }, 1000);

      statusMessage.value = `Indexed ${results.length.toLocaleString()} symbols from your project files`;
      statusType.value = "success";

      toastManager.success(
        `Found ${results.length.toLocaleString()} symbols`,
        "Symbols Indexed"
      );
    } catch (error: unknown) {
      const msg = toErrorMessage(error, "Could not index symbols");
      statusMessage.value = `Error loading symbols: ${msg}`;
      statusType.value = "error";
      toastManager.error(msg, "Failed to Load Symbols");
      showFetchProgress.value = false;
      fetchProgress.value = 0;
    } finally {
      stopProgress();
      progressStop = null;
      isFetchingAll.value = false;
    }
  };

  // Select a symbol result: open the code preview modal at the symbol's line
  // (via the useFilePreview singleton) and show a toast with the location.
  const selectSymbol = (symbol: SymbolInfo) => {
    toastManager.info(
      `${symbol.name} at ${formatFilePath(symbol.file)}:${symbol.line}`,
      "Symbol Selected"
    );

    // Dispatch an event that CodeSearch.vue listens for to open the preview
    // modal at the symbol's file:line with a flash highlight.
    window.dispatchEvent(new CustomEvent("symbol-selected", { detail: symbol }));
  };

  // Prefill search and navigate to symbol
  const prefillSearchAndNavigate = async (symbol: SymbolInfo) => {
    searchQuery.value = symbol.name;
    hasSearched.value = false;
    symbolResults.value = [];

    // Trigger search for this symbol
    await handleSymbolSearch();
  };

  // Release in-flight progress subscription and hide-progress timer on
  // unmount. Guarded so direct (non-component) callers in tests don't warn.
  if (getCurrentInstance()) {
    onUnmounted(() => {
      if (progressStop) {
        progressStop();
        progressStop = null;
      }
      if (hideProgressTimer !== null) {
        clearTimeout(hideProgressTimer);
        hideProgressTimer = null;
      }
    });
  }

  return {
    // state
    searchQuery,
    symbolResults,
    allSymbols,
    isSearching,
    isFetchingAll,
    selectedIndex,
    hasSearched,
    statusMessage,
    statusType,
    fetchProgress,
    showFetchProgress,
    // computed
    recentlySeenSymbols,
    // methods
    handleSymbolSearch,
    fetchAllSymbols,
    selectSymbol,
    prefillSearchAndNavigate,
  };
}
