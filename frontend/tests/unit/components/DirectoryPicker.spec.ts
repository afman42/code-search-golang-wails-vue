import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import DirectoryPicker from '@/components/ui/DirectoryPicker.vue';

describe('DirectoryPicker.vue', () => {
  it('renders with default props', () => {
    const wrapper = mount(DirectoryPicker, {
      props: { directory: '' },
    });

    expect(wrapper.find('label').text()).toBe('Directory:');
    expect(wrapper.find('input').attributes('id')).toBe('directory');
    expect(wrapper.find('input').attributes('placeholder')).toBe('Enter directory to search');
    expect(wrapper.find('.select-dir').text()).toBe('Browse');
  });

  it('renders with custom props', () => {
    const wrapper = mount(DirectoryPicker, {
      props: {
        id: 'customDir',
        label: 'Root Path:',
        placeholder: 'Select root path',
        buttonText: 'Select',
        directory: '/home/user/project',
      },
    });

    expect(wrapper.find('label').text()).toBe('Root Path:');
    expect(wrapper.find('input').attributes('id')).toBe('customDir');
    expect(wrapper.find('input').attributes('placeholder')).toBe('Select root path');
    expect(wrapper.find('.select-dir').text()).toBe('Select');
    expect(wrapper.find('input').element.value).toBe('/home/user/project');
  });

  it('emits select event on browse button click', async () => {
    const wrapper = mount(DirectoryPicker, {
      props: { directory: '' },
    });

    const input = wrapper.find('input');
    await input.setValue('/some/path');
    
    const browseButton = wrapper.find('.select-dir');
    await browseButton.trigger('click');

    expect(wrapper.emitted('select')).toBeTruthy();
    expect(wrapper.emitted('select')![0]).toEqual(['/some/path']);
  });

  it('updates local value when parent prop changes', async () => {
    const wrapper = mount(DirectoryPicker, {
      props: { directory: '/initial' },
    });

    expect(wrapper.find('input').element.value).toBe('/initial');

    await wrapper.setProps({ directory: '/updated/path' });

    expect(wrapper.find('input').element.value).toBe('/updated/path');
  });

  it('emits update event when input changes', async () => {
    const wrapper = mount(DirectoryPicker, {
      props: { directory: '' },
    });

    const input = wrapper.find('input');
    await input.setValue('/new/path');

    expect(wrapper.emitted('update')).toBeTruthy();
    expect(wrapper.emitted('update')![0]).toEqual(['/new/path']);
  });

  it('disables input and button when disabled prop is true', async () => {
    const wrapper = mount(DirectoryPicker, {
      props: { directory: '', disabled: true },
    });

    expect(wrapper.find('input').element.disabled).toBe(true);
    expect(wrapper.find('.select-dir').element.disabled).toBe(true);
  });

  it('has correct styling classes', () => {
    const wrapper = mount(DirectoryPicker, {
      props: { directory: '' },
    });

    expect(wrapper.find('.control-group').exists()).toBe(true);
    expect(wrapper.find('.directory-input').exists()).toBe(true);
    expect(wrapper.find('input').classes()).toContain('input');
    expect(wrapper.find('input').classes()).toContain('directory');
  });
});
