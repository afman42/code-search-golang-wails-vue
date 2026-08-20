import {createApp} from 'vue'
import App from '@/App.vue'
import './style.css';
import { initializeAppServices } from '@/services';

// In browser mode (vite dev / Playwright E2E) the Wails Go backend is absent.
// Install a mock backend BEFORE mounting so IsAppReady(), search, file reads,
// and symbol lookups work without the WebView. Gated on VITE_WAILS_MOCK.
//
// Dynamic import (exception to the static-import rule): this is a dev/test-only
// module that must NOT ship in the production Wails bundle. Loading it lazily
// puts it in its own chunk that is only fetched when the flag is set; in
// production the branch is statically dead and the chunk is never emitted into
// the entry. The await lives inside this async bootstrap (not top-level) so the
// es2020 build target — which forbids top-level await — still compiles.
async function bootstrap() {
  if (import.meta.env.VITE_WAILS_MOCK) {
    const { installWailsMock } = await import('./mocks/wailsMock');
    installWailsMock();
  }

  // Mount the app first so first paint is never blocked by background services.
  createApp(App).mount('#app');
}

void bootstrap().catch(console.error);

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
