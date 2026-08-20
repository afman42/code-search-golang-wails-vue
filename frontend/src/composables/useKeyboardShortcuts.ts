import { onMounted, onUnmounted, watch, unref, type MaybeRefOrGetter } from "vue";
import type { KeyboardShortcutHandlers } from "@/types";

/**
 * UseEffect-style composable for global keyboard shortcuts.
 *
 * Map:
 *   Ctrl+K       → focus search input
 *   Ctrl+Enter   → execute search
 *   ESC          → clear search / close modals
 *
 * Shortcuts are suppressed while the user is typing in an input, textarea,
 * select, or contentEditable element — EXCEPT for Ctrl+Enter and ESC, which
 * remain active so the user can submit or clear from within the field.
 *
 * `handlers` may be a plain object or a getter/ref. When given a getter/ref,
 * handler changes are picked up reactively without re-binding the DOM
 * listener (a plain object is captured once at setup).
 */
export function useKeyboardShortcuts(
  handlers?: KeyboardShortcutHandlers | MaybeRefOrGetter<KeyboardShortcutHandlers | undefined>,
) {
  // Resolve the current handler object (getter/ref re-read on each access).
  const resolveHandlers = ():
    | KeyboardShortcutHandlers
    | undefined =>
    typeof handlers === "function"
      ? handlers()
      : unref(handlers);

  let currentHandlers = resolveHandlers();

  const isTypingInField = (target: EventTarget | null): boolean => {
    const el = target as HTMLElement | null;
    if (!el) return false;
    const tag = el.tagName.toLowerCase();
    return (
      tag === "input" ||
      tag === "textarea" ||
      tag === "select" ||
      el.isContentEditable
    );
  };

  const handleKeyDown = (event: KeyboardEvent) => {
    const ctrl = event.ctrlKey || event.metaKey;
    const typing = isTypingInField(event.target);

    // Ctrl+K: focus search input — always active, even while typing elsewhere
    if (ctrl && (event.key === "k" || event.key === "K")) {
      event.preventDefault();
      event.stopPropagation();
      currentHandlers?.onFocusSearch?.();
      return;
    }

    // Ctrl+Enter: execute search — active even while typing in the search box
    if (ctrl && event.key === "Enter") {
      event.preventDefault();
      event.stopPropagation();
      currentHandlers?.onExecuteSearch?.();
      return;
    }

    // ESC: clear search / close modals — active everywhere
    if (event.key === "Escape") {
      // If typing in a field, ESC should blur/clear that field rather than
      // triggering a global clear; let the browser handle it.
      if (typing) {
        (event.target as HTMLElement).blur();
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      currentHandlers?.onClearSearch?.();
      return;
    }

    // Other shortcuts only fire when not typing in a form field
    if (typing) {
      return;
    }
  };

  onMounted(() => {
    window.addEventListener("keydown", handleKeyDown, true);
  });

  onUnmounted(() => {
    window.removeEventListener("keydown", handleKeyDown, true);
  });

  // Re-bind the handler object when a reactive handler source changes; the
  // keydown listener itself stays bound so mount/unmount churn is avoided.
  if (typeof handlers === "function") {
    watch(handlers, (next) => {
      currentHandlers = next ?? undefined;
    });
  }
}
