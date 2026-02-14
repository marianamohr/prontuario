import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  useEffect,
  type ReactNode,
} from 'react'
import { ThemeProvider } from '@mui/material/styles'
import type { ThemeMode, Density, ThemeConfig } from '../theme'
import { createAppTheme, PRIMARY_PRESETS } from '../theme'

const STORAGE_KEY = 'prontuario-theme-settings'

export interface ThemeSettings extends ThemeConfig {
  fontSizeLevel: 'normal' | 'large'
}

const defaultSettings: ThemeSettings = {
  mode: 'light',
  primaryColorKey: 'default',
  density: 'comfortable',
  fontSizeLevel: 'normal',
}

function loadSettings(): ThemeSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return defaultSettings
    const parsed = JSON.parse(raw) as Partial<ThemeSettings>
    return {
      mode: parsed.mode ?? defaultSettings.mode,
      primaryColorKey: parsed.primaryColorKey ?? defaultSettings.primaryColorKey,
      density: parsed.density ?? defaultSettings.density,
      fontSizeLevel: parsed.fontSizeLevel ?? defaultSettings.fontSizeLevel,
    }
  } catch {
    return defaultSettings
  }
}

function saveSettings(settings: ThemeSettings) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // ignore
  }
}

type ThemeSettingsContextValue = {
  settings: ThemeSettings
  setMode: (mode: ThemeMode) => void
  setPrimaryColorKey: (key: string) => void
  setDensity: (density: Density) => void
  setFontSizeLevel: (level: 'normal' | 'large') => void
  primaryPresets: Record<string, { main: string; dark?: string; light?: string }>
}

const ThemeSettingsContext = createContext<ThemeSettingsContextValue | null>(null)

export function ThemeSettingsProvider({ children }: { children: ReactNode }) {
  const [settings, setSettingsState] = useState<ThemeSettings>(loadSettings)
  const [systemDark, setSystemDark] = useState(
    () => typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches
  )

  useEffect(() => {
    saveSettings(settings)
  }, [settings])

  useEffect(() => {
    if (settings.mode !== 'system') return
    const m = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => setSystemDark(m.matches)
    m.addEventListener('change', handler)
    return () => m.removeEventListener('change', handler)
  }, [settings.mode])

  const setMode = useCallback((mode: ThemeMode) => {
    setSettingsState((prev) => ({ ...prev, mode }))
  }, [])

  const setPrimaryColorKey = useCallback((primaryColorKey: string) => {
    setSettingsState((prev) => ({ ...prev, primaryColorKey }))
  }, [])

  const setDensity = useCallback((density: Density) => {
    setSettingsState((prev) => ({ ...prev, density }))
  }, [])

  const setFontSizeLevel = useCallback((fontSizeLevel: 'normal' | 'large') => {
    setSettingsState((prev) => ({ ...prev, fontSizeLevel }))
  }, [])

  const themeConfig: ThemeConfig = useMemo(() => {
    const resolvedMode = settings.mode === 'system' ? (systemDark ? 'dark' : 'light') : settings.mode
    return {
      mode: resolvedMode as 'light' | 'dark',
      primaryColorKey: settings.primaryColorKey,
      density: settings.density,
    }
  }, [settings.mode, settings.primaryColorKey, settings.density, systemDark])

  const theme = useMemo(() => createAppTheme(themeConfig), [themeConfig])

  const value = useMemo(
    () => ({
      settings,
      setMode,
      setPrimaryColorKey,
      setDensity,
      setFontSizeLevel,
      primaryPresets: PRIMARY_PRESETS,
    }),
    [settings, setMode, setPrimaryColorKey, setDensity, setFontSizeLevel]
  )

  return (
    <ThemeSettingsContext.Provider value={value}>
      <ThemeProvider theme={theme}>{children}</ThemeProvider>
    </ThemeSettingsContext.Provider>
  )
}

export function useThemeSettings() {
  const ctx = useContext(ThemeSettingsContext)
  if (!ctx) throw new Error('useThemeSettings must be used within ThemeSettingsProvider')
  return ctx
}
