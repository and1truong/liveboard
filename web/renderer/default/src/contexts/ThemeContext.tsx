import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

export type Mode = 'light' | 'dark' | 'system'
export type ThemeName = 'indigo' | 'github' | 'gitlab' | 'emerald' | 'rose' | 'sunset' | 'aqua'

export const THEME_NAMES: ThemeName[] = ['indigo', 'github', 'gitlab', 'emerald', 'rose', 'sunset', 'aqua']

const MODE_KEY = 'liveboard:mode'
const THEME_KEY = 'liveboard:theme'

interface ThemeCtx {
  mode: Mode
  theme: ThemeName
  resolvedDark: boolean
  setMode: (m: Mode) => void
  setTheme: (t: ThemeName) => void
}

const Ctx = createContext<ThemeCtx | null>(null)

function readMode(): Mode {
  try {
    const v = localStorage.getItem(MODE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {}
  return 'system'
}

function readTheme(): ThemeName {
  try {
    const v = localStorage.getItem(THEME_KEY)
    if (v && (THEME_NAMES as string[]).includes(v)) return v as ThemeName
  } catch {}
  return 'indigo'
}

function systemPrefersDark(): boolean {
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  } catch {
    return false
  }
}

export function ThemeProvider({ children }: { children: ReactNode }): JSX.Element {
  const [mode, setModeState] = useState<Mode>(() => readMode())
  const [theme, setThemeState] = useState<ThemeName>(() => readTheme())
  const [systemDark, setSystemDark] = useState<boolean>(() => systemPrefersDark())

  // Subscribe to OS changes while mode === 'system'.
  useEffect(() => {
    if (mode !== 'system') return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = (e: MediaQueryListEvent): void => setSystemDark(e.matches)
    mq.addEventListener('change', onChange)
    // Also re-read in case the stored value drifted.
    setSystemDark(mq.matches)
    return () => mq.removeEventListener('change', onChange)
  }, [mode])

  const resolvedDark = mode === 'dark' || (mode === 'system' && systemDark)

  // Apply dark class + persist mode.
  useEffect(() => {
    document.documentElement.classList.toggle('dark', resolvedDark)
    try { localStorage.setItem(MODE_KEY, mode) } catch {}
  }, [mode, resolvedDark])

  // Apply theme class + persist.
  useEffect(() => {
    const el = document.documentElement
    for (const t of THEME_NAMES) el.classList.remove(`theme-${t}`)
    el.classList.add(`theme-${theme}`)
    try { localStorage.setItem(THEME_KEY, theme) } catch {}
  }, [theme])

  const value = useMemo<ThemeCtx>(
    () => ({ mode, theme, resolvedDark, setMode: setModeState, setTheme: setThemeState }),
    [mode, theme, resolvedDark],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useTheme(): ThemeCtx {
  const v = useContext(Ctx)
  if (!v) throw new Error('useTheme must be used within ThemeProvider')
  return v
}
