import { describe, test, expect, vi, beforeEach } from "vitest";
import { mount } from "@vue/test-utils";
import {
  SearchSymbols,
  GetAllSymbols,
} from "@wails/go/main/App";
import { EventsOn } from "@wails/runtime";
import SymbolSearch from "@/components/ui/SymbolSearch.vue";
import type { SymbolInfo } from "@/types/search";

// Mock the Wails Go bindings. The component imports `SearchSymbols as
// GoSearchSymbols` and `GetAllSymbols`; we mock the module path it imports
// from so both names are controlled here.
vi.mock("@wails/go/main/App", () => ({
  SearchSymbols: vi.fn(),
  GetAllSymbols: vi.fn(),
}));

// Mock the Wails runtime; fetchAllSymbols subscribes to a "symbol-progress"
// event whose unsubscribe function is captured via EventsOn's return value.
vi.mock("@wails/runtime", () => ({
  EventsOn: vi.fn().mockReturnValue(() => {}),
}));

const mockSearchSymbols = vi.mocked(SearchSymbols);
const mockGetAllSymbols = vi.mocked(GetAllSymbols);
const mockEventsOn = vi.mocked(EventsOn);

// A factory for sample symbols so each test gets fresh objects.
const makeSymbol = (overrides: Partial<SymbolInfo> = {}): SymbolInfo => ({
  name: "Foo",
  type: "function",
  line: 10,
  file: "/repo/src/foo.go",
  ...overrides,
});

// The component exposes reactive state via defineExpose. Through the test-utils
// public-instance proxy refs are auto-unwrapped, so `vm.searchQuery` is the raw
// string, `vm.allSymbols` the raw array, and computeds resolve to their value.
type VM = {
  searchQuery: string;
  allSymbols: SymbolInfo[];
  symbolResults: SymbolInfo[];
  statusMessage: string;
  statusType: string;
  hasSearched: boolean;
  recentlySeenSymbols: SymbolInfo[];
  handleSymbolSearch: () => Promise<void>;
  fetchAllSymbols: () => Promise<void>;
  selectSymbol: (symbol: SymbolInfo) => void;
};

const mountComponent = (props: { directory?: string } = {}) => {
  return mount(SymbolSearch, { props });
};

const vmOf = (wrapper: ReturnType<typeof mountComponent>): VM =>
  wrapper.vm as unknown as VM;

