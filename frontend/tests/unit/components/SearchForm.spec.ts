import { vi } from "vitest";
import { mount } from '@vue/test-utils';
import SearchForm from '../../../src/components/ui/SearchForm.vue';

// Mock the SearchState data
const mockData = {
  directory: '',
  query: '',
  extension: '',
  caseSensitive: false,
  useRegex: false,
  includeBinary: false,
  maxFileSize: 10485760,
  maxResults: 1000,
  searchSubdirs: true,
  resultText: 'Please enter search parameters below 👇',
  searchResults: [],
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
  availableEditors: [],
  editorDetectionStatus: {
    detectingEditors: false,
    detectionComplete: false,
    message: '',
    detectionProgress: 0,
    detectedEditors: []
  },
  knownTextExtensions: [],
  directories: [],
};

const mockSearchCode = vi.fn();
const mockSelectDirectory = vi.fn();
const mockCancelSearch = vi.fn();

describe('SearchForm.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('renders all modular child components', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    expect(wrapper.findComponent({ name: 'EditorStatusDisplay' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'DirectoryPicker' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'QueryInput' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'SearchOptions' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'PatternSelector' }).exists()).toBe(true);
    expect(wrapper.findComponent({ name: 'ActionButtons' }).exists()).toBe(true);
  });

  test('passes editor detection status to EditorStatusDisplay', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const editorStatus = wrapper.findComponent({ name: 'EditorStatusDisplay' });
    expect(editorStatus.props('editorDetectionStatus')).toEqual(mockData.editorDetectionStatus);
  });

  test('passes isSearching to ActionButtons', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: { ...mockData, isSearching: true },
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const actionButtons = wrapper.findComponent({ name: 'ActionButtons' });
    expect(actionButtons.props('isSearching')).toBe(true);
  });

  test('disables child components when searching', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: { ...mockData, isSearching: true },
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const directoryPicker = wrapper.findComponent({ name: 'DirectoryPicker' });
    expect(directoryPicker.props('disabled')).toBe(true);

    const queryInput = wrapper.findComponent({ name: 'QueryInput' });
    expect(queryInput.props('disabled')).toBe(true);
  });

  test('has main container styling', () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    expect(wrapper.find('.search-form').exists()).toBe(true);
  });

  test('calls searchCode when ActionButtons emits search', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const actionButtons = wrapper.findComponent({ name: 'ActionButtons' });
    actionButtons.vm.$emit('search');
    await wrapper.vm.$nextTick();

    expect(mockSearchCode).toHaveBeenCalled();
  });

  test('calls cancelSearch when ActionButtons emits cancel', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: { ...mockData, isSearching: true },
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const actionButtons = wrapper.findComponent({ name: 'ActionButtons' });
    actionButtons.vm.$emit('cancel');
    await wrapper.vm.$nextTick();

    expect(mockCancelSearch).toHaveBeenCalled();
  });

  test('calls selectDirectory when DirectoryPicker emits select', async () => {
    const wrapper = mount(SearchForm, {
      props: {
        data: mockData,
        searchCode: mockSearchCode,
        selectDirectory: mockSelectDirectory,
        cancelSearch: mockCancelSearch
      }
    });

    const directoryPicker = wrapper.findComponent({ name: 'DirectoryPicker' });
    directoryPicker.vm.$emit('select', '/some/path');
    await wrapper.vm.$nextTick();

    expect(mockSelectDirectory).toHaveBeenCalled();
  });
});
