import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Box, Typography, Paper } from '@mui/material'

const BASE = (import.meta.env.VITE_API_URL || '').replace(/\/$/, '')

type VerifyInfo = {
  contract_id: string
  status: string
  signed_at: string
  pdf_sha256: string | null
  verification_token: string | null
  body_html?: string
}

export function VerifyContract() {
  const { token } = useParams<{ token: string }>()
  const [info, setInfo] = useState<VerifyInfo | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) {
      setError('Token não informado.')
      return
    }
    fetch(`${BASE}/api/contracts/verify/${token}`)
      .then((r) => {
        if (!r.ok) throw new Error('Não encontrado.')
        return r.json()
      })
      .then(setInfo)
      .catch(() => setError('Contrato não encontrado ou token inválido.'))
  }, [token])

  if (error) {
    return (
      <Box sx={{ minHeight: '100vh', p: 2, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Typography color="error" sx={{ fontSize: '1.1rem' }}>{error}</Typography>
      </Box>
    )
  }
  if (!info) {
    return (
      <Box sx={{ minHeight: '100vh', p: 2, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Typography color="text.secondary">Verificando...</Typography>
      </Box>
    )
  }

  const signedAtFormatted = info.signed_at ? new Date(info.signed_at).toLocaleString('pt-BR') : ''

  return (
    <Box sx={{ minHeight: '100vh', width: '100%', boxSizing: 'border-box', bgcolor: 'grey.50', py: 2, px: 1.5 }}>
      <Box sx={{ maxWidth: 900, mx: 'auto' }}>
        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
          <Typography variant="h5" sx={{ mb: 1 }}>Verificação de contrato</Typography>
          <Typography><strong>Status:</strong> {info.status}</Typography>
          <Typography><strong>Assinado em:</strong> {signedAtFormatted}</Typography>
          {info.pdf_sha256 && (
            <Typography sx={{ wordBreak: 'break-all', fontSize: 14, mt: 0.5 }}>
              <strong>SHA-256 do PDF:</strong> {info.pdf_sha256}
            </Typography>
          )}
          <Typography sx={{ mt: 1, color: 'text.secondary', fontSize: 14 }}>
            Este documento foi assinado eletronicamente. A autenticidade pode ser verificada pelo hash acima.
          </Typography>
        </Paper>
        {info.body_html && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="h6" sx={{ mb: 1 }}>Preview do contrato</Typography>
            <Paper variant="outlined" className="verify-contract-body" sx={{ p: 2.5, minHeight: 400, overflow: 'auto' }} dangerouslySetInnerHTML={{ __html: info.body_html }} />
          </Box>
        )}
      </Box>
    </Box>
  )
}
