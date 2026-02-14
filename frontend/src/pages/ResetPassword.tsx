import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Box, Typography, Button, TextField, Paper } from '@mui/material'
import * as api from '../lib/api'

export function ResetPassword() {
  const [searchParams] = useSearchParams()
  const tokenFromUrl = searchParams.get('token') || ''
  const [token, setToken] = useState(tokenFromUrl)
  const [newPassword, setNewPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPassword !== confirm) {
      setError('As senhas não coincidem.')
      return
    }
    if (newPassword.length < 8) {
      setError('Senha deve ter no mínimo 8 caracteres.')
      return
    }
    setError('')
    setLoading(true)
    try {
      await api.resetPassword(token, newPassword)
      setDone(true)
    } catch {
      setError('Token inválido ou expirado.')
    } finally {
      setLoading(false)
    }
  }

  if (done) {
    return (
      <Box sx={{ maxWidth: 400, mx: 'auto', p: 2 }}>
        <Typography>Senha alterada com sucesso.</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Fazer login</Link>
        </Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ maxWidth: 400, mx: 'auto', p: 2 }}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="h5" sx={{ mb: 2 }}>Redefinir senha</Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <TextField label="Token" fullWidth required value={token} onChange={(e) => setToken(e.target.value)} sx={{ mb: 2 }} />
          <TextField label="Nova senha" type="password" fullWidth required inputProps={{ minLength: 8 }} value={newPassword} onChange={(e) => setNewPassword(e.target.value)} sx={{ mb: 2 }} />
          <TextField label="Confirmar senha" type="password" fullWidth required value={confirm} onChange={(e) => setConfirm(e.target.value)} sx={{ mb: 2 }} />
          {error && <Typography color="error" sx={{ mb: 2, fontSize: 14 }}>{error}</Typography>}
          <Button type="submit" variant="contained" fullWidth disabled={loading} sx={{ py: 0.75 }}>
            {loading ? 'Alterando...' : 'Alterar senha'}
          </Button>
        </Box>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Voltar ao login</Link>
        </Typography>
      </Paper>
    </Box>
  )
}
