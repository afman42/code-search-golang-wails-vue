import { ref, onMounted } from "vue";
import { loadHighlightJs, isHighlightingReady } from "../services/syntaxHighlightingService";

// Composable to handle syntax highlighting initialization
export function useSyntaxHighlighting() {
  const isSyntaxHighlightingReady = ref(false);
  const isLoading = ref(false);
  const loadError = ref<string | null>(null);

  // Initialize syntax highlighting
  const initializeSyntaxHighlighting = async () => {
    if (isHighlightingReady()) {
      isSyntaxHighlightingReady.value = true;
      return true;
    }

    isLoading.value = true;
    loadError.value = null;

    try {
      const result = await loadHighlightJs();
      isSyntaxHighlightingReady.value = result;
      return result;
    } catch (error) {
      console.error("Error initializing syntax highlighting:", error);
      loadError.value = error instanceof Error ? error.message : "Unknown error";
      return false;
    } finally {
      isLoading.value = false;
    }
  };

  // Preload syntax highlighting when component is mounted
  onMounted(() => {
    initializeSyntaxHighlighting();
  });

  return {
    isSyntaxHighlightingReady,
    isLoading,
    loadError,
    initializeSyntaxHighlighting,
  };
}