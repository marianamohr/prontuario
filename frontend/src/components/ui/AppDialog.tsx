import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  type DialogProps,
} from '@mui/material'

interface AppDialogProps extends Omit<DialogProps, 'title'> {
  open: boolean
  onClose: () => void
  title: React.ReactNode
  actions?: React.ReactNode
  maxWidth?: DialogProps['maxWidth']
}

/**
 * Modal padrão do produto: título, conteúdo e área de ações.
 */
export function AppDialog({
  open,
  onClose,
  title,
  children,
  actions,
  maxWidth = 'sm',
  ...rest
}: AppDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} maxWidth={maxWidth} fullWidth {...rest}>
      <DialogTitle sx={{ fontSize: '1.25rem', fontWeight: 600 }}>{title}</DialogTitle>
      <DialogContent dividers>{children}</DialogContent>
      {actions && <DialogActions sx={{ px: 2, py: 1.5 }}>{actions}</DialogActions>}
    </Dialog>
  )
}
