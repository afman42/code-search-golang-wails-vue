import { computed, unref, type MaybeRefOrGetter } from "vue";
import type { SearchState } from "@/types";

interface UseSelectionManagerOptions {
  totalResults: MaybeRefOrGetter<number>;
  startIndex: MaybeRefOrGetter<number>;
  endIndex: MaybeRefOrGetter<number>;
}

// Resolve a MaybeRefOrGetter<number> to its current numeric value. Handles
// plain numbers, refs/computeds (unref), and getter functions.
const read = (v: MaybeRefOrGetter<number>): number =>
  typeof v === "function" ? (v as () => number)() : unref(v);

export function useSelectionManager(options: UseSelectionManagerOptions) {
  const selectedIndices = new Set<number>();

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
  ) => {
    if (!Array.isArray(results)) return;
    const selected = Array.from(selectedIndices)
      .sort((a, b) => a - b)
      .map((i) => results[i])
      .filter(Boolean);
    if (selected.length === 0) return;

    const text = selected
      .map((r) => `${r.filePath}:${r.lineNum} ${r.content}`)
      .join("\n");

    try {
      await copyToClipboard(text);
    } catch (err: unknown) {
      console.error("Copy failed:", err);
    }
  };

  const exportSelectedResults = async (
    results: SearchState["searchResults"],
    exportFn: (results: unknown[], format: string) => Promise<string | undefined>
  ) => {
    if (!Array.isArray(results) || results.length === 0) return;

    let toExport = results;
    if (selectedIndices.size > 0) {
      toExport = Array.from(selectedIndices)
        .sort((a, b) => a - b)
        .map((i) => results[i])
        .filter(Boolean);
    }

    try {
      const savedPath = await exportFn(toExport, "csv");
      if (savedPath) return savedPath;
    } catch (err: unknown) {
      console.error("Export failed:", err);
    }
    return null;
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
