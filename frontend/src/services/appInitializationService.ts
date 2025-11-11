import { loadHighlightJs } from "./syntaxHighlightingService";

// Initialize all app services that should be loaded at startup
export const initializeAppServices = async () => {
  // Load syntax highlighting as one of the core services
  await loadHighlightJs();

  console.log("App services initialized successfully");
};
