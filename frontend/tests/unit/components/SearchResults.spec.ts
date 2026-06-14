import { vi } from "vitest";
import { mount } from '@vue/test-utils';
import SearchResults from '../../../src/components/ui/SearchResults.vue';
import {
  makeEditorAvailability,
  makeEditorDetectionStatus,
} from '../../fixtures/editorAvailability';

// Mock the SearchState data with results
const mockDataWithResults = {
  directory: '',
  query: 'test',
  extension: '',
  caseSensitive: false,
  useRegex: false,
  includeBinary: false,
  maxFileSize: 10485760,
  maxResults: 1000,
  searchSubdirs: true,
  resultText: 'Found 2 matches',
  searchResults: [
    {
      filePath: '/test/file1.go',
      lineNum: 5,
      content: 'fmt.Println("test message")',
      matchedText: 'test',
      contextBefore: ['package main', '', 'import "fmt"'],
      contextAfter: ['func main() {', '\tfmt.Println("another test")']
    },
    {
      filePath: '/test/file2.js',
      lineNum: 10,
      content: 'console.log("test");',
      matchedText: 'test',
      contextBefore: ['// This is a JS file', 'function testFunction() {'],
      contextAfter: ['\treturn true;', '}']
    }
  ],
  truncatedResults: false,
  isSearching: false,
  searchProgress: {
    processedFiles: 0,
    totalFiles: 0,
    currentFile: '',
    resultsCount: 0,
    status: ''
  },
  showProgress: false,
  minFileSize: 0,
  excludePatterns: [],
  recentSearches: [],
  error: null,
  availableEditors: makeEditorAvailability(),
  editorDetectionStatus: makeEditorDetectionStatus(),
};

const mockFormatFilePath = vi.fn((path: string) => path);
const mockHighlightMatch = vi.fn((text: string) => `<mark>${text}</mark>`);
const mockOpenFileLocation = vi.fn();
const mockCopyToClipboard = vi.fn();

describe('SearchResults.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders search results properly', () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        highlightMatch: mockHighlightMatch,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    // Check that results container exists
    expect(wrapper.find('.results-container').exists()).toBe(true);
    
    // Check that results header exists
    expect(wrapper.find('.results-header').exists()).toBe(true);
    expect(wrapper.text()).toContain('Search Results:');
    expect(wrapper.text()).toContain('Found 2 matches');
    
    // Check that result items exist
    const resultItems = wrapper.findAll('.result-item');
    expect(resultItems.length).toBe(2);
    
    // Check first result item
    const firstResult = resultItems[0];
    expect(firstResult.find('.file-path').text()).toBe('/test/file1.go');
    expect(firstResult.find('.line-num').text()).toBe('Line 5');
    expect(firstResult.find('.result-content').exists()).toBe(true);
    
    // Check context lines
    const contextLines = firstResult.findAll('.context-line');
    expect(contextLines.length).toBe(5); // 2 before + 2 after + 1 extra context line
    
    // Check second result item
    const secondResult = resultItems[1];
    expect(secondResult.find('.file-path').text()).toBe('/test/file2.js');
    expect(secondResult.find('.line-num').text()).toBe('Line 10');
  });

  test('shows truncated results message when applicable', () => {
    const truncatedData = { ...mockDataWithResults, truncatedResults: true };
    
    const wrapper = mount(SearchResults, {
      props: {
        data: truncatedData,
        formatFilePath: mockFormatFilePath,
        highlightMatch: mockHighlightMatch,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    expect(wrapper.text()).toContain('(truncated)');
  });

  test('does not render when no results', () => {
    const emptyData = { ...mockDataWithResults, searchResults: [] };
    
    const wrapper = mount(SearchResults, {
      props: {
        data: emptyData,
        formatFilePath: mockFormatFilePath,
        highlightMatch: mockHighlightMatch,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    expect(wrapper.find('.results-container').exists()).toBe(false);
  });

  test('calls openFileLocation when file path is clicked', async () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        highlightMatch: mockHighlightMatch,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    const filePath = wrapper.find('.file-path');
    await filePath.trigger('click');
    
    expect(mockOpenFileLocation).toHaveBeenCalledWith('/test/file1.go');
  });

  test('calls copyToClipboard when copy button is clicked', async () => {
    const wrapper = mount(SearchResults, {
      props: {
        data: mockDataWithResults,
        formatFilePath: mockFormatFilePath,
        highlightMatch: mockHighlightMatch,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard
      }
    });
    
    const copyButton = wrapper.find('.copy-btn');
    await copyButton.trigger('click');
    
    expect(mockCopyToClipboard).toHaveBeenCalledWith('fmt.Println("test message")');
  });

  test('only highlights the visible page, not all results', async () => {
    // Build 25 results so pagination (10/page) kicks in. Each result has one
    // content line plus one before/after context line = 3 highlight calls.
    const manyResults = Array.from({ length: 25 }, (_, i) => ({
      filePath: `/test/file${i}.go`,
      lineNum: i + 1,
      content: `line ${i} with test`,
      matchedText: 'test',
      contextBefore: [`before ${i}`],
      contextAfter: [`after ${i}`],
    }));

    const wrapper = mount(SearchResults, {
      props: {
        data: { ...mockDataWithResults, searchResults: manyResults },
        formatFilePath: mockFormatFilePath,
        highlightMatch: mockHighlightMatch,
        openFileLocation: mockOpenFileLocation,
        copyToClipboard: mockCopyToClipboard,
      },
    });

    // Only the first 10 results should be rendered...
    expect(wrapper.findAll('.result-item').length).toBe(10);

    // ...and highlighting should only run for those 10 (3 calls each = 30),
    // not for all 25 results (which would be 75 calls).
    expect(mockHighlightMatch).toHaveBeenCalledTimes(30);

    // Navigating to page 2 highlights that page's rows on demand.
    mockHighlightMatch.mockClear();
    // The top pagination controls have [Previous, Next] buttons; click Next.
    const navButtons = wrapper.findAll('.pagination-btn');
    await navButtons[1].trigger('click');
    await wrapper.vm.$nextTick();
    expect(mockHighlightMatch).toHaveBeenCalledTimes(30);
  });
});