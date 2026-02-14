import { Box } from '@mui/material'

interface PageContainerProps {
  children: React.ReactNode
  maxWidth?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | false
  disableGutters?: boolean
  sx?: object
}

/**
 * Container padrão para conteúdo de página: padding consistente e maxWidth opcional.
 */
export function PageContainer({
  children,
  maxWidth = 'lg',
  disableGutters = false,
  sx = {},
}: PageContainerProps) {
  return (
    <Box
      sx={{
        width: '100%',
        maxWidth: maxWidth === false ? 'none' : undefined,
        mx: 'auto',
        px: disableGutters ? 0 : { xs: 2, sm: 3 },
        py: 2,
        ...sx,
      }}
    >
      {children}
    </Box>
  )
}
