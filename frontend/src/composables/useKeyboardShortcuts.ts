import { onMounted, onUnmounted } from "vue";

export interface KeyboardShortcutHandlers {
  onFocusSearch?: () => void;
  onExecuteSearch?: () => void;
  onClearSearch?: () => void;
}

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
 */
export function useKeyboardShortcuts(handlers?: KeyboardShortcutHandlers) {
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
      handlers?.onFocusSearch?.();
      return;
    }

    // Ctrl+Enter: execute search — active even while typing in the search box
    if (ctrl && event.key === "Enter") {
      event.preventDefault();
      event.stopPropagation();
      handlers?.onExecuteSearch?.();
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
      handlers?.onClearSearch?.();
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
}
