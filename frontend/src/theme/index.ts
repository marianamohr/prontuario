import { createTheme } from '@mui/material/styles'
import { getPalette } from './palette'
import { typography } from './typography'
import { getComponentOverrides } from './components'

export type ThemeMode = 'light' | 'dark' | 'system'
export type Density = 'comfortable' | 'compact'
export { PRIMARY_PRESETS, getPalette } from './palette'
export { typography } from './typography'

export interface ThemeConfig {
  mode: ThemeMode
  primaryColorKey: string
  density: Density
}

function resolveMode(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') {
    if (typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
      return 'dark'
    }
    return 'light'
  }
  return mode
}

export function createAppTheme(config: ThemeConfig) {
  const resolvedMode = resolveMode(config.mode)
  const palette = getPalette(resolvedMode, config.primaryColorKey)
  const density = config.density
  const base = createTheme({
    palette,
    typography,
    shape: { borderRadius: 8 },
    spacing: 8,
  })
  return createTheme(base, {
    components: getComponentOverrides(base, density),
  })
}
