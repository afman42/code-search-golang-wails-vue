import { loadHighlightJs } from "./syntaxHighlightingService";

// Initialize all app services that should be loaded at startup.
// loadHighlightJs is idempotent (self-guards on repeat calls).
export const initializeAppServices = async () => {
  await loadHighlightJs();
};
