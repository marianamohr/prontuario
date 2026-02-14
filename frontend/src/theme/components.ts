import type { Components, Theme } from '@mui/material/styles'

/**
 * Overrides padrão dos componentes MUI para o produto:
 * bordas, altura de input, radius, hover, etc.
 */
export function getComponentOverrides(_theme: Theme, density: 'comfortable' | 'compact'): Components<Theme> {
  const isCompact = density === 'compact'
  const borderRadius = 8

  return {
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius,
          textTransform: 'none',
          fontWeight: 600,
          padding: isCompact ? '6px 14px' : '8px 18px',
        },
      },
    },
    MuiTextField: {
      defaultProps: {
        variant: 'outlined',
        size: isCompact ? 'small' : 'medium',
      },
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius,
          },
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          borderRadius,
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          borderRadius,
          backgroundImage: 'none',
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius,
        },
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: {
          borderRadius,
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          borderRadius: 0,
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 6,
        },
      },
    },
    MuiAlert: {
      styleOverrides: {
        root: {
          borderRadius,
        },
      },
    },
  }
}

