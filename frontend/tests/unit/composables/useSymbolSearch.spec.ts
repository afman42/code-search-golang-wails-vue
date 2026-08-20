import { describe, test, expect, vi, beforeEach } from "vitest";
import { ref } from "vue";
import { SearchSymbols } from "@wails/go/main/App";
import { useSymbolSearch } from "@/composables/useSymbolSearch";

vi.mock("@wails/go/main/App", () => ({
  GetAllSymbols: vi.fn(),
  SearchSymbols: vi.fn(),
}));
vi.mock("@wails/runtime", () => ({
  EventsOn: vi.fn().mockReturnValue(() => {}),
}));

const mockSearchSymbols = vi.mocked(SearchSymbols);

describe("useSymbolSearch", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("stale response does not overwrite a newer search's results", async () => {
    const directory = ref("/repo");
    const { searchQuery, symbolResults, handleSymbolSearch } = useSymbolSearch(
      () => directory.value,
    );

    // First call resolves slowly.
    const { promise: slow, resolve: resolveSlow } = Promise.withResolvers<
      unknown[]
    >();
    mockSearchSymbols.mockReturnValueOnce(slow as Promise<unknown[]>);
    // Second call resolves immediately.
    mockSearchSymbols.mockResolvedValueOnce([
      { name: "New", type: "fn", line: 1, file: "/repo/b.go" },
    ]);

    searchQuery.value = "old";
    const first = handleSymbolSearch();
    searchQuery.value = "new";
    const second = handleSymbolSearch();

    await second;
    expect(symbolResults.value).toEqual([
      { name: "New", type: "fn", line: 1, file: "/repo/b.go" },
    ]);

    // First (stale) response lands late — must NOT overwrite the newer
    // results (generation mismatch discards it).
    resolveSlow([{ name: "Old", type: "fn", line: 2, file: "/repo/a.go" }]);
    await first;

    expect(symbolResults.value).toEqual([
      { name: "New", type: "fn", line: 1, file: "/repo/b.go" },
    ]);
  });
});