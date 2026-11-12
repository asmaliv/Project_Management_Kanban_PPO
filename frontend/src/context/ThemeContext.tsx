import { createContext, useEffect, useMemo, useState, type ReactNode } from 'react'

export type Theme = 'light' | 'dark' | 'neobrutalism'

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => void
}

const STORAGE_KEY = 'ppo-theme'
const THEME_CLASSES: Theme[] = ['light', 'dark', 'neobrutalism']

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined)

function getStoredTheme(): Theme {
  const storedTheme = localStorage.getItem(STORAGE_KEY)
  return storedTheme && THEME_CLASSES.includes(storedTheme as Theme)
    ? (storedTheme as Theme)
    : 'light'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(getStoredTheme)

  useEffect(() => {
    const root = document.documentElement
    root.classList.remove(...THEME_CLASSES.map((item) => `theme-${item}`))
    root.classList.add(`theme-${theme}`)
    root.style.colorScheme = theme === 'dark' ? 'dark' : 'light'
    localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  const value = useMemo(() => ({ theme, setTheme }), [theme])

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export { ThemeContext }
