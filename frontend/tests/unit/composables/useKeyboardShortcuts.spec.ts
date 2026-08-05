import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import type { Mock } from "vitest";
import { defineComponent, h } from "vue";
import { mount } from "@vue/test-utils";
import type { VueWrapper } from "@vue/test-utils";
import { useKeyboardShortcuts } from "../../../src/composables/useKeyboardShortcuts";
import type { KeyboardShortcutHandlers } from "../../../src/types/keyboard";

// --- Helpers --------------------------------------------------------------

// Track mounted wrappers so afterEach can guarantee cleanup even if a test
// throws before reaching its own wrapper.unmount().
const wrappers: VueWrapper<unknown>[] = [];

function mountWithComposable(handlers?: KeyboardShortcutHandlers) {
  const Dummy = defineComponent({
    setup() {
      useKeyboardShortcuts(handlers);
      return () => h("div");
    },
  });
  const wrapper = mount(Dummy);
  wrappers.push(wrapper);
  return wrapper;
}

interface DispatchOptions {
  ctrlKey?: boolean;
  metaKey?: boolean;
  target?: EventTarget | null;
}

// Build a KeyboardEvent whose `target` is controllable. The KeyboardEvent
// constructor ignores `target`, so it is pinned via defineProperty. When no
// target is supplied we default to document.body (a non-field element), which
// keeps isTypingInField() from throwing on a window target that lacks tagName.
function makeKeyEvent(key: string, options: DispatchOptions = {}): KeyboardEvent {
  const event = new KeyboardEvent("keydown", {
    key,
    bubbles: true,
    cancelable: true,
    ctrlKey: options.ctrlKey ?? false,
    metaKey: options.metaKey ?? false,
  });
  const target = options.target !== undefined ? options.target : document.body;
  Object.defineProperty(event, "target", { value: target, configurable: true });
  return event;
}

function dispatchKey(key: string, options: DispatchOptions = {}): KeyboardEvent {
  const event = makeKeyEvent(key, options);
  window.dispatchEvent(event);
  return event;
}

// Create a fake form-field element attached to the DOM.
function makeField(tag: string): HTMLElement {
  const el = document.createElement(tag);
  document.body.appendChild(el);
  return el;
}

// --- Specs ----------------------------------------------------------------

