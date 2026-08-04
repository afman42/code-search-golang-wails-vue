import { ref } from "vue";

// Dark/light theme state persisted to localStorage and applied to <html> via
// the `data-theme` attribute, which style.css uses to override the design
// tokens. Components never know about the theme — they read tokens.

export const THEME_STORAGE_KEY = "codeSearchTheme";
export type AppTheme = "light" | "dark";

function readInitialTheme(): AppTheme {
  try {
    const saved = localStorage.getItem(THEME_STORAGE_KEY);
    if (saved === "light" || saved === "dark") {
      return saved;
    }
  } catch (error) {
    console.error("Failed to read theme from localStorage:", error);
  }
  // No saved preference: follow the OS setting.
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function applyTheme(theme: AppTheme): void {
  document.documentElement.dataset.theme = theme;
}

export function useTheme() {
  const theme = ref<AppTheme>(readInitialTheme());
  applyTheme(theme.value);

  const setTheme = (next: AppTheme): void => {
    theme.value = next;
    applyTheme(next);
    try {
      localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch (error) {
      console.error("Failed to save theme to localStorage:", error);
    }
  };

  const toggleTheme = (): void => {
    setTheme(theme.value === "light" ? "dark" : "light");
  };

  return {
    theme,
    isDark: theme,
    setTheme,
    toggleTheme,
  };
}
