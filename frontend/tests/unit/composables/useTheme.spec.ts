import { describe, test, expect, vi, beforeEach, afterEach } from "vitest";
import {
  useTheme,
  THEME_STORAGE_KEY,
} from "../../../src/composables/useTheme";

// `readInitialTheme` and `applyTheme` are module-private; they are exercised
// exclusively through `useTheme()`, which is the only exported surface.

/** Install a `window.matchMedia` mock reporting the given `prefers-color-scheme: dark` result. */
function mockMatchMedia(matches: boolean): void {
  const mql = {
    matches,
    media: "(prefers-color-scheme: dark)",
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(() => false),
  } as MediaQueryList;
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn(() => mql),
  });
}

describe("useTheme composable", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    // jsdom leaves the DOM attribute set across tests; reset it.
    delete document.documentElement.dataset.theme;
    // Default OS preference: light.
    mockMatchMedia(false);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("readInitialTheme (via useTheme)", () => {
    test("returns saved 'dark' from localStorage when present and valid", () => {
      localStorage.setItem(THEME_STORAGE_KEY, "dark");

      const { theme } = useTheme();

      expect(theme.value).toBe("dark");
    });

    test("returns saved 'light' from localStorage when present and valid", () => {
      localStorage.setItem(THEME_STORAGE_KEY, "light");

      const { theme } = useTheme();

      expect(theme.value).toBe("light");
    });

    test("saved value takes precedence over OS preference", () => {
      localStorage.setItem(THEME_STORAGE_KEY, "light");
      mockMatchMedia(true); // OS says dark

      const { theme } = useTheme();

      expect(theme.value).toBe("light");
    });

    test("ignores invalid localStorage value and follows OS dark preference", () => {
      localStorage.setItem(THEME_STORAGE_KEY, "purple");
      mockMatchMedia(true);

      const { theme } = useTheme();

      expect(theme.value).toBe("dark");
    });

    test("ignores invalid localStorage value and falls back to light when OS is not dark", () => {
      localStorage.setItem(THEME_STORAGE_KEY, "purple");
      mockMatchMedia(false);

      const { theme } = useTheme();

      expect(theme.value).toBe("light");
    });

    test("falls back to OS dark preference when no localStorage entry exists", () => {
      mockMatchMedia(true);

      const { theme } = useTheme();

      expect(theme.value).toBe("dark");
    });

    test("falls back to light when no localStorage entry and OS is not dark", () => {
      mockMatchMedia(false);

      const { theme } = useTheme();

      expect(theme.value).toBe("light");
    });

    test("falls back to light when matchMedia is unavailable", () => {
      // Simulate an environment without matchMedia (the source uses optional chaining).
      Object.defineProperty(window, "matchMedia", {
        configurable: true,
        writable: true,
        value: undefined,
      });

      const { theme } = useTheme();

      expect(theme.value).toBe("light");
    });

    test("catches localStorage.getItem throwing and falls back to OS setting", () => {
      vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
        throw new Error("localStorage disabled");
      });
      const errorSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => undefined);
      mockMatchMedia(true);

      const { theme } = useTheme();

      expect(theme.value).toBe("dark");
      expect(errorSpy).toHaveBeenCalledWith(
        "Failed to read theme from localStorage:",
        expect.any(Error),
      );
    });

    test("catches localStorage.getItem throwing and falls back to light when OS is not dark", () => {
      vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
        throw new Error("localStorage disabled");
      });
      vi.spyOn(console, "error").mockImplementation(() => undefined);
      mockMatchMedia(false);

      const { theme } = useTheme();

      expect(theme.value).toBe("light");
    });
  });

  describe("applyTheme (via useTheme)", () => {
    test("sets data-theme attribute on documentElement to the initial theme", () => {
      localStorage.setItem(THEME_STORAGE_KEY, "dark");

      useTheme();

      expect(document.documentElement.dataset.theme).toBe("dark");
    });

    test("applies initial light theme to DOM on creation", () => {
      mockMatchMedia(false);

      useTheme();

      expect(document.documentElement.dataset.theme).toBe("light");
    });

    test("applies initial dark theme to DOM on creation", () => {
      mockMatchMedia(true);

      useTheme();

      expect(document.documentElement.dataset.theme).toBe("dark");
    });
  });

  describe("setTheme", () => {
    test("updates the theme ref", () => {
      const { theme, setTheme } = useTheme();

      setTheme("dark");

      expect(theme.value).toBe("dark");
    });

    test("applies the new theme to the DOM", () => {
      const { setTheme } = useTheme();

      setTheme("dark");

      expect(document.documentElement.dataset.theme).toBe("dark");
    });

    test("persists the new theme to localStorage", () => {
      const setItemSpy = vi.spyOn(Storage.prototype, "setItem");

      const { setTheme } = useTheme();

      setTheme("dark");

      expect(setItemSpy).toHaveBeenCalledWith(THEME_STORAGE_KEY, "dark");
      expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe("dark");
    });

    test("catches localStorage.setItem failure without throwing", () => {
      vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
        throw new Error("quota exceeded");
      });
      const errorSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => undefined);

      const { theme, setTheme } = useTheme();

      expect(() => setTheme("dark")).not.toThrow();
      // Ref and DOM still update even when persistence fails.
      expect(theme.value).toBe("dark");
      expect(document.documentElement.dataset.theme).toBe("dark");
      expect(errorSpy).toHaveBeenCalledWith(
        "Failed to save theme to localStorage:",
        expect.any(Error),
      );
    });
  });

  describe("toggleTheme", () => {
    test("flips light to dark and persists", () => {
      mockMatchMedia(false);
      const setItemSpy = vi.spyOn(Storage.prototype, "setItem");

      const { theme, toggleTheme } = useTheme();

      expect(theme.value).toBe("light");

      toggleTheme();

      expect(theme.value).toBe("dark");
      expect(document.documentElement.dataset.theme).toBe("dark");
      expect(setItemSpy).toHaveBeenCalledWith(THEME_STORAGE_KEY, "dark");
    });

    test("flips dark to light and persists", () => {
      localStorage.setItem(THEME_STORAGE_KEY, "dark");
      const setItemSpy = vi.spyOn(Storage.prototype, "setItem");

      const { theme, toggleTheme } = useTheme();

      expect(theme.value).toBe("dark");

      toggleTheme();

      expect(theme.value).toBe("light");
      expect(document.documentElement.dataset.theme).toBe("light");
      expect(setItemSpy).toHaveBeenCalledWith(THEME_STORAGE_KEY, "light");
    });

    test("repeated toggles cycle between light and dark", () => {
      mockMatchMedia(false);
      const { theme, toggleTheme } = useTheme();

      toggleTheme();
      expect(theme.value).toBe("dark");
      toggleTheme();
      expect(theme.value).toBe("light");
      toggleTheme();
      expect(theme.value).toBe("dark");
    });
  });

  describe("isDark", () => {
    test("reflects the initial light theme", () => {
      mockMatchMedia(false);

      const { isDark } = useTheme();

      expect(isDark.value).toBe("light");
    });

    test("reflects the initial dark theme", () => {
      mockMatchMedia(true);

      const { isDark } = useTheme();

      expect(isDark.value).toBe("dark");
    });

    test("stays in sync with the theme after setTheme", () => {
      mockMatchMedia(false);
      const { isDark, setTheme } = useTheme();

      setTheme("dark");
      expect(isDark.value).toBe("dark");

      setTheme("light");
      expect(isDark.value).toBe("light");
    });

    test("stays in sync with the theme after toggleTheme", () => {
      mockMatchMedia(false);
      const { isDark, toggleTheme } = useTheme();

      toggleTheme();
      expect(isDark.value).toBe("dark");
    });

    test("is the same reactive ref as theme", () => {
      const { theme, isDark } = useTheme();

      expect(isDark).toBe(theme);
    });
  });
});
