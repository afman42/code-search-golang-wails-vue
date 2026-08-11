import { ref, nextTick, onUpdated, type Ref } from "vue";

// ---------------------------------------------------------------------------
// Composable
// ---------------------------------------------------------------------------

/**
 * useLogViewer — component-specific view state for the LogViewer UI (not part
 * of the streaming logic in useLogStreaming). Owns the collapse/expand state
 * and the auto-scroll-to-bottom (tail) behavior.
 *
 * @param autoScroll reactive tail-mode flag (from useLogStreaming). When true,
 *   the container is scrolled to the bottom on every update; when false the
 *   user's scroll position is preserved.
 * @param containerRef template ref to the scrollable log container element,
 *   owned by the SFC so its `ref="containerRef"` binding is statically visible.
 */
export function useLogViewer(
  autoScroll: Ref<boolean>,
  containerRef: Ref<HTMLElement | null>,
) {
  // Component-specific state (not part of the streaming logic)
  const isCollapsed = ref(true); // Track whether logs are collapsed

  // Toggle collapse/expand and scroll to bottom
  const toggleCollapseAndScroll = () => {
    isCollapsed.value = !isCollapsed.value;
  };

  onUpdated(() => {
    // Only auto-scroll to bottom when autoScroll is enabled (tail mode).
    // When paused, the user's scroll position is preserved.
    if (autoScroll.value) {
      nextTick(() => {
        if (containerRef.value) {
          containerRef.value.scrollTop = containerRef.value.scrollHeight;
        }
      });
    }
  });

  return { isCollapsed, toggleCollapseAndScroll };
}
