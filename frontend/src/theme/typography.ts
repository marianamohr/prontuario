import type { TypographyOptions } from '@mui/material/styles/createTypography'

export const typography: TypographyOptions = {
  fontFamily: '"DM Sans", "Roboto", "Helvetica", "Arial", sans-serif',
  h1: { fontSize: '1.75rem', fontWeight: 600, lineHeight: 1.3 },
  h2: { fontSize: '1.5rem', fontWeight: 600, lineHeight: 1.35 },
  h3: { fontSize: '1.25rem', fontWeight: 600, lineHeight: 1.4 },
  h4: { fontSize: '1.125rem', fontWeight: 600, lineHeight: 1.4 },
  h5: { fontSize: '1rem', fontWeight: 600, lineHeight: 1.5 },
  h6: { fontSize: '0.9375rem', fontWeight: 600, lineHeight: 1.5 },
  body1: { fontSize: '1rem', lineHeight: 1.6 },
  body2: { fontSize: '0.875rem', lineHeight: 1.6 },
  button: { textTransform: 'none' as const, fontWeight: 600 },
}
