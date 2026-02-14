import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Box, Typography, Button, TextField, Paper } from '@mui/material'
import { useAuth } from '../contexts/AuthContext'
import * as api from '../lib/api'

export function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { user, login } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (user) {
      if (user.role === 'SUPER_ADMIN') navigate('/backoffice/audit', { replace: true })
      else navigate('/patients', { replace: true })
    }
  }, [user, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await api.login(email, password)
      login(res.token, res.user)
      if (res.user.role === 'SUPER_ADMIN') navigate('/backoffice/audit', { replace: true })
      else navigate('/patients', { replace: true })
    } catch (err: unknown) {
      const m = err instanceof Error ? err.message : 'Falha no login'
      setError(m.includes('credentials') ? 'E-mail ou senha incorretos.' : m)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Box sx={{ maxWidth: 400, mx: 'auto', p: 2 }}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="h5" sx={{ mb: 2 }}>Entrar</Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <TextField label="E-mail" type="email" fullWidth required value={email} onChange={(e) => setEmail(e.target.value)} sx={{ mb: 2 }} />
          <TextField label="Senha" type="password" fullWidth required value={password} onChange={(e) => setPassword(e.target.value)} sx={{ mb: 2 }} />
          {error && <Typography color="error" sx={{ mb: 2, fontSize: 14 }}>{error}</Typography>}
          <Button type="submit" variant="contained" fullWidth disabled={loading} sx={{ py: 0.75 }}>
            {loading ? 'Entrando...' : 'Entrar'}
          </Button>
        </Box>
        <Typography sx={{ mt: 2, fontSize: 14 }}>
          <Link to="/forgot-password" style={{ color: 'inherit' }}>Esqueci minha senha</Link>
        </Typography>
      </Paper>
    </Box>
  )
}
