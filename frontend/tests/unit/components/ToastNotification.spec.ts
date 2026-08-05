import { describe, test, expect, beforeEach } from "vitest";
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import ToastNotification from '../../../src/components/ui/ToastNotification.vue';
import { toastManager } from '../../../src/composables/useToast';

describe('ToastNotification.vue', () => {
  beforeEach(() => {
    // Clear all toasts (and any pending timers) between tests
    toastManager.clearAll();
  });

  test('renders nothing when there are no toasts', () => {
    const wrapper = mount(ToastNotification);
    expect(wrapper.findAll('.toast')).toHaveLength(0);
    expect(wrapper.text()).toBe('');
  });

  test('renders one .toast per toast, keyed by toast.id', async () => {
    toastManager.addToast('First message', { title: 'First' });
    toastManager.addToast('Second message', { title: 'Second' });

    const wrapper = mount(ToastNotification);
    await nextTick();

    const toasts = wrapper.findAll('.toast');
    expect(toasts).toHaveLength(2);

    // Each toast renders its title and message
    expect(wrapper.text()).toContain('First');
    expect(wrapper.text()).toContain('First message');
    expect(wrapper.text()).toContain('Second');
    expect(wrapper.text()).toContain('Second message');
  });

  test('applies toast--<type> class for each type', async () => {
    const types = ['success', 'error', 'warning', 'info'] as const;
    for (const type of types) {
      toastManager.addToast(`${type} message`, { type, title: type });
    }

    const wrapper = mount(ToastNotification);
    await nextTick();

    const toasts = wrapper.findAll('.toast');
    expect(toasts).toHaveLength(4);

    const classes = toasts.map(t => t.classes().filter(c => c.startsWith('toast--')));
    expect(classes).toEqual(
      expect.arrayContaining([
        ['toast--success'],
        ['toast--error'],
        ['toast--warning'],
        ['toast--info'],
      ])
    );

    // Ensure no extra/unexpected toast-- classes
    for (const c of classes) {
      expect(c).toHaveLength(1);
    }
  });

  test('mouseenter calls pauseToast and mouseleave calls resumeToast', async () => {
    const id = toastManager.addToast('Hover me', { duration: 10000 });

    const wrapper = mount(ToastNotification);
    await nextTick();

    const toastEl = wrapper.find('.toast');

    // Initially not paused
    const state = toastManager.toasts;
    const toast = state.find(t => t.id === id);
    expect(toast).toBeDefined();
    expect(toast?.paused).toBe(false);
    expect(toast?.timer).not.toBeNull();

    // Hover in -> pause
    await toastEl.trigger('mouseenter');
    const paused = toastManager.toasts.find(t => t.id === id);
    expect(paused?.paused).toBe(true);

    // Hover out -> resume
    await toastEl.trigger('mouseleave');
    const resumed = toastManager.toasts.find(t => t.id === id);
    expect(resumed?.paused).toBe(false);
    expect(resumed?.timer).not.toBeNull();
  });

  test('close button click calls removeToast and removes the toast', async () => {
    const id = toastManager.addToast('Close me');

    const wrapper = mount(ToastNotification);
    await nextTick();

    expect(wrapper.findAll('.toast')).toHaveLength(1);

    const closeBtn = wrapper.find('.toast__close');
    expect(closeBtn.exists()).toBe(true);

    await closeBtn.trigger('click');
    await nextTick();

    // The toast is removed from the store and from the DOM
    expect(wrapper.findAll('.toast')).toHaveLength(0);
    expect(toastManager.toasts.find(t => t.id === id)).toBeUndefined();
  });

  test('renders a progress element with animationDuration based on duration', async () => {
    const duration = 3000;
    toastManager.addToast('Timed', { duration });

    const wrapper = mount(ToastNotification);
    await nextTick();

    const progress = wrapper.find('.toast__progress');
    expect(progress.exists()).toBe(true);
    const style = progress.attributes('style') ?? '';
    expect(style).toContain(`animation-duration: ${duration}ms`);
  });
});
