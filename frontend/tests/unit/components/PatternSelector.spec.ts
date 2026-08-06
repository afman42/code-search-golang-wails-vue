import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import { PatternSelector } from '@/components/ui';

describe('PatternSelector.vue', () => {
  it('renders with empty patterns', () => {
    const wrapper = mount(PatternSelector);

    expect(wrapper.find('.pattern-group').exists()).toBe(true);
    expect(wrapper.findAll('.pattern-select').length).toBe(2);
  });

  it('displays exclude patterns as tags', () => {
    const wrapper = mount(PatternSelector, {
      props: {
        excludePatterns: ['node_modules', '.git'],
        allowedFileTypes: [],
      },
    });

    expect(wrapper.findAll('.pattern-tag').length).toBe(2);
    expect(wrapper.text()).toContain('node_modules');
    expect(wrapper.text()).toContain('.git');
  });

  it('displays allowed file types as tags', () => {
    const wrapper = mount(PatternSelector, {
      props: {
        excludePatterns: [],
        allowedFileTypes: ['.go', '.ts'],
      },
    });

    expect(wrapper.findAll('.pattern-tag').length).toBe(2);
    expect(wrapper.text()).toContain('.go');
    expect(wrapper.text()).toContain('.ts');
  });

  it('emits removePattern when remove button clicked', async () => {
    const wrapper = mount(PatternSelector, {
      props: {
        excludePatterns: ['node_modules', '.git', 'dist'],
        allowedFileTypes: [],
      },
    });

    const removeBtn = wrapper.findAll('.remove-btn')[1];
    await removeBtn.trigger('click');

    expect(wrapper.emitted('removePattern')).toBeTruthy();
    expect(wrapper.emitted('removePattern')![0]).toEqual(['exclude', 1]);
  });

  it('emits removePattern for allowed types', async () => {
    const wrapper = mount(PatternSelector, {
      props: {
        excludePatterns: [],
        allowedFileTypes: ['.go', '.ts', '.js'],
      },
    });

    const removeBtn = wrapper.findAll('.remove-btn')[1];
    await removeBtn.trigger('click');

    expect(wrapper.emitted('removePattern')).toBeTruthy();
    expect(wrapper.emitted('removePattern')![0]).toEqual(['allow', 1]);
  });

  it('emits update when custom pattern added on enter', async () => {
    const wrapper = mount(PatternSelector);

    const input = wrapper.find('input[placeholder="Custom pattern..."]');
    await input.setValue('my_custom_dir');
    await input.trigger('keyup.enter');

    expect(wrapper.emitted('update')).toBeTruthy();
  });

  it('updates UI when props change', async () => {
    const wrapper = mount(PatternSelector, {
      props: {
        excludePatterns: [],
        allowedFileTypes: [],
      },
    });

    await wrapper.setProps({
      excludePatterns: ['new_pattern'],
      allowedFileTypes: ['.new'],
    });

    expect(wrapper.findAll('.pattern-tag').length).toBe(2);
  });

  it('has available options in selects', () => {
    const wrapper = mount(PatternSelector);

    const selects = wrapper.findAll('.pattern-select');
    expect(selects.length).toBeGreaterThanOrEqual(2);
    
    const options = wrapper.findAll('option');
    expect(options.length).toBeGreaterThan(0);
  });
});
