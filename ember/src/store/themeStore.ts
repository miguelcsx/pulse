import { create } from "zustand";

type Theme = "light" | "dark" | "system";
type Resolved = "light" | "dark";

interface ThemeState {
  theme: Theme;
  resolved: Resolved;
  setTheme: (theme: Theme) => void;
}

function getSystemTheme(): Resolved {
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function resolve(theme: Theme): Resolved {
  return theme === "system" ? getSystemTheme() : theme;
}

function applyTheme(resolved: Resolved) {
  document.documentElement.classList.toggle("dark", resolved === "dark");
}

function loadTheme(): Theme {
  const stored = localStorage.getItem("pulse_theme");
  if (stored === "light" || stored === "dark" || stored === "system")
    return stored;
  return "system";
}

const initial = loadTheme();
const initialResolved = resolve(initial);
applyTheme(initialResolved);

export const useThemeStore = create<ThemeState>()((set) => ({
  theme: initial,
  resolved: initialResolved,

  setTheme: (theme) => {
    const resolved = resolve(theme);
    localStorage.setItem("pulse_theme", theme);
    applyTheme(resolved);
    set({ theme, resolved });
  },
}));

// Listen for system theme changes when in "system" mode
window
  .matchMedia("(prefers-color-scheme: dark)")
  .addEventListener("change", () => {
    const state = useThemeStore.getState();
    if (state.theme === "system") {
      const resolved = getSystemTheme();
      applyTheme(resolved);
      useThemeStore.setState({ resolved });
    }
  });
