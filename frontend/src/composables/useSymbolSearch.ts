import { ref, computed } from "vue";
import { GetAllSymbols, SearchSymbols as GoSearchSymbols } from "@wails/go/main/App";
import { EventsOn } from "@wails/runtime";
import { formatFilePath, toErrorMessage } from "@/utils";
import type { SymbolInfo } from "@/types";
import { toastManager } from "./useToast";

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
    isSearching.value = true;
    hasSearched.value = true;
    symbolResults.value = [];
    selectedIndex.value = -1;
    statusMessage.value = "";
    statusType.value = "";

    try {
      const results = (await GoSearchSymbols(query, directory() as string, 50)) as SymbolInfo[];
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
      const msg = toErrorMessage(error, "Could not search symbols");
      statusMessage.value = `Error searching symbols: ${msg}`;
      statusType.value = "error";
      toastManager.error(msg, "Symbol Search Failed");
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
      statusMessage.value = "All symbols loaded. Start typing to search.";
      statusType.value = "info";
      return;
    }

    if (!directory()) {
      statusMessage.value = "Select a directory in the search form first";
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
      const p = (payload && typeof payload === "object" ? payload : {}) as Record<string, unknown>;
      const processed = typeof p.processed === "number" ? p.processed : 0;
      const total = typeof p.total === "number" ? p.total : 0;
      fetchProgress.value = total > 0 ? Math.round((processed / total) * 100) : 0;
    });

    try {
      // Call GetAllSymbols which processes files under the selected directory.
      const results = (await GetAllSymbols(directory() as string, 2000)) as SymbolInfo[];

      allSymbols.value = results;
      fetchProgress.value = 100;

      setTimeout(() => {
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