describe("SymbolSearch.vue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchSymbols.mockReset();
    mockGetAllSymbols.mockReset();
  });

  // ---- handleSymbolSearch ----

  test("handleSymbolSearch no-ops when searchQuery is empty or whitespace", async () => {
    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    vm.searchQuery = "   ";
    await vm.handleSymbolSearch();

    expect(mockSearchSymbols).not.toHaveBeenCalled();
    // No status set, not flagged as searched.
    expect(vm.statusMessage).toBe("");
    expect(vm.hasSearched).toBe(false);
  });

  test("handleSymbolSearch sets info status and returns when directory is unset", async () => {
    const wrapper = await mountComponent({ directory: undefined });
    const vm = vmOf(wrapper);

    vm.searchQuery = "Foo";
    await vm.handleSymbolSearch();

    expect(mockSearchSymbols).not.toHaveBeenCalled();
    expect(vm.statusMessage).toBe(
      "Select a directory in the search form first",
    );
    expect(vm.statusType).toBe("info");
  });

  test("handleSymbolSearch calls SearchSymbols(query, directory, 50) and populates results", async () => {
    const results = [makeSymbol(), makeSymbol({ name: "Bar", line: 20 })];
    mockSearchSymbols.mockResolvedValue(results);

    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    // Leading/trailing whitespace is trimmed before the call.
    vm.searchQuery = "  Foo  ";
    await vm.handleSymbolSearch();

    expect(mockSearchSymbols).toHaveBeenCalledTimes(1);
    expect(mockSearchSymbols).toHaveBeenCalledWith("Foo", "/repo", 50);
    expect(vm.symbolResults).toEqual(results);
    expect(vm.hasSearched).toBe(true);
    expect(vm.statusType).toBe("success");
    expect(vm.statusMessage).toBe("Found 2 symbols");
  });

  test("handleSymbolSearch reports info status when SearchSymbols returns no results", async () => {
    mockSearchSymbols.mockResolvedValue([]);

    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    vm.searchQuery = "NoSuch";
    await vm.handleSymbolSearch();

    expect(mockSearchSymbols).toHaveBeenCalledWith("NoSuch", "/repo", 50);
    expect(vm.symbolResults).toEqual([]);
    expect(vm.statusType).toBe("info");
    expect(vm.statusMessage).toBe('No symbols found matching "NoSuch"');
  });

  test("handleSymbolSearch sets error status when SearchSymbols rejects", async () => {
    mockSearchSymbols.mockRejectedValue(new Error("boom"));

    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    vm.searchQuery = "Foo";
    await vm.handleSymbolSearch();

    expect(vm.statusType).toBe("error");
    expect(vm.statusMessage).toContain("boom");
    expect(vm.symbolResults).toEqual([]);
  });

  // ---- fetchAllSymbols ----

  test("fetchAllSymbols short-circuits when allSymbols already populated", async () => {
    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    // Seed the cache so the short-circuit path is taken.
    vm.allSymbols = [makeSymbol()];
    vm.symbolResults = [makeSymbol({ name: "Stale" })];
    vm.hasSearched = true;

    await vm.fetchAllSymbols();

    expect(mockGetAllSymbols).not.toHaveBeenCalled();
    expect(mockEventsOn).not.toHaveBeenCalled();
    // Short-circuit resets search state and surfaces an info message.
    expect(vm.hasSearched).toBe(false);
    expect(vm.symbolResults).toEqual([]);
    expect(vm.statusType).toBe("info");
    expect(vm.statusMessage).toBe(
      "All symbols loaded. Start typing to search.",
    );
  });

  test("fetchAllSymbols sets info status and returns when directory is unset", async () => {
    const wrapper = await mountComponent({ directory: undefined });
    const vm = vmOf(wrapper);

    await vm.fetchAllSymbols();

    expect(mockGetAllSymbols).not.toHaveBeenCalled();
    expect(mockEventsOn).not.toHaveBeenCalled();
    expect(vm.statusMessage).toBe(
      "Select a directory in the search form first",
    );
    expect(vm.statusType).toBe("info");
  });

  test("fetchAllSymbols calls GetAllSymbols(directory, 2000) and populates allSymbols", async () => {
    const symbols = [
      makeSymbol({ name: "A", line: 1 }),
      makeSymbol({ name: "B", line: 2 }),
    ];
    mockGetAllSymbols.mockResolvedValue(symbols);

    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    await vm.fetchAllSymbols();

    expect(mockGetAllSymbols).toHaveBeenCalledTimes(1);
    expect(mockGetAllSymbols).toHaveBeenCalledWith("/repo", 2000);
    // Subscribes to progress while fetching, then unsubscribes on completion.
    expect(mockEventsOn).toHaveBeenCalledWith(
      "symbol-progress",
      expect.any(Function),
    );
    expect(vm.allSymbols).toEqual(symbols);
    expect(vm.statusType).toBe("success");
  });

  test("fetchAllSymbols sets error status when GetAllSymbols rejects", async () => {
    mockGetAllSymbols.mockRejectedValue(new Error("index failed"));

    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    await vm.fetchAllSymbols();

    expect(vm.statusType).toBe("error");
    expect(vm.statusMessage).toContain("index failed");
    expect(vm.allSymbols).toEqual([]);
  });

  // ---- selectSymbol ----

  test("selectSymbol dispatches a 'symbol-selected' CustomEvent with the symbol as detail", async () => {
    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    const handler = vi.fn();
    window.addEventListener("symbol-selected", handler);

    const symbol = makeSymbol({ name: "DoStuff", line: 42 });
    vm.selectSymbol(symbol);

    expect(handler).toHaveBeenCalledTimes(1);
    const event = handler.mock.calls[0][0] as CustomEvent<SymbolInfo>;
    expect(event instanceof CustomEvent).toBe(true);
    expect(event.type).toBe("symbol-selected");
    expect(event.detail).toEqual(symbol);

    window.removeEventListener("symbol-selected", handler);
  });

  // ---- recentlySeenSymbols ----

  test("recentlySeenSymbols returns the last 5 of allSymbols", async () => {
    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    const symbols = Array.from({ length: 8 }, (_, i) =>
      makeSymbol({ name: `S${i}`, line: i }),
    );
    vm.allSymbols = symbols;

    expect(vm.recentlySeenSymbols).toEqual(symbols.slice(-5));
    expect(vm.recentlySeenSymbols).toHaveLength(5);
  });

  test("recentlySeenSymbols returns empty array when allSymbols is empty", async () => {
    const wrapper = await mountComponent({ directory: "/repo" });
    const vm = vmOf(wrapper);

    expect(vm.recentlySeenSymbols).toEqual([]);
  });
});
