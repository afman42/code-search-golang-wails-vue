import { describe, test, expect, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { EditorSelect } from '@/components/ui';
import type { EditorAvailability } from '@/types';

// All editor keys defined on EditorAvailability (excluding `systemdefault`,
// which is not rendered as an <option> by the component).
const EDITOR_KEYS = [
  'vscode',
  'vscodium',
  'sublime',
  'jetbrains',
  'geany',
  'goland',
  'pycharm',
  'intellij',
  'webstorm',
  'phpstorm',
  'clion',
  'rider',
  'androidstudio',
  'emacs',
  'neovide',
  'codeblocks',
  'devcpp',
  'notepadplusplus',
  'visualstudio',
  'eclipse',
  'netbeans',
  'neovim',
  'vim',
] as const;

const allFalse: EditorAvailability = {
  vscode: false,
  vscodium: false,
  sublime: false,
  jetbrains: false,
  geany: false,
  neovim: false,
  vim: false,
  goland: false,
  pycharm: false,
  intellij: false,
  webstorm: false,
  phpstorm: false,
  clion: false,
  rider: false,
  androidstudio: false,
  systemdefault: false,
  emacs: false,
  neovide: false,
  codeblocks: false,
  devcpp: false,
  notepadplusplus: false,
  visualstudio: false,
  eclipse: false,
  netbeans: false,
};

const allTrue: EditorAvailability = { ...allFalse, ...Object.fromEntries(EDITOR_KEYS.map((k) => [k, true])) } as EditorAvailability;

describe('EditorSelect.vue', () => {
  beforeEach(() => {
    // ensure no state leaks between tests
  });

  test('always renders the placeholder option with empty value', () => {
    const wrapper = mount(EditorSelect, { props: { availableEditors: allFalse } });
    const placeholder = wrapper.find('option[value=""]');
    expect(placeholder.exists()).toBe(true);
    expect(placeholder.text()).toBe('Editor...');
  });

  test('always renders the System Default option with value="default"', () => {
    const wrapper = mount(EditorSelect, { props: { availableEditors: allFalse } });
    const def = wrapper.find('option[value="default"]');
    expect(def.exists()).toBe(true);
    expect(def.text()).toBe('System Default');
  });

  test('renders only placeholder and System Default when availableEditors is all-false', () => {
    const wrapper = mount(EditorSelect, { props: { availableEditors: allFalse } });
    const options = wrapper.findAll('option');
    expect(options).toHaveLength(2);
    expect(options[0].attributes('value')).toBe('');
    expect(options[0].text()).toBe('Editor...');
    expect(options[1].attributes('value')).toBe('default');
    expect(options[1].text()).toBe('System Default');
  });

  test('renders an option for every editor key whose value is true', () => {
    const available: EditorAvailability = {
      ...allFalse,
      vscode: true,
      sublime: true,
      neovim: true,
    };
    const wrapper = mount(EditorSelect, { props: { availableEditors: available } });

    // 2 fixed options + 3 enabled editor options
    const options = wrapper.findAll('option');
    expect(options).toHaveLength(5);

    const values = options.map((o) => o.attributes('value'));
    expect(values).toContain('vscode');
    expect(values).toContain('sublime');
    expect(values).toContain('neovim');

    expect(wrapper.find('option[value="vscode"]').text()).toBe('VSCode');
    expect(wrapper.find('option[value="sublime"]').text()).toBe('Sublime Text');
    expect(wrapper.find('option[value="neovim"]').text()).toBe('Neovim');
  });

  test('does not render an option when its editor key is false', () => {
    const available: EditorAvailability = { ...allFalse, vscode: true };
    const wrapper = mount(EditorSelect, { props: { availableEditors: available } });
    expect(wrapper.find('option[value="vscodium"]').exists()).toBe(false);
    expect(wrapper.find('option[value="vim"]').exists()).toBe(false);
  });

  test('renders all editor options when every editor key is true', () => {
    const wrapper = mount(EditorSelect, { props: { availableEditors: allTrue } });
    const options = wrapper.findAll('option');
    // 2 fixed (placeholder + System Default) + one per EDITOR_KEYS
    expect(options).toHaveLength(EDITOR_KEYS.length + 2);
    for (const key of EDITOR_KEYS) {
      expect(wrapper.find(`option[value="${key}"]`).exists()).toBe(true);
    }
  });

  test('emits editorSelect with the native change Event when selection changes', async () => {
    const wrapper = mount(EditorSelect, { props: { availableEditors: allTrue } });
    const select = wrapper.find('select');
    await select.setValue('vscode');

    const emitted = wrapper.emitted('editorSelect');
    expect(emitted).toBeTruthy();
    expect(emitted).toHaveLength(1);
    // The handler does $emit('editorSelect', $event), so the payload is the Event itself.
    const event = emitted![0][0];
    expect(event).toBeInstanceOf(Event);
    expect((event.target as HTMLSelectElement).value).toBe('vscode');
  });

  test('emits editorSelect on every change, not just the first', async () => {
    const wrapper = mount(EditorSelect, { props: { availableEditors: allTrue } });
    const select = wrapper.find('select');

    await select.setValue('vscode');
    await select.setValue('default');
    await select.setValue('vim');

    const emitted = wrapper.emitted('editorSelect');
    expect(emitted).toBeTruthy();
    expect(emitted).toHaveLength(3);
    expect((emitted![0][0] as Event).target).toBeInstanceOf(HTMLSelectElement);
  });

  test('has the editor-select class and title attribute', () => {
    const wrapper = mount(EditorSelect, { props: { availableEditors: allFalse } });
    const select = wrapper.find('select');
    expect(select.classes()).toContain('editor-select');
    expect(select.attributes('title')).toBe('Open in editor');
  });
});
