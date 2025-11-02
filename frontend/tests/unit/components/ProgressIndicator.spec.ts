import { mount } from "@vue/test-utils";
import ProgressIndicator from "../../../src/components/ui/ProgressIndicator.vue";

// Mock the SearchState data with progress
const mockDataWithProgress = {
  directory: "",
  query: "",
  extension: "",
  caseSensitive: false,
  useRegex: false,
  includeBinary: false,
  maxFileSize: 10485760,
  maxResults: 1000,
  searchSubdirs: true,
  resultText: "Searching...",
  searchResults: [],
  truncatedResults: false,
  isSearching: true,
  searchProgress: {
    processedFiles: 50,
    totalFiles: 100,
    currentFile: "/test/file.go",
    resultsCount: 25,
    status: "in-progress",
  },
  showProgress: true,
  minFileSize: 0,
  excludePatterns: [],
  recentSearches: [],
  error: null,
};

const mockFormatFilePath = jest.fn((path: string) => path);

describe("ProgressIndicator.vue", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders progress bar when showProgress is true", () => {
    const wrapper = mount(ProgressIndicator, {
      props: {
        data: mockDataWithProgress,
        formatFilePath: mockFormatFilePath,
      },
    });

    // Check that the progress container exists
    expect(wrapper.find(".progress-container").exists()).toBe(true);

    // Check that the progress bar exists
    expect(wrapper.find(".progress-bar").exists()).toBe(true);

    // Check that the progress fill exists and has correct width
    const progressFill = wrapper.find(".progress-fill");
    expect(progressFill.exists()).toBe(true);
    expect(progressFill.attributes("style")).toContain("width: 50%");

    // Check that progress info exists
    expect(wrapper.find(".progress-info").exists()).toBe(true);
    expect(wrapper.text()).toContain("Processed: 50 / 100 files");
    expect(wrapper.text()).toContain("Results: 25");

    // Check that current file is displayed
    expect(wrapper.find(".current-file").exists()).toBe(true);
    expect(wrapper.text()).toContain("Processing: /test/file.go");
  });

  test("does not render when showProgress is false", () => {
    const mockDataWithoutProgress = {
      ...mockDataWithProgress,
      showProgress: false,
    };

    const wrapper = mount(ProgressIndicator, {
      props: {
        data: mockDataWithoutProgress,
        formatFilePath: mockFormatFilePath,
      },
    });

    // Check that the progress container does not exist
    expect(wrapper.find(".progress-container").exists()).toBe(false);
  });

  test("displays correct progress percentage", () => {
    const testData = { ...mockDataWithProgress };
    testData.searchProgress.processedFiles = 75;
    testData.searchProgress.totalFiles = 100;

    const wrapper = mount(ProgressIndicator, {
      props: {
        data: testData,
        formatFilePath: mockFormatFilePath,
      },
    });

    const progressFill = wrapper.find(".progress-fill");
    expect(progressFill.attributes("style")).toContain("width: 75%");
  });

  test("handles zero total files gracefully", () => {
    const testData = { ...mockDataWithProgress };
    testData.searchProgress.totalFiles = 0;

    const wrapper = mount(ProgressIndicator, {
      props: {
        data: testData,
        formatFilePath: mockFormatFilePath,
      },
    });

    const progressFill = wrapper.find(".progress-fill");
    expect(progressFill.attributes("style")).toContain("width: 0%");
  });
});
