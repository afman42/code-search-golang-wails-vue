import { loadHighlightJs, isHighlightJsLoaded } from "./syntaxHighlightingService";

// Initialize all app services that should be loaded at startup
export const initializeAppServices = async () => {
  // Load syntax highlighting only if not already loaded (lazy-load optimized)
  if (!isHighlightJsLoaded()) {
    await loadHighlightJs();
  }

  console.log("App services initialized successfully");
};
