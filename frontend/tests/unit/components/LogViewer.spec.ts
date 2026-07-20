import { vi } from "vitest";
import { mount } from "@vue/test-utils";
import LogViewer from "../../../src/components/ui/LogViewer.vue";
import {
  makeEditorAvailability,
  makeEditorDetectionStatus,
} from "../../fixtures/editorAvailability";
import { parseLogEntry } from "../../../src/composables/useLogStreaming";

// Mock the Wails binding modules before any imports that use them.
// The composable (useLogStreaming) imports these, so the mock applies there too.
vi.mock("../../../wailsjs/go/main/App", () => ({
  GetInitialLogs: vi.fn(),
  GetNewLogs: vi.fn(),
}));

import {
  GetInitialLogs,
  GetNewLogs,
} from "../../../wailsjs/go/main/App";

// Track the wrappers so afterEach can unmount them
let wrappers: ReturnType<typeof mount>[] = [];

// Mock fileUtils handleEditorSelect
vi.mock("../../../src/utils/fileUtils", () => ({
  handleEditorSelect: vi.fn(),
}));

const mockData = {
  directory: "",
  query: "",
  extension: "",
  caseSensitive: false,
  useRegex: false,
  includeBinary: false,
  maxFileSize: 10485760,
  maxResults: 1000,
  searchSubdirs: true,
  resultText: "",
  searchResults: [],
  truncatedResults: false,
  isSearching: false,
  searchProgress: {
    processedFiles: 0,
    totalFiles: 0,
    currentFile: "",
    resultsCount: 0,
    status: "",
  },
  showProgress: false,
  minFileSize: 0,
  excludePatterns: [],
  allowedFileTypes: [],
  knownTextExtensions: [],
  recentSearches: [],
  error: null,
  availableEditors: makeEditorAvailability(),
  editorDetectionStatus: makeEditorDetectionStatus(),
};

function createWrapper() {
  const wrapper = mount(LogViewer, {
    props: {
      data: mockData,
    },
    attachTo: document.body,
  });
  wrappers.push(wrapper);
  return wrapper;
}

// Wait for the initial async call to resolve (composable's getInitialLogsWithRetry)
async function waitForInitialCall() {
  // The composable's onMounted calls startPolling -> getInitialLogsWithRetry.
  // Resolve the mock so it returns an empty array and the retry loop exits.
  vi.mocked(GetInitialLogs).mockResolvedValue([]);
  // Flush microtasks so the async handler runs
  await new Promise((r) => setTimeout(r, 0));
  await new Promise((r) => setTimeout(r, 0));
}

