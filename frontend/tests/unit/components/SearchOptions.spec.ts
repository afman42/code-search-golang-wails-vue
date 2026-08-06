import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import { SearchOptions } from '@/components/ui';

describe('SearchOptions.vue', () => {
  it('renders with default values', () => {
    const wrapper = mount(SearchOptions);
    
    expect(wrapper.find('.options-group').exists()).toBe(true);
    expect(wrapper.findAll('.checkbox-group').length).toBe(4);
  });

  it('shows all 4 checkboxes', async () => {
    const wrapper = mount(SearchOptions);
    
    expect(wrapper.find('#case-sensitive').exists()).toBe(true);
    expect(wrapper.find('#regex-search').exists()).toBe(true);
    expect(wrapper.find('#include-binary').exists()).toBe(true);
    expect(wrapper.find('#fuzzy-search').exists()).toBe(true);
  });

  it('emits update event when case sensitive changes', async () => {
    const wrapper = mount(SearchOptions);
    const checkbox = wrapper.find('#case-sensitive');
    
    await checkbox.setValue(true);
    
    expect(wrapper.emitted('update')).toBeTruthy();
    const emittedValue = wrapper.emitted('update')![0];
    expect(emittedValue[0]).toMatchObject({ 
      caseSensitive: true,
      useRegex: false,
      includeBinary: false,
      fuzzySearch: false,
    });
  });

  it('emits update event when regex changes', async () => {
    const wrapper = mount(SearchOptions);
    const checkbox = wrapper.find('#regex-search');
    
    await checkbox.setValue(true);
    
    expect(wrapper.emitted('update')).toBeTruthy();
    const emittedValue = wrapper.emitted('update')![0];
    expect(emittedValue[0].useRegex).toBe(true);
  });

  it('emits update event when include binary changes', async () => {
    const wrapper = mount(SearchOptions);
    const checkbox = wrapper.find('#include-binary');
    
    await checkbox.setValue(true);
    
    expect(wrapper.emitted('update')).toBeTruthy();
    const emittedValue = wrapper.emitted('update')![0];
    expect(emittedValue[0].includeBinary).toBe(true);
  });

  it('emits update event when fuzzy search changes', async () => {
    const wrapper = mount(SearchOptions);
    const checkbox = wrapper.find('#fuzzy-search');
    
    await checkbox.setValue(true);
    
    expect(wrapper.emitted('update')).toBeTruthy();
    const emittedValue = wrapper.emitted('update')![0];
    expect(emittedValue[0].fuzzySearch).toBe(true);
  });

  it('applies custom IDs if provided', () => {
    const wrapper = mount(SearchOptions, {
      props: { caseSensitiveId: 'custom-case' }
    });
    
    expect(wrapper.find('#custom-case').exists()).toBe(true);
  });

  it('disables inputs when disabled prop is true', () => {
    const wrapper = mount(SearchOptions, {
      props: { disabled: true }
    });
    
    const checkboxes = wrapper.findAll('input[type="checkbox"]');
    checkboxes.forEach(cb => expect(cb.attributes('disabled')).toBeDefined());
  });

  it('has correct styling classes', () => {
    const wrapper = mount(SearchOptions);
    
    expect(wrapper.find('.options-group').exists()).toBe(true);
    expect(wrapper.findAll('.checkbox-group').length).toBe(4);
  });
});
