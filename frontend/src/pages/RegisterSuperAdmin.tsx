import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Box, Typography, Button, TextField, Paper } from '@mui/material'
import * as api from '../lib/api'

export function RegisterSuperAdmin() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const [invite, setInvite] = useState<{ email: string; full_name: string; expires_at: string } | null>(null)
  const [loading, setLoading] = useState(!!token)
  const [error, setError] = useState('')
  const [fullName, setFullName] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    if (!token) {
      setLoading(false)
      return
    }
    api
      .getSuperAdminInviteByToken(token)
      .then((data) => {
        setInvite(data)
        setFullName(data.full_name)
      })
      .catch(() => setError('Link inválido ou expirado.'))
      .finally(() => setLoading(false))
  }, [token])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (password !== confirmPassword) {
      setError('As senhas não coincidem.')
      return
    }
    if (password.length < 8) {
      setError('A senha deve ter pelo menos 8 caracteres.')
      return
    }
    setSubmitting(true)
    try {
      await api.acceptSuperAdminInvite({
        token,
        password,
        full_name: fullName || undefined,
      })
      setSuccess(true)
    } catch (err: unknown) {
      const m = err instanceof Error ? err.message : 'Não foi possível concluir o cadastro.'
      setError(m)
    } finally {
      setSubmitting(false)
    }
  }

  if (!token) {
    return (
      <Box sx={{ maxWidth: 400, mx: 'auto', p: 2, textAlign: 'center' }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Link inválido</Typography>
        <Typography>Use o link recebido por e-mail para acessar o formulário de cadastro.</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Ir para o login</Link>
        </Typography>
      </Box>
    )
  }

  if (loading) {
    return (
      <Box sx={{ maxWidth: 400, mx: 'auto', p: 2 }}>
        <Typography color="text.secondary">Carregando...</Typography>
      </Box>
    )
  }

  if (error && !invite) {
    return (
      <Box sx={{ maxWidth: 400, mx: 'auto', p: 2, textAlign: 'center' }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Link inválido ou expirado</Typography>
        <Typography>{error}</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Ir para o login</Link>
        </Typography>
      </Box>
    )
  }

  if (success) {
    return (
      <Box sx={{ maxWidth: 400, mx: 'auto', p: 2, textAlign: 'center' }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Cadastro concluído</Typography>
        <Typography>Faça login como super admin com seu e-mail e senha.</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ fontWeight: 600, color: 'inherit' }}>Ir para o login</Link>
        </Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ maxWidth: 480, mx: 'auto', p: 2 }}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Concluir cadastro (Super Admin)</Typography>
        <Typography color="text.secondary" sx={{ mb: 2, fontSize: 14 }}>
          Você foi convidado a acessar o backoffice. Defina sua senha abaixo.
        </Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <TextField label="E-mail" type="email" fullWidth value={invite?.email ?? ''} InputProps={{ readOnly: true }} sx={{ mb: 1.5 }} />
          <TextField label="Nome completo" fullWidth required value={fullName} onChange={(e) => setFullName(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Senha" type="password" fullWidth required inputProps={{ minLength: 8 }} placeholder="Mínimo 8 caracteres" value={password} onChange={(e) => setPassword(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Confirmar senha" type="password" fullWidth required inputProps={{ minLength: 8 }} value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} sx={{ mb: 1.5 }} />
          {error && <Typography color="error" sx={{ mb: 1.5, fontSize: 14 }}>{error}</Typography>}
          <Button type="submit" variant="contained" fullWidth disabled={submitting} sx={{ py: 0.75 }}>
            {submitting ? 'Salvando...' : 'Concluir cadastro'}
          </Button>
        </Box>
        <Typography sx={{ mt: 2, fontSize: 14 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Já tem conta? Entrar</Link>
        </Typography>
      </Paper>
    </Box>
  )
}

