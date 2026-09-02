import { describe, test, expect, beforeEach } from "vitest";
import { ref, nextTick, type Ref } from "vue";
import type { SearchResult } from "@/types";
import { useSelectionManager } from "@/composables/useSelectionManager";

// These tests defend the reactivity contract of the selection manager. The
// original implementation held selection in a PLAIN Set, so mutations never
// re-ran selectedCount / allVisibleSelected — the "N selected" badge never
// updated and Copy/Export stayed disabled. The spec below fails on a plain
// Set (the computed stays cached at its first value) and passes on
// reactive(Set). The read-then-mutate-then-read ordering is deliberate: it
// primes the computed cache exactly the way a rendered component does before
// the user clicks.

function makeResult(i: number): SearchResult {
  return {
    filePath: `/proj/file${i}.go`,
    lineNum: i + 1,
    content: `line ${i} content`,
    matchedText: "content",
    contextBefore: [],
    contextAfter: [],
  };
}

describe("useSelectionManager", () => {
  // A single page of 4 results (indices 0..3).
  let totalResults: Ref<number>;
  let startIndex: Ref<number>;
  let endIndex: Ref<number>;

  beforeEach(() => {
    totalResults = ref(4);
    startIndex = ref(0);
    endIndex = ref(4);
  });

  function make() {
    return useSelectionManager({
      totalResults: () => totalResults.value,
      startIndex: () => startIndex.value,
      endIndex: () => endIndex.value,
    });
  }

  test("selectedCount is 0 initially", () => {
    const sm = make();
    expect(sm.selectedCount.value).toBe(0);
    expect(sm.isAnySelected()).toBe(false);
  });

  test("toggleSelected updates reactive selectedCount (the regression guard)", async () => {
    const sm = make();
    // Prime the computed cache the way a rendered template does.
    expect(sm.selectedCount.value).toBe(0);

    sm.toggleSelected(0);
    await nextTick();
    // On a plain (non-reactive) Set this stays 0 — the bug. Must be 1.
    expect(sm.selectedCount.value).toBe(1);
    expect(sm.isSelected(0)).toBe(true);
    expect(sm.isAnySelected()).toBe(true);

    sm.toggleSelected(0);
    await nextTick();
    expect(sm.selectedCount.value).toBe(0);
    expect(sm.isSelected(0)).toBe(false);
  });

  test("toggleSelected on distinct indices accumulates", async () => {
    const sm = make();
    expect(sm.selectedCount.value).toBe(0);
    sm.toggleSelected(0);
    sm.toggleSelected(2);
    await nextTick();
    expect(sm.selectedCount.value).toBe(2);
    expect(sm.isSelected(0)).toBe(true);
    expect(sm.isSelected(1)).toBe(false);
    expect(sm.isSelected(2)).toBe(true);
  });

  test("allVisibleSelected reflects the current page and stays reactive", async () => {
    const sm = make();
    expect(sm.allVisibleSelected.value).toBe(false);

    sm.toggleSelectAll();
    await nextTick();
    expect(sm.allVisibleSelected.value).toBe(true);
    expect(sm.selectedCount.value).toBe(4);
    for (let i = 0; i < 4; i++) expect(sm.isSelected(i)).toBe(true);

    // Toggling again clears the visible page.
    sm.toggleSelectAll();
    await nextTick();
    expect(sm.allVisibleSelected.value).toBe(false);
    expect(sm.selectedCount.value).toBe(0);
  });

  test("allVisibleSelected is false when the page is empty (no results)", () => {
    totalResults.value = 0;
    startIndex.value = 0;
    endIndex.value = 0;
    const sm = make();
    expect(sm.allVisibleSelected.value).toBe(false);
  });

  test("allVisibleSelected turns false when one visible item is unselected", async () => {
    const sm = make();
    sm.toggleSelectAll();
    await nextTick();
    expect(sm.allVisibleSelected.value).toBe(true);

    sm.toggleSelected(1);
    await nextTick();
    expect(sm.allVisibleSelected.value).toBe(false);
    expect(sm.selectedCount.value).toBe(3);
  });

  test("clearSelection empties the selection reactively", async () => {
    const sm = make();
    sm.toggleSelectAll();
    await nextTick();
    expect(sm.selectedCount.value).toBe(4);

    sm.clearSelection();
    await nextTick();
    expect(sm.selectedCount.value).toBe(0);
    expect(sm.isAnySelected()).toBe(false);
  });

  test("copySelectedResults copies only selected rows, sorted by index", async () => {
    const sm = make();
    const results = [0, 1, 2, 3].map(makeResult);
    sm.toggleSelected(2);
    sm.toggleSelected(0);

    let copied = "";
    await sm.copySelectedResults(results, async (text) => {
      copied = text;
      return true;
    });

    // Index 0 before index 2 despite selection order.
    expect(copied).toBe(
      `/proj/file0.go:1 line 0 content\n/proj/file2.go:3 line 2 content`,
    );
  });

  test("copySelectedResults is a no-op when nothing is selected", async () => {
    const sm = make();
    const results = [0, 1].map(makeResult);
    let called = false;
    await sm.copySelectedResults(results, async () => {
      called = true;
      return true;
    });
    expect(called).toBe(false);
  });

  test("exportSelectedResults exports the selected subset when any are selected", async () => {
    const sm = make();
    const results = [0, 1, 2, 3].map(makeResult);
    sm.toggleSelected(1);

    let exported: unknown[] = [];
    const path = await sm.exportSelectedResults(results, async (rows, fmt) => {
      exported = rows;
      expect(fmt).toBe("csv");
      return "/tmp/out.csv";
    });

    expect(path).toBe("/tmp/out.csv");
    expect(exported).toHaveLength(1);
    expect((exported[0] as SearchResult).filePath).toBe("/proj/file1.go");
  });

  test("exportSelectedResults falls back to ALL results when none selected", async () => {
    const sm = make();
    const results = [0, 1, 2, 3].map(makeResult);

    let exported: unknown[] = [];
    const path = await sm.exportSelectedResults(results, async (rows) => {
      exported = rows;
      return "/tmp/all.csv";
    });

    expect(path).toBe("/tmp/all.csv");
    expect(exported).toHaveLength(4);
  });

  test("exportSelectedResults threads the requested format through", async () => {
    const sm = make();
    const results = [0, 1].map(makeResult);

    // Regression guard: the format was hardcoded to "csv" here, so the UI's
    // "Export JSON" button silently wrote a CSV file.
    let seenFormat = "";
    await sm.exportSelectedResults(
      results,
      async (_rows, fmt) => {
        seenFormat = fmt;
        return "/tmp/out.json";
      },
      "json",
    );

    expect(seenFormat).toBe("json");
  });

  test("results watch clears selection when result-set identity changes", async () => {
    // Use a ref so the composable tracks the watch.
    const results = ref([makeResult(0), makeResult(1)]);
    const sm = useSelectionManager({
      totalResults: () => totalResults.value,
      startIndex: () => startIndex.value,
      endIndex: () => endIndex.value,
      results,
    });
    sm.toggleSelected(0);
    await nextTick();
    expect(sm.selectedCount.value).toBe(1);

    // Replace the result set.
    results.value = [makeResult(2), makeResult(3)];
    await nextTick();
    expect(sm.selectedCount.value).toBe(0);
  });
});
