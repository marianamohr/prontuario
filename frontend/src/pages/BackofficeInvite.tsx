import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  TextField,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
} from '@mui/material'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import SendIcon from '@mui/icons-material/Send'
import { useAuth } from '../contexts/AuthContext'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'

function formatDate(s: string) {
  if (!s) return '—'
  const d = new Date(s)
  return d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function isPendingAndNotExpired(item: api.BackofficeInviteItem) {
  if (item.status !== 'PENDING') return false
  return new Date(item.expires_at) > new Date()
}

export function BackofficeInvite() {
  const { user } = useAuth()
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteFullName, setInviteFullName] = useState('')
  const [inviteSending, setInviteSending] = useState(false)
  const [inviteSuccess, setInviteSuccess] = useState('')
  const [invites, setInvites] = useState<api.BackofficeInviteItem[]>([])
  const [loading, setLoading] = useState(false)
  const [actionId, setActionId] = useState<string | null>(null)

  const loadInvites = useCallback(async () => {
    if (user?.role !== 'SUPER_ADMIN') return
    setLoading(true)
    try {
      const list = await api.listInvites()
      setInvites(list)
    } catch {
      setInvites([])
    } finally {
      setLoading(false)
    }
  }, [user?.role])

  useEffect(() => {
    loadInvites()
  }, [loadInvites])

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
      await loadInvites()
    } catch {
      setInviteSuccess('')
      alert('Falha ao enviar convite.')
    } finally {
      setInviteSending(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm('Remover este convite? O link deixará de funcionar.')) return
    setActionId(id)
    try {
      await api.deleteInvite(id)
      await loadInvites()
    } catch {
      alert('Falha ao remover convite.')
    } finally {
      setActionId(null)
    }
  }

  const handleResend = async (id: string) => {
    setActionId(id)
    try {
      await api.resendInvite(id)
      await loadInvites()
    } catch (e: unknown) {
      let msg = 'Falha ao reenviar convite.'
      if (e instanceof Error && e.message) {
        try {
          const o = JSON.parse(e.message) as { error?: string }
          if (o.error) msg = o.error
        } catch {
          msg = e.message
        }
      }
      alert(msg)
    } finally {
      setActionId(null)
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
      <Typography variant="h4" sx={{ mb: 2 }}>Convites para profissional</Typography>

      <Paper variant="outlined" sx={{ p: 2, maxWidth: 420, mb: 3 }}>
        <Typography variant="subtitle1" sx={{ mb: 1.5 }}>Enviar novo convite</Typography>
        <Box component="form" onSubmit={handleSendInvite}>
          <TextField label="E-mail" type="email" fullWidth required placeholder="email@exemplo.com" value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Nome completo" fullWidth required placeholder="Nome do profissional" value={inviteFullName} onChange={(e) => setInviteFullName(e.target.value)} sx={{ mb: 1.5 }} />
          {inviteSuccess && <Typography color="success.main" sx={{ mb: 1, fontSize: 14 }}>{inviteSuccess}</Typography>}
          <Button type="submit" variant="contained" disabled={inviteSending}>{inviteSending ? 'Enviando...' : 'Enviar convite'}</Button>
        </Box>
      </Paper>

      <Typography variant="subtitle1" sx={{ mb: 1 }}>Convites enviados</Typography>
      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>E-mail</TableCell>
              <TableCell>Nome</TableCell>
              <TableCell>Status</TableCell>
              <TableCell>Enviado em</TableCell>
              <TableCell>Expira em</TableCell>
              <TableCell align="right">Ações</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading && invites.length === 0 ? (
              <TableRow><TableCell colSpan={6}>Carregando...</TableCell></TableRow>
            ) : invites.length === 0 ? (
              <TableRow><TableCell colSpan={6}>Nenhum convite enviado.</TableCell></TableRow>
            ) : (
              invites.map((inv) => (
                <TableRow key={inv.id}>
                  <TableCell>{inv.email}</TableCell>
                  <TableCell>{inv.full_name}</TableCell>
                  <TableCell>{inv.status === 'PENDING' && new Date(inv.expires_at) <= new Date() ? 'Expirado' : inv.status}</TableCell>
                  <TableCell>{formatDate(inv.created_at)}</TableCell>
                  <TableCell>{formatDate(inv.expires_at)}</TableCell>
                  <TableCell align="right">
                    <IconButton
                      size="small"
                      color="primary"
                      title="Reenviar e-mail"
                      disabled={!isPendingAndNotExpired(inv) || actionId !== null}
                      onClick={() => handleResend(inv.id)}
                    >
                      <SendIcon fontSize="small" />
                    </IconButton>
                    <IconButton
                      size="small"
                      color="error"
                      title="Excluir convite"
                      disabled={actionId !== null}
                      onClick={() => handleDelete(inv.id)}
                    >
                      <DeleteOutlineIcon fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </PageContainer>
  )
}
