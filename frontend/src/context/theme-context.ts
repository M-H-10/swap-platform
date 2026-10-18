import { createContext, useContext } from 'react';

export type Theme = 'light' | 'dark';

export interface ThemeContextValue {
  /** The theme on the screen right now. */
  theme: Theme;
  /** Switch light <-> dark and remember the choice. */
  toggleTheme: () => void;
}

export const ThemeContext = createContext<ThemeContextValue | null>(null);

/** The key in localStorage. index.html reads the same key before React starts. */
export const THEME_KEY = 'swap-theme';

/** The theme that index.html already put on <html data-theme="...">. */
export function currentTheme(): Theme {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
}

/** Use the theme in any component inside <ThemeProvider>.
 *  Charts call this so they draw again with the new colours after a switch. */
export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error('useTheme must be used inside <ThemeProvider>');
  }
  return ctx;
}