describe("useKeyboardShortcuts", () => {
  let handlers: {
    onFocusSearch: Mock;
    onExecuteSearch: Mock;
    onClearSearch: Mock;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    handlers = {
      onFocusSearch: vi.fn(),
      onExecuteSearch: vi.fn(),
      onClearSearch: vi.fn(),
    };
  });

  afterEach(() => {
    for (const wrapper of wrappers) {
      // unmount is a no-op if already unmounted.
      wrapper.unmount();
    }
    wrappers.length = 0;
    document.body.innerHTML = "";
  });

  describe("Ctrl+K / Cmd+K", () => {
    test("Ctrl+K calls onFocusSearch and prevents default + stops propagation", () => {
      mountWithComposable(handlers);
      const event = makeKeyEvent("k", { ctrlKey: true });
      const preventSpy = vi.spyOn(event, "preventDefault");
      const stopSpy = vi.spyOn(event, "stopPropagation");

      window.dispatchEvent(event);

      expect(handlers.onFocusSearch).toHaveBeenCalledTimes(1);
      expect(preventSpy).toHaveBeenCalledTimes(1);
      expect(stopSpy).toHaveBeenCalledTimes(1);
    });

    test("Cmd+K (metaKey) also calls onFocusSearch", () => {
      mountWithComposable(handlers);
      dispatchKey("k", { metaKey: true });
      expect(handlers.onFocusSearch).toHaveBeenCalledTimes(1);
    });

    test("uppercase 'K' with Ctrl calls onFocusSearch", () => {
      mountWithComposable(handlers);
      dispatchKey("K", { ctrlKey: true });
      expect(handlers.onFocusSearch).toHaveBeenCalledTimes(1);
    });

    test("Ctrl+K is active even while typing in a field", () => {
      mountWithComposable(handlers);
      const input = makeField("input");
      dispatchKey("k", { ctrlKey: true, target: input });
      expect(handlers.onFocusSearch).toHaveBeenCalledTimes(1);
    });

    test("plain 'k' without Ctrl/Cmd does not call onFocusSearch", () => {
      mountWithComposable(handlers);
      dispatchKey("k");
      expect(handlers.onFocusSearch).not.toHaveBeenCalled();
    });

    test("does not call onExecuteSearch or onClearSearch for Ctrl+K", () => {
      mountWithComposable(handlers);
      dispatchKey("k", { ctrlKey: true });
      expect(handlers.onExecuteSearch).not.toHaveBeenCalled();
      expect(handlers.onClearSearch).not.toHaveBeenCalled();
    });
  });

  describe("Ctrl+Enter", () => {
    test("Ctrl+Enter calls onExecuteSearch and prevents default + stops propagation", () => {
      mountWithComposable(handlers);
      const event = makeKeyEvent("Enter", { ctrlKey: true });
      const preventSpy = vi.spyOn(event, "preventDefault");
      const stopSpy = vi.spyOn(event, "stopPropagation");

      window.dispatchEvent(event);

      expect(handlers.onExecuteSearch).toHaveBeenCalledTimes(1);
      expect(preventSpy).toHaveBeenCalledTimes(1);
      expect(stopSpy).toHaveBeenCalledTimes(1);
    });

    test("Cmd+Enter (metaKey) also calls onExecuteSearch", () => {
      mountWithComposable(handlers);
      dispatchKey("Enter", { metaKey: true });
      expect(handlers.onExecuteSearch).toHaveBeenCalledTimes(1);
    });

    test("Ctrl+Enter is active even while typing in a field", () => {
      mountWithComposable(handlers);
      const textarea = makeField("textarea");
      const event = dispatchKey("Enter", { ctrlKey: true, target: textarea });
      expect(handlers.onExecuteSearch).toHaveBeenCalledTimes(1);
      expect(event.defaultPrevented).toBe(true);
    });

    test("plain Enter without Ctrl/Cmd does not call onExecuteSearch", () => {
      mountWithComposable(handlers);
      dispatchKey("Enter");
      expect(handlers.onExecuteSearch).not.toHaveBeenCalled();
    });
  });

  describe("ESC", () => {
    test("ESC while NOT typing calls onClearSearch and prevents default + stops propagation", () => {
      mountWithComposable(handlers);
      const event = makeKeyEvent("Escape");
      const preventSpy = vi.spyOn(event, "preventDefault");
      const stopSpy = vi.spyOn(event, "stopPropagation");

      window.dispatchEvent(event);

      expect(handlers.onClearSearch).toHaveBeenCalledTimes(1);
      expect(preventSpy).toHaveBeenCalledTimes(1);
      expect(stopSpy).toHaveBeenCalledTimes(1);
      expect(handlers.onFocusSearch).not.toHaveBeenCalled();
      expect(handlers.onExecuteSearch).not.toHaveBeenCalled();
    });

    test("ESC while typing in an input blurs the target and does NOT call onClearSearch", () => {
      mountWithComposable(handlers);
      const input = makeField("input") as HTMLInputElement;
      const blurSpy = vi.spyOn(input, "blur");

      const event = dispatchKey("Escape", { target: input });

      expect(handlers.onClearSearch).not.toHaveBeenCalled();
      expect(blurSpy).toHaveBeenCalledTimes(1);
      expect(event.defaultPrevented).toBe(false);
    });

    test("ESC while typing in a textarea blurs and does NOT call onClearSearch", () => {
      mountWithComposable(handlers);
      const textarea = makeField("textarea") as HTMLTextAreaElement;
      const blurSpy = vi.spyOn(textarea, "blur");

      dispatchKey("Escape", { target: textarea });

      expect(handlers.onClearSearch).not.toHaveBeenCalled();
      expect(blurSpy).toHaveBeenCalledTimes(1);
    });

    test("ESC while typing in a select blurs and does NOT call onClearSearch", () => {
      mountWithComposable(handlers);
      const select = makeField("select") as HTMLSelectElement;
      const blurSpy = vi.spyOn(select, "blur");

      dispatchKey("Escape", { target: select });

      expect(handlers.onClearSearch).not.toHaveBeenCalled();
      expect(blurSpy).toHaveBeenCalledTimes(1);
    });

    test("ESC while focused on a contentEditable element blurs and does NOT call onClearSearch", () => {
      mountWithComposable(handlers);
      const div = makeField("div") as HTMLElement;
      // jsdom does not reliably reflect isContentEditable from contentEditable,
      // so pin it directly to simulate a contentEditable element.
      Object.defineProperty(div, "isContentEditable", {
        value: true,
        configurable: true,
      });
      const blurSpy = vi.spyOn(div, "blur");

      dispatchKey("Escape", { target: div });

      expect(handlers.onClearSearch).not.toHaveBeenCalled();
      expect(blurSpy).toHaveBeenCalledTimes(1);
    });

    test("ESC when event target is null does not throw and calls onClearSearch", () => {
      mountWithComposable(handlers);
      // target null → isTypingInField returns false (guard) → onClearSearch fires.
      expect(() => dispatchKey("Escape", { target: null })).not.toThrow();
      expect(handlers.onClearSearch).toHaveBeenCalledTimes(1);
    });
  });

  describe("suppression while typing", () => {
    test("plain Enter while typing in a field does not trigger any handler", () => {
      mountWithComposable(handlers);
      const input = makeField("input");
      dispatchKey("Enter", { target: input });
      expect(handlers.onExecuteSearch).not.toHaveBeenCalled();
      expect(handlers.onFocusSearch).not.toHaveBeenCalled();
      expect(handlers.onClearSearch).not.toHaveBeenCalled();
    });

    test("arbitrary letter while typing in a field does not trigger handlers", () => {
      mountWithComposable(handlers);
      const input = makeField("input");
      dispatchKey("x", { target: input });
      expect(handlers.onFocusSearch).not.toHaveBeenCalled();
      expect(handlers.onExecuteSearch).not.toHaveBeenCalled();
      expect(handlers.onClearSearch).not.toHaveBeenCalled();
    });

    test("arbitrary letter while NOT typing does not trigger handlers", () => {
      mountWithComposable(handlers);
      dispatchKey("x");
      expect(handlers.onFocusSearch).not.toHaveBeenCalled();
      expect(handlers.onExecuteSearch).not.toHaveBeenCalled();
      expect(handlers.onClearSearch).not.toHaveBeenCalled();
    });
  });

  describe("optional / omitted handlers", () => {
    test("does not throw when handlers is undefined", () => {
      expect(() => mountWithComposable(undefined)).not.toThrow();
    });

    test("does not throw when handlers is an empty object", () => {
      expect(() => mountWithComposable({})).not.toThrow();
    });

    test("does not throw when individual handler callbacks are omitted", () => {
      mountWithComposable({ onFocusSearch: vi.fn() });
      // Only onFocusSearch provided; Ctrl+Enter / ESC should not throw.
      expect(() => dispatchKey("Enter", { ctrlKey: true })).not.toThrow();
      expect(() => dispatchKey("Escape")).not.toThrow();
    });

    test("does not throw when handlers is omitted and Ctrl+K fires", () => {
      mountWithComposable();
      expect(() => dispatchKey("k", { ctrlKey: true })).not.toThrow();
    });

    test("does not throw when handlers is omitted and ESC fires", () => {
      mountWithComposable();
      expect(() => dispatchKey("Escape")).not.toThrow();
    });
  });

  describe("listener lifecycle", () => {
    test("listener is registered on mount and removed on unmount", () => {
      const addSpy = vi.spyOn(window, "addEventListener");
      const removeSpy = vi.spyOn(window, "removeEventListener");

      const wrapper = mountWithComposable(handlers);

      const addCalls = addSpy.mock.calls.filter(
        (c) => c[0] === "keydown" && c[2] === true,
      );
      expect(addCalls).toHaveLength(1);

      // While mounted, the shortcut works.
      dispatchKey("k", { ctrlKey: true });
      expect(handlers.onFocusSearch).toHaveBeenCalledTimes(1);

      wrapper.unmount();

      // After unmount, the capture listener is removed and shortcuts no longer fire.
      const removeCalls = removeSpy.mock.calls.filter(
        (c) => c[0] === "keydown" && c[2] === true,
      );
      expect(removeCalls).toHaveLength(1);

      handlers.onFocusSearch.mockClear();
      dispatchKey("k", { ctrlKey: true });
      expect(handlers.onFocusSearch).not.toHaveBeenCalled();

      addSpy.mockRestore();
      removeSpy.mockRestore();
    });

    test("re-mounting registers a fresh listener (idempotent)", () => {
      const wrapper = mountWithComposable(handlers);
      dispatchKey("Escape");
      expect(handlers.onClearSearch).toHaveBeenCalledTimes(1);
      wrapper.unmount();

      handlers.onClearSearch.mockClear();
      // No listener after unmount.
      dispatchKey("Escape");
      expect(handlers.onClearSearch).not.toHaveBeenCalled();

      mountWithComposable(handlers);
      dispatchKey("Escape");
      expect(handlers.onClearSearch).toHaveBeenCalledTimes(1);
    });
  });
});
