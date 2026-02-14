import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Box, Typography, Button, TextField, Paper } from '@mui/material'
import { useAuth } from '../contexts/AuthContext'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'

export function BackofficeInvite() {
  const { user } = useAuth()
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteFullName, setInviteFullName] = useState('')
  const [inviteSending, setInviteSending] = useState(false)
  const [inviteSuccess, setInviteSuccess] = useState('')

  useEffect(() => {
    if (user?.role !== 'SUPER_ADMIN') return
  }, [user?.role])

  const handleSendInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!inviteEmail.trim() || !inviteFullName.trim()) return
    setInviteSuccess('')
    setInviteSending(true)
    try {
      await api.createInvite(inviteEmail.trim(), inviteFullName.trim())
      setInviteSuccess('Convite enviado por e-mail.')
      setInviteEmail('')
      setInviteFullName('')
    } catch {
      setInviteSuccess('')
      alert('Falha ao enviar convite.')
    } finally {
      setInviteSending(false)
    }
  }

  if (user?.role !== 'SUPER_ADMIN') {
    return (
      <PageContainer>
        <Typography>Acesso negado. Apenas super admin.</Typography>
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      <Typography component={Link} to="/backoffice" sx={{ display: 'block', mb: 2, color: 'primary.main', textDecoration: 'none', fontSize: 14 }}>← Voltar ao backoffice</Typography>
      <Typography variant="h4" sx={{ mb: 2 }}>Enviar convite para profissional</Typography>
      <Paper variant="outlined" sx={{ p: 2, maxWidth: 420 }}>
        <Box component="form" onSubmit={handleSendInvite}>
          <TextField label="E-mail" type="email" fullWidth required placeholder="email@exemplo.com" value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Nome completo" fullWidth required placeholder="Nome do profissional" value={inviteFullName} onChange={(e) => setInviteFullName(e.target.value)} sx={{ mb: 1.5 }} />
          {inviteSuccess && <Typography color="success.main" sx={{ mb: 1, fontSize: 14 }}>{inviteSuccess}</Typography>}
          <Button type="submit" variant="contained" disabled={inviteSending}>{inviteSending ? 'Enviando...' : 'Enviar convite'}</Button>
        </Box>
      </Paper>
    </PageContainer>
  )
}
