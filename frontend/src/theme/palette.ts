import type { PaletteOptions } from '@mui/material/styles'

/** Cores primárias predefinidas para o seletor de tema */
export const PRIMARY_PRESETS: Record<string, { main: string; dark?: string; light?: string }> = {
  default: { main: '#2d8f7e', dark: '#247268', light: '#4db6ac' },
  blue: { main: '#1976d2', dark: '#1565c0', light: '#42a5f5' },
  teal: { main: '#00796b', dark: '#004d40', light: '#4db6ac' },
  indigo: { main: '#3949ab', dark: '#303f9f', light: '#5c6bc0' },
  deepPurple: { main: '#512da8', dark: '#4527a0', light: '#7e57c2' },
  green: { main: '#2e7d32', dark: '#1b5e20', light: '#4caf50' },
}

export function getPalette(mode: 'light' | 'dark', primaryKey: string): PaletteOptions {
  const primary = PRIMARY_PRESETS[primaryKey] ?? PRIMARY_PRESETS.default
  const isDark = mode === 'dark'
  return {
    mode,
    primary: {
      main: primary.main,
      dark: primary.dark,
      light: primary.light,
    },
    background: {
      default: isDark ? '#121212' : '#f7faf9',
      paper: isDark ? '#1e1e1e' : '#ffffff',
    },
    text: {
      primary: isDark ? 'rgba(255,255,255,0.87)' : '#1a1a2e',
      secondary: isDark ? 'rgba(255,255,255,0.6)' : '#6b7280',
    },
  }
}
