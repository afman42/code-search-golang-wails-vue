import { describe, it, expect, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { ActionButtons } from '@/components/ui';

describe('ActionButtons.vue', () => {
  it('renders search button when not searching', () => {
    const wrapper = mount(ActionButtons);

    expect(wrapper.find('.btn-search').exists()).toBe(true);
    expect(wrapper.find('.btn-cancel').exists()).toBe(false);
    expect(wrapper.text()).toContain('Search Code');
  });

  it('renders cancel button when searching', () => {
    const wrapper = mount(ActionButtons, {
      props: { isSearching: true },
    });

    expect(wrapper.find('.btn-search').exists()).toBe(false);
    expect(wrapper.find('.btn-cancel').exists()).toBe(true);
    expect(wrapper.text()).toContain('Cancel Search');
  });

  it('emits search event on search button click', async () => {
    const wrapper = mount(ActionButtons);

    await wrapper.find('.btn-search').trigger('click');

    expect(wrapper.emitted('search')).toBeTruthy();
  });

  it('emits cancel event on cancel button click', async () => {
    const wrapper = mount(ActionButtons, {
      props: { isSearching: true },
    });

    await wrapper.find('.btn-cancel').trigger('click');

    expect(wrapper.emitted('cancel')).toBeTruthy();
  });

  it('disables search button when disabled prop is true', () => {
    const wrapper = mount(ActionButtons, {
      props: { disabled: true },
    });

    expect(wrapper.find('.btn-search').element.disabled).toBe(true);
  });

  it('has correct styling classes', () => {
    const wrapper = mount(ActionButtons);

    expect(wrapper.find('.btn-primary').exists()).toBe(true);
    expect(wrapper.find('.btn-secondary').exists()).toBe(false);
  });

  it('switches to cancel mode correctly', async () => {
    const wrapper = mount(ActionButtons);

    // Initial state
    expect(wrapper.find('.btn-search').exists()).toBe(true);

    // Change to searching state
    await wrapper.setProps({ isSearching: true });

    expect(wrapper.find('.btn-search').exists()).toBe(false);
    expect(wrapper.find('.btn-cancel').exists()).toBe(true);
  });

  it('handles rapid state changes', async () => {
    const wrapper = mount(ActionButtons);

    await wrapper.setProps({ isSearching: true });
    await wrapper.setProps({ isSearching: false });

    expect(wrapper.find('.btn-search').exists()).toBe(true);
    expect(wrapper.find('.btn-cancel').exists()).toBe(false);
  });
});
