import { computed, reactive, unref, watch, type MaybeRefOrGetter } from "vue";
import type { SearchState } from "@/types";

/** Formats the Go `ExportSearchResults` binding accepts (export.go switches on
 *  these two; anything else falls back to CSV backend-side). Declared here
 *  because this composable is what threads the format from the UI to the
 *  binding. */
export type ExportFormat = "csv" | "json";

interface UseSelectionManagerOptions {
  totalResults: MaybeRefOrGetter<number>;
  startIndex: MaybeRefOrGetter<number>;
  endIndex: MaybeRefOrGetter<number>;
  /** Current result set — when it is replaced, the selection is cleared so
   *  stale indices can't point at a different result set. */
  results?: MaybeRefOrGetter<SearchState["searchResults"]>;
}

// Resolve a MaybeRefOrGetter<number> to its current numeric value. Handles
// plain numbers, refs/computeds (unref), and getter functions.
const read = (v: MaybeRefOrGetter<number>): number =>
  typeof v === "function" ? (v as () => number)() : unref(v);

export function useSelectionManager(options: UseSelectionManagerOptions) {
  // reactive() so mutations to the Set (add/delete) re-run selectedCount /
  // allVisibleSelected and re-render the batch-actions UI. A plain Set is not
  // tracked by Vue, so the selected-count badge never appeared (#regression).
  const selectedIndices = reactive(new Set<number>());

  // Selection indices are meaningless once the result set is replaced (new
  // search, directory change, re-run): indices into the old array would
  // silently select different rows in the new one. Watch the result-set
  // identity and drop the selection on replacement.
  if (options.results !== undefined) {
    watch(
      () =>
        typeof options.results === "function"
          ? options.results()
          : unref(options.results),
      () => {
        selectedIndices.clear();
      },
    );
  }

  const selectedCount = computed(() => selectedIndices.size);

  const allVisibleSelected = computed(() => {
    if (read(options.totalResults) === 0) return false;
    for (let i = read(options.startIndex); i < read(options.endIndex); i++) {
      if (!selectedIndices.has(i)) return false;
    }
    return true;
  });

  const isSelected = (idx: number) => selectedIndices.has(idx);

  const toggleSelected = (idx: number) => {
    if (selectedIndices.has(idx)) {
      selectedIndices.delete(idx);
    } else {
      selectedIndices.add(idx);
    }
  };

  const toggleSelectAll = () => {
    if (allVisibleSelected.value) {
      for (let i = read(options.startIndex); i < read(options.endIndex); i++) {
        selectedIndices.delete(i);
      }
    } else {
      for (let i = read(options.startIndex); i < read(options.endIndex); i++) {
        selectedIndices.add(i);
      }
    }
  };

  const copySelectedResults = async (
    results: SearchState["searchResults"],
    copyToClipboard: (text: string) => Promise<boolean>
  ): Promise<boolean> => {
    if (!Array.isArray(results)) return false;
    const selected = Array.from(selectedIndices)
      .sort((a, b) => a - b)
      .map((i) => results[i])
      .filter(Boolean);
    if (selected.length === 0) return false;

    const text = selected
      .map((r) => `${r.filePath}:${r.lineNum} ${r.content}`)
      .join("\n");

    // Direct propagation: copyToClipboardWithToast owns failure feedback
    // (error toast + boolean), so the caller can gate its success toast on
    // the result instead of celebrating a failed copy.
    return await copyToClipboard(text);
  };

  const exportSelectedResults = async (
    results: SearchState["searchResults"],
    exportFn: (results: unknown[], format: ExportFormat) => Promise<string | undefined>,
    // Default keeps pre-format callers (and the plain "Export" path) on CSV.
    format: ExportFormat = "csv"
  ): Promise<string | null> => {
    if (!Array.isArray(results) || results.length === 0) return null;

    let toExport = results;
    if (selectedIndices.size > 0) {
      toExport = Array.from(selectedIndices)
        .sort((a, b) => a - b)
        .map((i) => results[i])
        .filter(Boolean);
    }

    // Rejections propagate to the caller's catch (which shows the error
    // toast); a backend cancel ("") still maps to null without throwing.
    return (await exportFn(toExport, format)) ?? null;
  };

  const clearSelection = () => {
    selectedIndices.clear();
  };

  const isAnySelected = () => selectedIndices.size > 0;

  return {
    selectedCount,
    allVisibleSelected,
    isSelected,
    toggleSelected,
    toggleSelectAll,
    copySelectedResults,
    exportSelectedResults,
    clearSelection,
    isAnySelected,
  };
}
