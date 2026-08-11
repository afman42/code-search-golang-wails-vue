// Barrel re-export of app-level services.
// Import from '@/services' rather than individual files.

export { initializeAppServices } from "./appInitializationService";
export {
  isHighlightJsLoaded,
  loadHighlightJs,
  detectLanguage,
  highlightCode,
} from "./syntaxHighlightingService";
