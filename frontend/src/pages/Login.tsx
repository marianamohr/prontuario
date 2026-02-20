import { useEffect, useState } from 'react'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { Box, Typography, Button, TextField, Link } from '@mui/material'
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
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: { xs: 'column', md: 'row' },
      }}
    >
      {/* Coluna esquerda: hero da landing (oculto no celular) */}
      <Box
        sx={{
          display: { xs: 'none', md: 'flex' },
          flex: '1 1 50%',
          flexDirection: 'column',
          justifyContent: 'flex-start',
          alignItems: 'center',
          px: { md: 5, lg: 7 },
          py: { md: 6, lg: 7 },
          bgcolor: 'background.default',
        }}
      >
        <Box sx={{ width: '100%', maxWidth: 560, my: 'auto', mx: 'auto' }}>
          <Typography variant="h6" fontWeight={700} color="primary" sx={{ mb: 6, fontSize: 20, lineHeight: 1.2 }}>
            Camihealth
          </Typography>
          <Typography
            variant="h3"
            fontWeight={800}
            sx={{ lineHeight: 1.05, mb: 2.5, fontSize: 40 }}
          >
            Seu consultório,{' '}
            <Typography component="span" variant="inherit" fontWeight={800} color="primary">
              sempre com você.
            </Typography>
          </Typography>
          <Typography variant="body1" color="text.secondary" sx={{ mb: 4, maxWidth: 520, lineHeight: 1.65 }}>
            Prontuário, agenda, contratos e atendimentos em um só lugar.
            Feito para profissionais de saúde que atendem por conta própria.
          </Typography>
          <Box
            sx={{
              width: '100%',
              maxWidth: 520,
              borderRadius: 4,
              overflow: 'hidden',
              boxShadow: '0 4px 20px rgba(0,0,0,0.08)',
            }}
          >
            <img
              src="/hero-image.jpg"
              alt="Interface do Camihealth"
              style={{ width: '100%', height: 'auto', display: 'block' }}
            />
          </Box>
        </Box>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', pb: 2 }}>
          © {new Date().getFullYear()} Camihealth. Todos os direitos reservados.
        </Typography>
      </Box>

      {/* Coluna direita: formulário de login */}
      <Box
        sx={{
          flex: { xs: '1 1 auto', md: '1 1 50%' },
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          p: 2,
          bgcolor: { xs: 'background.default', md: 'background.paper' },
        }}
      >
        <Box
          sx={{
            width: '100%',
            maxWidth: 420,
            borderRadius: 4,
            boxShadow: '0 4px 24px rgba(0,0,0,0.06)',
            bgcolor: 'background.paper',
            p: 4,
          }}
        >
          <Typography variant="h5" fontWeight={600} sx={{ mb: 0.5 }}>Entrar</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Acesse sua conta para gerenciar seu consultório.
          </Typography>
          <Box component="form" onSubmit={handleSubmit}>
            <TextField
              label="E-mail"
              type="email"
              placeholder="seu@email.com"
              fullWidth
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              sx={{ mb: 2, '& .MuiOutlinedInput-root': { borderRadius: 3 } }}
            />
            <TextField
              label="Senha"
              type="password"
              fullWidth
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              sx={{ mb: 2, '& .MuiOutlinedInput-root': { borderRadius: 3 } }}
            />
            {error && <Typography color="error" sx={{ mb: 2, fontSize: 14 }}>{error}</Typography>}
            <Button
              type="submit"
              variant="contained"
              fullWidth
              disabled={loading}
              sx={{ py: 1.25, borderRadius: 3 }}
            >
              {loading ? 'Entrando...' : 'Entrar'}
            </Button>
          </Box>
          <Typography sx={{ mt: 2, fontSize: 14 }}>
            <RouterLink to="/forgot-password" style={{ color: 'inherit' }}>Esqueci minha senha</RouterLink>
          </Typography>
          <Typography sx={{ mt: 1.5, fontSize: 14, color: 'text.secondary' }}>
            Ainda não tem conta?{' '}
            <Link component={RouterLink} to="/register" color="primary" sx={{ fontWeight: 600 }}>
              Criar conta
            </Link>
          </Typography>
        </Box>
      </Box>
    </Box>
  )
}
