import { mount } from '@vue/test-utils';
import EnhancedTreeItem from '../src/components/ui/EnhancedTreeItem.vue';
import { describe, it, expect, beforeEach, afterEach } from '@jest/globals';

// Define the TreeItem interface to match the component
interface TreeItem {
  name: string;
  path: string;
  children?: TreeItem[];
  isFile?: boolean;
  isExpanded?: boolean;
}

describe('EnhancedTreeItem', () => {
  let mockTreeItem: TreeItem;

  beforeEach(() => {
    mockTreeItem = {
      name: 'src',
      path: '/project/src',
      isFile: false,
      children: [
        {
          name: 'index.js',
          path: '/project/src/index.js',
          isFile: true,
        },
        {
          name: 'utils',
          path: '/project/src/utils',
          isFile: false,
          children: [
            {
              name: 'helper.js',
              path: '/project/src/utils/helper.js',
              isFile: true,
            }
          ]
        }
      ]
    };
  });

  afterEach(() => {
    // Clean up any potential cache or state
  });

  describe('Rendering', () => {
    it('renders a folder when item is not a file', () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
        }
      });

      expect(wrapper.find('.tree-item-header').classes()).toContain('is-folder');
      expect(wrapper.find('.tree-item-header').classes()).not.toContain('is-file');
      expect(wrapper.find('.tree-item-name').text()).toContain('src');
    });

    it('renders a file when item is a file', () => {
      const fileItem = {
        name: 'test.js',
        path: '/project/test.js',
        isFile: true
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: fileItem,
          currentFilePath: '',
        }
      });

      expect(wrapper.find('.tree-item-header').classes()).toContain('is-file');
      expect(wrapper.find('.tree-item-header').classes()).not.toContain('is-folder');
      expect(wrapper.find('.tree-item-name').text()).toContain('test.js');
    });

    it('highlights current file when path matches', () => {
      const fileItem = {
        name: 'test.js',
        path: '/project/test.js',
        isFile: true
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: fileItem,
          currentFilePath: '/project/test.js',
        }
      });

      expect(wrapper.find('.tree-item-header').classes()).toContain('current-file');
    });

    it('shows item count when showItemCount is true and item has children', () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          showItemCount: true,
        }
      });

      expect(wrapper.find('.item-count').exists()).toBe(true);
      expect(wrapper.find('.item-count').text()).toBe('(2)'); // 2 items in children array
    });

    it('does not show item count when showItemCount is false', () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          showItemCount: false,
        }
      });

      expect(wrapper.find('.item-count').exists()).toBe(false);
    });

    it('does not show item count for files', () => {
      const fileItem = {
        name: 'test.js',
        path: '/project/test.js',
        isFile: true
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: fileItem,
          currentFilePath: '',
          showItemCount: true,
        }
      });

      expect(wrapper.find('.item-count').exists()).toBe(false);
    });
  });

  describe('Expansion behavior', () => {
    it('expands when toggle is clicked', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
        }
      });

      // Initially collapsed
      expect(wrapper.find('.tree-item-children').exists()).toBe(false);

      // Click the toggle
      await wrapper.find('.tree-item-toggle').trigger('click');
      
      // Should now be expanded
      expect(wrapper.find('.tree-item-children').exists()).toBe(true);
    });

    it('does not expand if item is a file', async () => {
      const fileItem = {
        name: 'test.js',
        path: '/project/test.js',
        isFile: true
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: fileItem,
          currentFilePath: '',
        }
      });

      // No toggle should exist for files
      expect(wrapper.find('.tree-item-toggle').exists()).toBe(false);
    });

    it('respects global expanded prop', () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          expanded: true,
        }
      });

      // Should be expanded due to global prop
      expect(wrapper.find('.tree-item-children').exists()).toBe(true);
    });

    it('individual override takes precedence after user interaction', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          expanded: false, // Global state is collapsed
        }
      });

      // Initially collapsed due to global prop
      expect(wrapper.find('.tree-item-children').exists()).toBe(false);

      // User expands the item
      await wrapper.find('.tree-item-toggle').trigger('click');

      // Should be expanded despite global prop being false
      expect(wrapper.find('.tree-item-children').exists()).toBe(true);
    });
  });

  describe('Filtering behavior', () => {
    it('filters children based on filter text', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          filterText: 'index',
        }
      });

      // Click to expand (since it starts collapsed)
      await wrapper.find('.tree-item-toggle').trigger('click');

      // Should only show the child that matches the filter
      const children = wrapper.findAllComponents(EnhancedTreeItem);
      expect(children).toHaveLength(1); // Only index.js should match
      expect(children[0].props('item').name).toBe('index.js');
    });

    it('shows "No matching items" when filter yields no results', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          filterText: 'nonexistent',
        }
      });

      // Click to expand
      await wrapper.find('.tree-item-toggle').trigger('click');

      expect(wrapper.find('.no-results').exists()).toBe(true);
      expect(wrapper.find('.no-results').text()).toBe('No matching items');
    });

    it('shows parent if a descendant matches the filter', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          filterText: 'helper',
        }
      });

      // Click to expand
      await wrapper.find('.tree-item-toggle').trigger('click');

      // Parent should be visible because it has a matching descendant
      const children = wrapper.findAllComponents(EnhancedTreeItem);
      expect(children).toHaveLength(1); // Should show the 'utils' folder that contains 'helper.js'
      expect(children[0].props('item').name).toBe('utils');
    });

    it('clears cache when filter text changes', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          filterText: 'old-filter',
        }
      });

      // Initially with old filter
      await wrapper.setProps({ filterText: 'helper' });
      
      // Should now match the helper file
      await wrapper.find('.tree-item-toggle').trigger('click');
      
      const children = wrapper.findAllComponents(EnhancedTreeItem);
      expect(children).toHaveLength(1); // Should show the 'utils' folder
      expect(children[0].props('item').name).toBe('utils');
    });
  });

  describe('File click behavior', () => {
    it('emits file-click event when file is clicked', async () => {
      const fileItem = {
        name: 'test.js',
        path: '/project/test.js',
        isFile: true
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: fileItem,
          currentFilePath: '',
        }
      });

      await wrapper.find('.tree-item-header').trigger('click');
      
      expect(wrapper.emitted('file-click')).toBeTruthy();
      expect(wrapper.emitted('file-click')![0]).toEqual(['/project/test.js']);
    });

    it('does not emit file-click when folder is clicked', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
        }
      });

      await wrapper.find('.tree-item-header').trigger('click');
      
      expect(wrapper.emitted('file-click')).toBeUndefined();
    });
  });

  describe('Edge cases', () => {
    it('handles item with no children gracefully', () => {
      const itemWithoutChildren = {
        name: 'empty-folder',
        path: '/project/empty',
        isFile: false,
        children: []
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: itemWithoutChildren,
          currentFilePath: '',
        }
      });

      expect(wrapper.find('.tree-item-toggle').exists()).toBe(false);
      expect(wrapper.find('.tree-item-children').exists()).toBe(false);
    });

    it('handles item with null/undefined children', () => {
      const itemWithoutChildren = {
        name: 'no-children',
        path: '/project/no-children',
        isFile: false,
        // children property is intentionally omitted
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: itemWithoutChildren,
          currentFilePath: '',
        }
      });

      expect(wrapper.find('.tree-item-toggle').exists()).toBe(false);
      expect(wrapper.find('.tree-item-children').exists()).toBe(false);
    });

    it('handles deeply nested tree structure', () => {
      const deepTree: TreeItem = {
        name: 'level0',
        path: '/level0',
        isFile: false,
        children: [
          {
            name: 'level1',
            path: '/level0/level1',
            isFile: false,
            children: [
              {
                name: 'level2',
                path: '/level0/level1/level2',
                isFile: false,
                children: [
                  {
                    name: 'deep-file.txt',
                    path: '/level0/level1/level2/deep-file.txt',
                    isFile: true
                  }
                ]
              }
            ]
          }
        ]
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: deepTree,
          currentFilePath: '',
        }
      });

      expect(wrapper.find('.tree-item-name').text()).toContain('level0');
      expect(wrapper.find('.tree-item-toggle').exists()).toBe(true);
    });

    it('handles empty filter text', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          filterText: '',
        }
      });

      await wrapper.find('.tree-item-toggle').trigger('click');
      
      // Should show all children when filter is empty
      const children = wrapper.findAllComponents(EnhancedTreeItem);
      expect(children).toHaveLength(2); // index.js and utils folder
    });

    it('handles whitespace-only filter text', async () => {
      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: mockTreeItem,
          currentFilePath: '',
          filterText: '   ', // whitespace only
        }
      });

      await wrapper.find('.tree-item-toggle').trigger('click');
      
      // Should treat whitespace-only filter as empty
      const children = wrapper.findAllComponents(EnhancedTreeItem);
      expect(children).toHaveLength(2); // Should show all children
    });

    it('handles special characters in filter', async () => {
      const specialTree = {
        name: 'special[folder]',
        path: '/special[folder]',
        isFile: false,
        children: [
          {
            name: 'test[file].js',  // This should match the filter '[file]'
            path: '/special[folder]/test[file].js',
            isFile: true
          }
        ]
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: specialTree,
          currentFilePath: '',
          filterText: '[file]',  // Looking for files that contain [file] in their name
        }
      });

      await wrapper.find('.tree-item-toggle').trigger('click');
      
      const children = wrapper.findAllComponents(EnhancedTreeItem);
      expect(children).toHaveLength(1);
      expect(children[0].props('item').name).toBe('test[file].js');
    });

    it('properly handles case sensitivity in filtering', async () => {
      const caseTree = {
        name: 'mixedCase',
        path: '/mixedCase',
        isFile: false,
        children: [
          {
            name: 'LowerCase.js',
            path: '/mixedCase/LowerCase.js',
            isFile: true
          }
        ]
      };

      const wrapper = mount(EnhancedTreeItem, {
        props: {
          item: caseTree,
          currentFilePath: '',
          filterText: 'lowercase', // lowercase version
        }
      });

      await wrapper.find('.tree-item-toggle').trigger('click');
      
      // Filter is case-insensitive, so it should match
      const children = wrapper.findAllComponents(EnhancedTreeItem);
      expect(children).toHaveLength(1);
    });
  });
});