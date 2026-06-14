import {createApp} from 'vue'
import App from './App.vue'
import './style.css';
import { initializeAppServices } from './services/appInitializationService';

// Mount the app first so first paint is never blocked by background services.
createApp(App).mount('#app')

// Warm up non-critical services (e.g. highlight.js) after the UI is interactive.
// highlightCode() also lazy-loads on demand, so this is purely a head start and
// is safe to defer. Use requestIdleCallback when available, falling back to a
// macrotask so it still runs after the initial render.
const warmUpServices = () => {
  void initializeAppServices();
};

if (typeof window !== 'undefined' && 'requestIdleCallback' in window) {
  (window as Window & typeof globalThis).requestIdleCallback(warmUpServices);
} else {
  setTimeout(warmUpServices, 0);
}
