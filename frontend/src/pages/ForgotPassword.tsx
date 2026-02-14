import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Box, Typography, Button, TextField, Paper } from '@mui/material'
import * as api from '../lib/api'

export function ForgotPassword() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      await api.forgotPassword(email)
      setSent(true)
    } catch {
      setSent(true)
    } finally {
      setLoading(false)
    }
  }

  if (sent) {
    return (
      <Box sx={{ maxWidth: 400, mx: 'auto', p: 2 }}>
        <Typography>Se o e-mail existir, você receberá instruções para redefinir a senha.</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Voltar ao login</Link>
        </Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ maxWidth: 400, mx: 'auto', p: 2 }}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="h5" sx={{ mb: 2 }}>Esqueci minha senha</Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <TextField label="E-mail" type="email" fullWidth required value={email} onChange={(e) => setEmail(e.target.value)} sx={{ mb: 2 }} />
          <Button type="submit" variant="contained" fullWidth disabled={loading} sx={{ py: 0.75 }}>
            {loading ? 'Enviando...' : 'Enviar instruções'}
          </Button>
        </Box>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Voltar ao login</Link>
        </Typography>
      </Paper>
    </Box>
  )
}
