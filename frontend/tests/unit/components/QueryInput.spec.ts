import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import QueryInput from '@/components/ui/QueryInput.vue';

describe('QueryInput.vue', () => {
  it('renders with default props', () => {
    const wrapper = mount(QueryInput, {
      props: { query: '' },
    });

    expect(wrapper.find('label').text()).toBe('Search Query:');
    expect(wrapper.find('input').attributes('id')).toBe('query');
    expect(wrapper.find('input').attributes('placeholder')).toBe('Enter search term');
  });

  it('renders with custom props', () => {
    const wrapper = mount(QueryInput, {
      props: {
        id: 'customQuery',
        label: 'Find:',
        placeholder: 'Type something...',
        query: 'test value',
      },
    });

    expect(wrapper.find('label').text()).toBe('Find:');
    expect(wrapper.find('input').element.value).toBe('test value');
  });

  it('emits search event on enter key', async () => {
    const wrapper = mount(QueryInput, {
      props: { query: '' },
    });

    await wrapper.find('input').trigger('keyup.enter');

    expect(wrapper.emitted('search')).toBeTruthy();
  });

  it('emits update event when input changes', async () => {
    const wrapper = mount(QueryInput, {
      props: { query: '' },
    });

    await wrapper.find('input').setValue('new query');

    expect(wrapper.emitted('update')).toBeTruthy();
    expect(wrapper.emitted('update')![0]).toEqual(['new query']);
  });

  it('updates local value when parent prop changes', async () => {
    const wrapper = mount(QueryInput, {
      props: { query: '/initial' },
    });

    expect(wrapper.find('input').element.value).toBe('/initial');

    await wrapper.setProps({ query: '/updated/path' });

    expect(wrapper.find('input').element.value).toBe('/updated/path');
  });

  it('disables input when disabled prop is true', () => {
    const wrapper = mount(QueryInput, {
      props: { query: '', disabled: true },
    });

    expect(wrapper.find('input').element.disabled).toBe(true);
  });

  it('emits focus and blur events', async () => {
    const wrapper = mount(QueryInput, {
      props: { query: '' },
    });

    const input = wrapper.find('input');
    await input.trigger('focus');
    await input.trigger('blur');

    expect(wrapper.emitted('focus')).toBeTruthy();
    expect(wrapper.emitted('blur')).toBeTruthy();
  });

  it('has correct styling classes', () => {
    const wrapper = mount(QueryInput, {
      props: { query: '' },
    });

    expect(wrapper.find('input').classes()).toContain('input');
  });
});
