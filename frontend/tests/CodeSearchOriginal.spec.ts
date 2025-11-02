import { mount, flushPromises } from '@vue/test-utils';
import CodeSearch from '@/components/CodeSearch.vue';
import { SearchCode } from '../../wailsjs/go/main/App';

describe('CodeSearch.vue', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders search controls properly', () => {
    const wrapper = mount(CodeSearch);
    
    // Check that the search controls exist
    expect(wrapper.find('input#directory').exists()).toBe(true);
    expect(wrapper.find('input#query').exists()).toBe(true);
    expect(wrapper.find('input#extension').exists()).toBe(true);
    expect(wrapper.find('input#case-sensitive').exists()).toBe(true);
    expect(wrapper.find('button.search-btn').exists()).toBe(true);
  });

  test('allows user to input directory, query, and extension', async () => {
    const wrapper = mount(CodeSearch);
    
    const directoryInput = wrapper.find('input#directory');
    const queryInput = wrapper.find('input#query');
    const extensionInput = wrapper.find('input#extension');
    
    await directoryInput.setValue('/test/directory');
    await queryInput.setValue('testQuery');
    await extensionInput.setValue('go');
    
    expect(directoryInput.element.value).toBe('/test/directory');
    expect(queryInput.element.value).toBe('testQuery');
    expect(extensionInput.element.value).toBe('go');
  });

  test('allows toggling case sensitivity', async () => {
    const wrapper = mount(CodeSearch);
    
    const checkbox = wrapper.find('input#case-sensitive');
    
    // Initially unchecked
    expect(checkbox.element.checked).toBe(false);
    
    // Check the checkbox
    await checkbox.setChecked(true);
    expect(checkbox.element.checked).toBe(true);
    
    // Uncheck the checkbox
    await checkbox.setChecked(false);
    expect(checkbox.element.checked).toBe(false);
  });

  test('calls SearchCode when search button is clicked with valid inputs', async () => {
    const mockResults = [
      {
        filePath: '/test/file.go',
        lineNum: 5,
        content: 'fmt.Println("hello world")'
      }
    ];
    SearchCode.mockResolvedValue(mockResults);
    
    const wrapper = mount(CodeSearch);
    
    // Fill in the search parameters
    await wrapper.find('input#directory').setValue('/test/directory');
    await wrapper.find('input#query').setValue('hello');
    await wrapper.find('input#extension').setValue('go');
    await wrapper.find('input#case-sensitive').setChecked(false);
    
    // Click the search button
    await wrapper.find('button.search-btn').trigger('click');
    
    // Check that SearchCode was called with the correct parameters
    expect(SearchCode).toHaveBeenCalledWith({
      directory: '/test/directory',
      query: 'hello',
      extension: 'go',
      caseSensitive: false
    });
  });

  test('shows error message when directory or query is missing', async () => {
    const wrapper = mount(CodeSearch);
    
    // Don't fill in directory and query
    await wrapper.find('button.search-btn').trigger('click');
    
    // Check that error message is shown
    await flushPromises();
    expect(wrapper.text()).toContain('Please specify both directory and search query');
  });

  test('shows results after successful search', async () => {
    const mockResults = [
      {
        filePath: '/test/file1.go',
        lineNum: 5,
        content: 'fmt.Println("hello")'
      },
      {
        filePath: '/test/file2.go',
        lineNum: 10,
        content: 'fmt.Println("world")'
      }
    ];
    SearchCode.mockResolvedValue(mockResults);
    
    const wrapper = mount(CodeSearch);
    
    // Fill in the search parameters
    await wrapper.find('input#directory').setValue('/test/directory');
    await wrapper.find('input#query').setValue('fmt');
    
    // Click the search button
    await wrapper.find('button.search-btn').trigger('click');
    
    // Wait for the async operation to complete
    await flushPromises();
    
    // Check that results are displayed
    expect(wrapper.text()).toContain('Found 2 matches');
    expect(wrapper.findAll('.result-item')).toHaveLength(2);
    expect(wrapper.text()).toContain('/test/file1.go');
    expect(wrapper.text()).toContain('/test/file2.go');
  });

  test('shows error message when search fails', async () => {
    SearchCode.mockRejectedValue(new Error('Search failed'));
    
    const wrapper = mount(CodeSearch);
    
    // Fill in the search parameters
    await wrapper.find('input#directory').setValue('/test/directory');
    await wrapper.find('input#query').setValue('test');
    
    // Click the search button
    await wrapper.find('button.search-btn').trigger('click');
    
    // Wait for the async operation to complete
    await flushPromises();
    
    // Check that error message is shown
    expect(wrapper.text()).toContain('Error: Search failed');
  });

  test('disables search button during search', async () => {
    // Make the search call take some time to simulate loading
    SearchCode.mockImplementation(() => new Promise(resolve => {
      setTimeout(() => resolve([]), 100);
    }));
    
    const wrapper = mount(CodeSearch);
    
    // Fill in the search parameters
    await wrapper.find('input#directory').setValue('/test/directory');
    await wrapper.find('input#query').setValue('test');
    
    // Click the search button
    const searchButton = wrapper.find('button.search-btn');
    await searchButton.trigger('click');
    
    // Check that the button is disabled
    expect(searchButton.element.disabled).toBe(true);
  });

  test('handles search with no results', async () => {
    SearchCode.mockResolvedValue([]);
    
    const wrapper = mount(CodeSearch);
    
    // Fill in the search parameters
    await wrapper.find('input#directory').setValue('/test/directory');
    await wrapper.find('input#query').setValue('nonexistent');
    
    // Click the search button
    await wrapper.find('button.search-btn').trigger('click');
    
    // Wait for the async operation to complete
    await flushPromises();
    
    // Check that "no results" message is shown
    expect(wrapper.text()).toContain('No matches found');
    expect(wrapper.findAll('.result-item')).toHaveLength(0);
  });
});