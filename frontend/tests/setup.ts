// Global setup for Vitest, run before each test file.
import { vi, beforeEach, beforeAll } from "vitest";
import { loadHighlightJs } from "@/services";

// Preload highlight.js once per test file. The CodeModal component lazily imports
// ~25 highlight.js language modules on first use; preloading here ensures the
// `isHighlightingLoaded` cache is warm before any component mounts, so per-test
// highlighting resolves within a tick instead of racing the dynamic imports.
beforeAll(async () => {
  await loadHighlightJs();
});

// Mock IntersectionObserver for the CodeModal component.
interface MockObserverInit {
  root?: Element | null;
  rootMargin?: string;
  threshold?: number | number[];
}

class MockIntersectionObserver implements IntersectionObserver {
  readonly root: Element | null = null;
  readonly rootMargin = "";
  readonly thresholds: ReadonlyArray<number> = [];
  private callback: IntersectionObserverCallback;

  constructor(callback: IntersectionObserverCallback, options?: MockObserverInit) {
    this.callback = callback;
    if (options?.threshold !== undefined) {
      this.thresholds = Array.isArray(options.threshold)
        ? options.threshold
        : [options.threshold];
    }
  }

  takeRecords(): IntersectionObserverEntry[] {
    return [];
  }

  observe() {
    // no-op
  }

  unobserve() {
    // no-op
  }

  disconnect() {
    // no-op
  }

  static toString() {
    return "function IntersectionObserver() { [native code] }";
  }
}

(globalThis as unknown as Record<string, unknown>).IntersectionObserver =
  MockIntersectionObserver;

// jsdom does not implement Element.scrollIntoView. CodeModal calls it when
// navigating between matches and jumping to lines, so stub it out.
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = vi.fn();
}

// Mock document.execCommand for the clipboard fallback path.
Object.defineProperty(document, "execCommand", {
  value: vi.fn(() => true),
  writable: true,
});

// Reset mock state between tests without dropping mock implementations.
beforeEach(() => {
  vi.clearAllMocks();
});
