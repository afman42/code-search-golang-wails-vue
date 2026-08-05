import { describe, it, expect, beforeEach } from 'vitest';
import { mount, type VueWrapper } from '@vue/test-utils';
import SizeLimitOptions from '@/components/ui/SizeLimitOptions.vue';

interface UpdatePayload {
  minFileSize: number;
  maxFileSize: number;
  maxResults: number;
  contextLines: number;
}

describe('SizeLimitOptions.vue', () => {
  // Helper: extract the payload of the last emitted 'update' event.
  const lastUpdate = (wrapper: VueWrapper): UpdatePayload | undefined => {
    const events = wrapper.emitted('update');
    if (!events || events.length === 0) return undefined;
    return events[events.length - 1][0] as {
      minFileSize: number;
      maxFileSize: number;
      maxResults: number;
      contextLines: number;
    };
  };

  beforeEach(() => {
    // Each test mounts fresh; nothing global to clear, but keep hook for parity.
  });

  it('renders all four inputs with default ids', () => {
    const wrapper = mount(SizeLimitOptions);
    expect(wrapper.find('#min-filesize').exists()).toBe(true);
    expect(wrapper.find('#max-filesize').exists()).toBe(true);
    expect(wrapper.find('#max-results').exists()).toBe(true);
    expect(wrapper.find('#context-lines').exists()).toBe(true);
  });

  it('initializes local refs from default values when props omitted', () => {
    const wrapper = mount(SizeLimitOptions);
    expect((wrapper.find('#min-filesize').element as HTMLInputElement).value).toBe('0');
    expect((wrapper.find('#max-filesize').element as HTMLInputElement).value).toBe('10485760');
    expect((wrapper.find('#max-results').element as HTMLInputElement).value).toBe('1000');
    expect((wrapper.find('#context-lines').element as HTMLInputElement).value).toBe('3');
  });

  it('initializes local refs from provided props', () => {
    const wrapper = mount(SizeLimitOptions, {
      props: {
        minFileSize: 100,
        maxFileSize: 2048,
        maxResults: 50,
        contextLines: 5,
      },
    });
    expect((wrapper.find('#min-filesize').element as HTMLInputElement).value).toBe('100');
    expect((wrapper.find('#max-filesize').element as HTMLInputElement).value).toBe('2048');
    expect((wrapper.find('#max-results').element as HTMLInputElement).value).toBe('50');
    expect((wrapper.find('#context-lines').element as HTMLInputElement).value).toBe('5');
  });

  it('uses custom ids when provided', () => {
    const wrapper = mount(SizeLimitOptions, {
      props: {
        minFileSizeId: 'custom-min',
        maxFileSizeId: 'custom-max',
        maxResultsId: 'custom-results',
        contextLinesId: 'custom-ctx',
      },
    });
    expect(wrapper.find('#custom-min').exists()).toBe(true);
    expect(wrapper.find('#custom-max').exists()).toBe(true);
    expect(wrapper.find('#custom-results').exists()).toBe(true);
    expect(wrapper.find('#custom-ctx').exists()).toBe(true);
    // default ids should NOT be present
    expect(wrapper.find('#min-filesize').exists()).toBe(false);
  });

  it('emits update when minFileSize changes', async () => {
    const wrapper = mount(SizeLimitOptions);
    wrapper.emitted(); // clear initial baseline by reading
    await wrapper.find('#min-filesize').setValue('42');
    const payload = lastUpdate(wrapper);
    expect(payload).toBeDefined();
    expect(payload!.minFileSize).toBe(42);
    expect(payload).toMatchObject({
      minFileSize: 42,
      maxFileSize: 10485760,
      maxResults: 1000,
      contextLines: 3,
    });
  });

  it('emits update when maxFileSize changes', async () => {
    const wrapper = mount(SizeLimitOptions);
    await wrapper.find('#max-filesize').setValue('9999');
    const payload = lastUpdate(wrapper);
    expect(payload).toBeDefined();
    expect(payload!.maxFileSize).toBe(9999);
  });

  it('emits update when maxResults changes', async () => {
    const wrapper = mount(SizeLimitOptions);
    await wrapper.find('#max-results').setValue('777');
    const payload = lastUpdate(wrapper);
    expect(payload).toBeDefined();
    expect(payload!.maxResults).toBe(777);
  });

  it('emits update when contextLines changes', async () => {
    const wrapper = mount(SizeLimitOptions);
    await wrapper.find('#context-lines').setValue('7');
    const payload = lastUpdate(wrapper);
    expect(payload).toBeDefined();
    expect(payload!.contextLines).toBe(7);
  });

  it('watch(contextLines) clamps falsy value to 3', async () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { contextLines: 5 },
    });
    // Update prop to a falsy value; watcher clamps local ref to 3.
    await wrapper.setProps({ contextLines: 0 });
    expect((wrapper.find('#context-lines').element as HTMLInputElement).value).toBe('3');
    const payload = lastUpdate(wrapper);
    expect(payload).toBeDefined();
    expect(payload!.contextLines).toBe(3);
  });

  it('watch(contextLines) clamps negative value to 3', async () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { contextLines: 5 },
    });
    await wrapper.setProps({ contextLines: -2 });
    expect((wrapper.find('#context-lines').element as HTMLInputElement).value).toBe('3');
    const payload = lastUpdate(wrapper);
    expect(payload).toBeDefined();
    expect(payload!.contextLines).toBe(3);
  });

  it('watch(contextLines) keeps positive value as-is', async () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { contextLines: 1 },
    });
    await wrapper.setProps({ contextLines: 8 });
    expect((wrapper.find('#context-lines').element as HTMLInputElement).value).toBe('8');
  });

  it('watch(minFileSize) falls back to 0 on falsy prop update', async () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { minFileSize: 100 },
    });
    await wrapper.setProps({ minFileSize: 0 });
    expect((wrapper.find('#min-filesize').element as HTMLInputElement).value).toBe('0');
  });

  it('watch(maxFileSize) falls back to default 10485760 on falsy prop update', async () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { maxFileSize: 5000 },
    });
    await wrapper.setProps({ maxFileSize: 0 });
    expect((wrapper.find('#max-filesize').element as HTMLInputElement).value).toBe('10485760');
  });

  it('watch(maxResults) falls back to 1000 on falsy prop update', async () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { maxResults: 25 },
    });
    await wrapper.setProps({ maxResults: 0 });
    expect((wrapper.find('#max-results').element as HTMLInputElement).value).toBe('1000');
  });

  it('disables all inputs when disabled prop is true', () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { disabled: true },
    });
    expect((wrapper.find('#min-filesize').element as HTMLInputElement).disabled).toBe(true);
    expect((wrapper.find('#max-filesize').element as HTMLInputElement).disabled).toBe(true);
    expect((wrapper.find('#max-results').element as HTMLInputElement).disabled).toBe(true);
    expect((wrapper.find('#context-lines').element as HTMLInputElement).disabled).toBe(true);
  });

  it('leaves inputs enabled when disabled prop is false (default)', () => {
    const wrapper = mount(SizeLimitOptions);
    expect((wrapper.find('#min-filesize').element as HTMLInputElement).disabled).toBe(false);
    expect((wrapper.find('#context-lines').element as HTMLInputElement).disabled).toBe(false);
  });

  it('initializes contextLines to default 3 when prop is falsy', () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { contextLines: 0 },
    });
    expect((wrapper.find('#context-lines').element as HTMLInputElement).value).toBe('3');
  });

  it('initializes minFileSize to 0 when prop is falsy', () => {
    const wrapper = mount(SizeLimitOptions, {
      props: { minFileSize: 0 },
    });
    expect((wrapper.find('#min-filesize').element as HTMLInputElement).value).toBe('0');
  });

  it('emits update with the full payload shape on any change', async () => {
    const wrapper = mount(SizeLimitOptions, {
      props: {
        minFileSize: 10,
        maxFileSize: 20,
        maxResults: 30,
        contextLines: 4,
      },
    });
    await wrapper.find('#min-filesize').setValue('11');
    const payload = lastUpdate(wrapper);
    expect(Object.keys(payload!)).toEqual(
      expect.arrayContaining([
        'minFileSize',
        'maxFileSize',
        'maxResults',
        'contextLines',
      ]),
    );
    expect(payload).toMatchObject({
      minFileSize: 11,
      maxFileSize: 20,
      maxResults: 30,
      contextLines: 4,
    });
  });
});