describe("LogViewer.vue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    wrappers = [];

    // Default: GetInitialLogs returns an empty array, GetNewLogs too
    vi.mocked(GetInitialLogs).mockResolvedValue([]);
    vi.mocked(GetNewLogs).mockResolvedValue([]);
  });

  afterEach(() => {
    // Unmount all wrappers to clean up intervals and event listeners
    wrappers.forEach((w) => {
      if (w && w.unmount) w.unmount();
    });
    wrappers = [];
    document.body.innerHTML = "";
  });

  describe("Collapse/Expand", () => {
    test("starts collapsed by default", async () => {
      const wrapper = createWrapper();
      expect(wrapper.find(".log-collapsed").exists()).toBe(true);
      expect(wrapper.find(".log-content-wrapper").exists()).toBe(false);
    });

    test("expands when toggle button is clicked", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");
      expect(wrapper.find(".log-collapsed").exists()).toBe(false);
      expect(wrapper.find(".log-content-wrapper").exists()).toBe(true);
    });

    test("collapses when toggle button is clicked again", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");
      expect(wrapper.find(".log-content-wrapper").exists()).toBe(true);

      await wrapper.find(".log-toggle-button").trigger("click");
      expect(wrapper.find(".log-collapsed").exists()).toBe(true);
    });
  });

  describe("Header", () => {
    test("renders header with title and controls when expanded", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");

      expect(wrapper.find("h3").text()).toBe("Live Log Viewer");
      expect(wrapper.find(".btn-primary").exists()).toBe(true);
      expect(wrapper.find(".btn-secondary").exists()).toBe(true);
      expect(wrapper.find(".log-filter").exists()).toBe(true);
    });

    test("shows Start Streaming button when not streaming", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");
      expect(wrapper.find(".btn-primary").text()).toBe("Start Streaming");
    });
  });

  describe("Placeholder", () => {
    test("shows placeholder when no logs and no preview", async () => {
      // Resolve the initial call first (returns empty, so no previews)
      await waitForInitialCall();

      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");
      await wrapper.vm.$nextTick();

      expect(wrapper.find(".log-placeholder").exists()).toBe(true);
      expect(wrapper.find(".placeholder-title").text()).toBe("No logs yet");
      expect(wrapper.find(".placeholder-hint").exists()).toBe(true);
    });

    test("shows preview logs when previewLogs is populated", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");

      // Let the initial call complete first (sets previewLogs to [] from empty response).
      await waitForInitialCall();

      // Access the composable's previewLogs via the component proxy
      (wrapper.vm as any).previewLogs = [
        {
          timestamp: "10:00:00 AM",
          level: "INFO",
          message: "Application started",
        },
      ];
      await wrapper.vm.$nextTick();

      expect(wrapper.find(".log-preview").exists()).toBe(true);
      expect(wrapper.find(".preview-badge").text()).toBe("PREVIEW");
      expect(wrapper.find(".preview-source").text()).toBe("logs/app.log");
      expect(wrapper.text()).toContain("Application started");
    });

    test("preview hides when live logs arrive", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");

      // Let the initial call complete first
      await waitForInitialCall();

      // Set preview logs
      (wrapper.vm as any).previewLogs = [
        {
          timestamp: "10:00:00 AM",
          level: "INFO",
          message: "Preview entry",
        },
      ];
      await wrapper.vm.$nextTick();
      expect(wrapper.find(".log-preview").exists()).toBe(true);

      // Add a live log entry via the exposed composable's addLogEntry method
      (wrapper.vm as any).addLogEntry({
        type: "log",
        content: { msg: "Live log entry", level: "info" },
      });
      await wrapper.vm.$nextTick();

      expect(wrapper.find(".log-preview").exists()).toBe(false);
    });
  });

  describe("Clear button", () => {
    test("clear button resets both logs and previewLogs", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");

      // Add a preview log before resolving call
      (wrapper.vm as any).previewLogs = [
        { timestamp: "10:00:00 AM", level: "INFO", message: "Preview" },
      ];
      await waitForInitialCall();

      // Add a live log via the exposed composable method
      (wrapper.vm as any).addLogEntry({
        type: "log",
        content: { msg: "Live log", level: "info" },
      });
      await wrapper.vm.$nextTick();

      // Click Clear
      await wrapper.find(".btn-secondary").trigger("click");
      await wrapper.vm.$nextTick();

      // Both should be cleared — placeholder should show
      expect(wrapper.find(".log-placeholder").exists()).toBe(true);
      expect(wrapper.find(".log-preview").exists()).toBe(false);
    });
  });

  describe("Log level filter", () => {
    test("filtering by level shows only matching logs", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");
      await waitForInitialCall();

      const vm = wrapper.vm as any;
      vm.addLogEntry({
        type: "log",
        content: { msg: "Info message", level: "info" },
      });
      vm.addLogEntry({
        type: "log",
        content: { msg: "Error message", level: "error" },
      });
      vm.addLogEntry({
        type: "log",
        content: { msg: "Warning message", level: "warn" },
      });
      await wrapper.vm.$nextTick();

      vm.logLevelFilter = "error";
      await wrapper.vm.$nextTick();

      const logEntries = wrapper.findAll(".log-entry");
      expect(logEntries.length).toBe(1);
      expect(logEntries[0].text()).toContain("Error message");
    });

    test("filtering by 'All Levels' shows all logs", async () => {
      const wrapper = createWrapper();
      await wrapper.find(".log-toggle-button").trigger("click");
      await waitForInitialCall();

      const vm = wrapper.vm as any;
      vm.addLogEntry({
        type: "log",
        content: { msg: "Info message", level: "info" },
      });
      vm.addLogEntry({
        type: "log",
        content: { msg: "Error message", level: "error" },
      });
      await wrapper.vm.$nextTick();

      vm.logLevelFilter = "";
      await wrapper.vm.$nextTick();

      const logEntries = wrapper.findAll(".log-entry");
      expect(logEntries.length).toBe(2);
    });
  });

  describe("Log parsing (via composable)", () => {
    test("parseLogEntry handles structured JSON log", () => {
      const result = parseLogEntry({
        type: "log",
        content: {
          msg: "Test message",
          level: "info",
          time: "2024-01-01T00:00:00Z",
        },
      });

      expect(result).not.toBeNull();
      expect(result!.message).toBe("Test message");
      expect(result!.level).toBe("INFO");
    });

    test("parseLogEntry skips entries with 'Skipping' in message", () => {
      const result = parseLogEntry({
        type: "log",
        content: { msg: "Skipping hidden directory" },
      });

      expect(result).toBeNull();
    });

    test("parseLogEntry handles plain text content", () => {
      const result = parseLogEntry({
        type: "log",
        content: "Plain text log line",
      });

      expect(result).not.toBeNull();
      expect(result!.message).toBe("Plain text log line");
    });

    test("parseLogEntry handles missing content gracefully", () => {
      const result = parseLogEntry({
        type: "log",
        content: null,
      });

      expect(result).not.toBeNull();
      expect(result!.message).toContain("Received log event without content");
    });
  });
});
