import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  TextField,
  Alert,
  Paper,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  IconButton,
  InputAdornment,
} from '@mui/material'
import VisibilityIcon from '@mui/icons-material/Visibility'
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff'
import { useAuth } from '../contexts/AuthContext'
import { PageContainer } from '../components/ui/PageContainer'
import { AppDialog } from '../components/ui/AppDialog'
import * as api from '../lib/api'
import { isValidCPF } from '../lib/cpf'

type UserRow = {
  type: string
  id: string
  email: string
  full_name: string
  status: string
}

export function Backoffice() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [users, setUsers] = useState<UserRow[]>([])
  const [loading, setLoading] = useState(true)
  const [filterName, setFilterName] = useState('')
  const [filterId, setFilterId] = useState('')
  const [filterType, setFilterType] = useState('')

  const [view, setView] = useState<'users' | 'related'>('users')
  const [relatedProfessional, setRelatedProfessional] = useState<UserRow | null>(null)
  const [relatedLoading, setRelatedLoading] = useState(false)
  const [relatedError, setRelatedError] = useState('')
  const [relatedPatients, setRelatedPatients] = useState<{ id: string; full_name: string; birth_date?: string | null }[]>([])
  const [relatedGuardians, setRelatedGuardians] = useState<{ id: string; full_name: string; email: string; status: string; patients_count: number }[]>([])
  const [selectedPatient, setSelectedPatient] = useState<{ id: string; full_name: string } | null>(null)
  const [selectedPatientGuardians, setSelectedPatientGuardians] = useState<api.GuardianInfo[]>([])
  const [selectedPatientLoading, setSelectedPatientLoading] = useState(false)
  const [selectedPatientError, setSelectedPatientError] = useState('')
  const [impersonateReason, setImpersonateReason] = useState('')
  const [impersonateTarget, setImpersonateTarget] = useState<{ type: string; id: string } | null>(null)
  const [editingTarget, setEditingTarget] = useState<{ type: string; id: string } | null>(null)
  const [editLoading, setEditLoading] = useState(false)
  const [editError, setEditError] = useState('')
  const [editEmail, setEditEmail] = useState('')
  const [editFullName, setEditFullName] = useState('')
  const [editTradeName, setEditTradeName] = useState('')
  const [editStatus, setEditStatus] = useState('ACTIVE')
  const [editBirthDate, setEditBirthDate] = useState('')
  const [editStreet, setEditStreet] = useState('')
  const [editNumber, setEditNumber] = useState('')
  const [editComplement, setEditComplement] = useState('')
  const [editNeighborhood, setEditNeighborhood] = useState('')
  const [editCity, setEditCity] = useState('')
  const [editState, setEditState] = useState('')
  const [editCountry, setEditCountry] = useState('')
  const [editZip, setEditZip] = useState('')
  const [editPhone, setEditPhone] = useState('')
  const [editMaritalStatus, setEditMaritalStatus] = useState('')
  const [editCPF, setEditCPF] = useState('')
  const [showCPF, setShowCPF] = useState(false)
  const [editNewPassword, setEditNewPassword] = useState('')
  const [reminderProfessionalId, setReminderProfessionalId] = useState('')
  const [reminderLoading, setReminderLoading] = useState(false)
  const [reminderResult, setReminderResult] = useState<{ sent: number; skipped: number } | null>(null)
  const [reminderError, setReminderError] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    api
      .listBackofficeUsers()
      .then((r) => setUsers(r.users))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (user?.role !== 'SUPER_ADMIN') return
    load()
  }, [user?.role, load])

  const filteredUsers = users.filter((u) => {
    const nameOk = !filterName.trim() || (u.full_name || '').toLowerCase().includes(filterName.trim().toLowerCase())
    const idOk = !filterId.trim() || (u.id || '').toLowerCase().includes(filterId.trim().toLowerCase())
    const typeOk = !filterType || u.type === filterType
    return nameOk && idOk && typeOk
  })

  const handleOpenRelated = async (u: UserRow) => {
    if (u.type !== 'PROFESSIONAL') return
    setRelatedProfessional(u)
    setView('related')
    setRelatedLoading(true)
    setRelatedError('')
    setRelatedPatients([])
    setRelatedGuardians([])
    setSelectedPatient(null)
    setSelectedPatientGuardians([])
    setSelectedPatientError('')
    try {
      const res = await api.getBackofficeProfessionalRelated(u.id)
      setRelatedPatients(res.patients || [])
      setRelatedGuardians(res.guardians || [])
      // UX: se houver pacientes, seleciona o primeiro e carrega responsáveis.
      if ((res.patients || []).length > 0) {
        const p0 = res.patients[0]
        setSelectedPatient({ id: p0.id, full_name: p0.full_name })
        setSelectedPatientLoading(true)
        setSelectedPatientError('')
        api.listPatientGuardians(p0.id)
          .then((r) => setSelectedPatientGuardians(r.guardians || []))
          .catch((e) => setSelectedPatientError((e as Error)?.message || 'Falha ao carregar responsáveis do paciente.'))
          .finally(() => setSelectedPatientLoading(false))
      }
    } catch (e) {
      setRelatedError((e as Error)?.message || 'Falha ao carregar dados relacionados.')
    } finally {
      setRelatedLoading(false)
    }
  }

  const handleSelectPatient = async (p: { id: string; full_name: string }) => {
    setSelectedPatient({ id: p.id, full_name: p.full_name })
    setSelectedPatientLoading(true)
    setSelectedPatientError('')
    setSelectedPatientGuardians([])
    try {
      const res = await api.listPatientGuardians(p.id)
      setSelectedPatientGuardians(res.guardians || [])
    } catch (e) {
      setSelectedPatientError((e as Error)?.message || 'Falha ao carregar responsáveis do paciente.')
    } finally {
      setSelectedPatientLoading(false)
    }
  }

  const handleImpersonate = async () => {
    if (!impersonateTarget || !impersonateReason.trim()) return
    try {
      // Salva o contexto do admin para conseguir restaurar ao encerrar.
      // (Não logamos token/user por conterem dados sensíveis.)
      const adminToken = localStorage.getItem('token')
      const adminUser = localStorage.getItem('user')
      if (adminToken) localStorage.setItem('impersonate_admin_token', adminToken)
      if (adminUser) localStorage.setItem('impersonate_admin_user', adminUser)
      const res = await api.impersonateStart(
        impersonateTarget.type,
        impersonateTarget.id,
        impersonateReason.trim()
      )
      localStorage.setItem('token', res.token)
      localStorage.setItem('impersonating', '1')
      localStorage.setItem('user', JSON.stringify({ ...user, role: impersonateTarget.type, id: impersonateTarget.id }))
      setImpersonateTarget(null)
      setImpersonateReason('')
      window.location.href = '/patients'
    } catch {
      // Se falhar, evita deixar backup "sujo".
      localStorage.removeItem('impersonate_admin_token')
      localStorage.removeItem('impersonate_admin_user')
      alert('Falha ao iniciar impersonate.')
    }
  }

  const handleOpenEdit = async (u: UserRow) => {
    setEditError('')
    setEditLoading(true)
    setEditingTarget({ type: u.type, id: u.id })
    setEditEmail(u.email)
    setEditFullName(u.full_name)
    setEditTradeName('')
    setEditStatus(u.status || 'ACTIVE')
    setEditBirthDate('')
    setEditStreet('')
    setEditNumber('')
    setEditComplement('')
    setEditNeighborhood('')
    setEditCity('')
    setEditState('')
    setEditCountry('')
    setEditZip('')
    setEditPhone('')
    setEditMaritalStatus('')
    setEditCPF('')
    setShowCPF(false)
    setEditNewPassword('')
    try {
      const res = await api.getBackofficeUser(u.type, u.id)
      const d = res.user
      setEditEmail(d.email || u.email)
      setEditFullName(d.full_name || u.full_name)
      setEditTradeName(d.trade_name ? String(d.trade_name) : '')
      setEditStatus(d.status || u.status || 'ACTIVE')
      setEditBirthDate(d.birth_date ? String(d.birth_date) : '')
      const addr = d.address
      if (addr && typeof addr === 'object') {
        setEditStreet(addr.street ?? '')
        setEditNumber(addr.number ?? '')
        setEditComplement(addr.complement ?? '')
        setEditNeighborhood(addr.neighborhood ?? '')
        setEditCity(addr.city ?? '')
        setEditState(addr.state ?? '')
        setEditCountry(addr.country ?? '')
        setEditZip(addr.zip ?? '')
      } else {
        setEditStreet('')
        setEditNumber('')
        setEditComplement('')
        setEditNeighborhood('')
        setEditCity('')
        setEditState('')
        setEditCountry('')
        setEditZip('')
      }
      setEditPhone(d.phone ? String(d.phone) : '')
      setEditMaritalStatus(d.marital_status ? String(d.marital_status) : '')
      setEditCPF(d.cpf ? String(d.cpf) : '')
    } catch {
      setEditError('Falha ao carregar detalhes do usuário.')
    } finally {
      setEditLoading(false)
    }
  }

  const handleSaveEdit = async () => {
    if (!editingTarget) return
    setEditError('')
    if (!editEmail.trim() || !editFullName.trim()) {
      setEditError('E-mail e nome são obrigatórios.')
      return
    }
    if (editCPF.trim() && !isValidCPF(editCPF)) {
      setEditError('CPF inválido.')
      return
    }
    try {
      const payload: Record<string, string | api.Address | undefined> = {
        email: editEmail.trim(),
        full_name: editFullName.trim(),
        status: editStatus,
      }
      const hasAddress = editStreet.trim() || editNeighborhood.trim() || editCity.trim() || editState.trim() || editCountry.trim() || editZip.trim()
      const addressPayload: api.Address | undefined = hasAddress
        ? {
            street: editStreet.trim(),
            number: editNumber.trim() || undefined,
            complement: editComplement.trim() || undefined,
            neighborhood: editNeighborhood.trim(),
            city: editCity.trim(),
            state: editState.trim(),
            country: editCountry.trim(),
            zip: editZip.replace(/\D/g, ''),
          }
        : undefined
      if (editingTarget.type === 'PROFESSIONAL') {
        if (editTradeName.trim()) payload.trade_name = editTradeName.trim()
        payload.birth_date = editBirthDate.trim()
        payload.address = addressPayload
        payload.marital_status = editMaritalStatus
      } else if (editingTarget.type === 'LEGAL_GUARDIAN') {
        payload.birth_date = editBirthDate.trim()
        payload.address = addressPayload
        payload.phone = editPhone.trim()
      }
      if (editCPF.trim()) payload.cpf = editCPF.trim()
      if (editNewPassword.trim()) payload.new_password = editNewPassword

      await api.patchBackofficeUser(editingTarget.type, editingTarget.id, payload as Parameters<typeof api.patchBackofficeUser>[2])
      await load()
      setEditingTarget(null)
    } catch (e) {
      const msg = String((e as Error)?.message || '')
      setEditError(msg.includes('email') || msg.includes('cpf') ? msg : 'Falha ao salvar alterações.')
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
      <Typography variant="h4" sx={{ mb: 1.5 }}>Usuários</Typography>

      <Tabs value={view} onChange={(_, v) => setView(v)} sx={{ mb: 2 }}>
        <Tab value="users" label="Usuários" />
        <Tab value="related" label={relatedProfessional ? `Profissional: ${relatedProfessional.full_name}` : 'Profissional: relacionados'} disabled={!relatedProfessional} />
      </Tabs>

      {view === 'related' && relatedProfessional && (
        <Box sx={{ mb: 2 }}>
          <Button variant="outlined" onClick={() => setView('users')}>Voltar</Button>
        </Box>
      )}

      <Box sx={{ mb: 2, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
        <Button variant="contained" onClick={() => navigate('/backoffice/invites')}>Enviar convite para profissional</Button>
      </Box>

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Typography variant="subtitle1" sx={{ mb: 1 }}>Lembretes de consulta (amanhã)</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          Dispara lembretes WhatsApp para consultas de amanhã. Opcional: ID do profissional para enviar só dele.
        </Typography>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
          <TextField
            size="small"
            label="ID do profissional (opcional)"
            placeholder="UUID do profissional"
            value={reminderProfessionalId}
            onChange={(e) => setReminderProfessionalId(e.target.value)}
            sx={{ minWidth: 280 }}
          />
          <Button
            variant="outlined"
            disabled={reminderLoading}
            onClick={async () => {
              setReminderLoading(true)
              setReminderResult(null)
              setReminderError('')
              try {
                const res = await api.triggerReminder(reminderProfessionalId.trim() || undefined)
                setReminderResult({ sent: res.sent, skipped: res.skipped })
              } catch (e) {
                setReminderError(e instanceof Error ? e.message : 'Erro ao disparar lembretes')
              } finally {
                setReminderLoading(false)
              }
            }}
          >
            {reminderLoading ? 'Disparando...' : 'Disparar lembretes'}
          </Button>
        </Box>
        {reminderResult && (
          <Alert severity="success" sx={{ mt: 1 }}>
            Enviados: {reminderResult.sent}, ignorados: {reminderResult.skipped}
          </Alert>
        )}
        {reminderError && (
          <Alert severity="error" sx={{ mt: 1 }}>{reminderError}</Alert>
        )}
      </Paper>
      <Box sx={{ mb: 2 }}>
        <Button variant="outlined" size="small" onClick={load}>Atualizar</Button>
      </Box>

      {view === 'users' && (
        <>
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 2 }}>
            <TextField size="small" label="Filtrar por nome" value={filterName} onChange={(e) => setFilterName(e.target.value)} />
            <TextField size="small" label="Filtrar por ID" value={filterId} onChange={(e) => setFilterId(e.target.value)} sx={{ minWidth: 260 }} />
            <FormControl size="small" sx={{ minWidth: 200 }}>
              <InputLabel>Tipo</InputLabel>
              <Select value={filterType} label="Tipo" onChange={(e) => setFilterType(String(e.target.value))}>
                <MenuItem value="">Todos</MenuItem>
                <MenuItem value="PROFESSIONAL">PROFESSIONAL</MenuItem>
                <MenuItem value="LEGAL_GUARDIAN">LEGAL_GUARDIAN</MenuItem>
                <MenuItem value="SUPER_ADMIN">SUPER_ADMIN</MenuItem>
              </Select>
            </FormControl>
          </Box>

          {loading && <Typography color="text.secondary">Carregando...</Typography>}
          {!loading && (
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Tipo</TableCell>
                    <TableCell>Nome</TableCell>
                    <TableCell>E-mail</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Ação</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredUsers.map((u) => (
                    <TableRow
                      key={`${u.type}-${u.id}`}
                      hover
                      onClick={() => handleOpenRelated(u)}
                      sx={{ cursor: u.type === 'PROFESSIONAL' ? 'pointer' : 'default' }}
                      title={u.type === 'PROFESSIONAL' ? 'Clique para ver pacientes e responsáveis desta clínica' : undefined}
                    >
                      <TableCell>{u.type}</TableCell>
                      <TableCell>{u.full_name}</TableCell>
                      <TableCell>{u.email}</TableCell>
                      <TableCell>{u.status}</TableCell>
                      <TableCell onClick={(e) => e.stopPropagation()}>
                        {(u.type === 'PROFESSIONAL' || u.type === 'LEGAL_GUARDIAN') && (
                          <Button size="small" variant="outlined" sx={{ mr: 1 }} onClick={() => handleOpenEdit(u)}>
                            Editar
                          </Button>
                        )}
                        {u.type !== 'SUPER_ADMIN' && (
                          <Button size="small" variant="outlined" onClick={() => setImpersonateTarget({ type: u.type, id: u.id })}>Impersonate</Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {filteredUsers.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5}>
                        <Typography color="text.secondary">Nenhum usuário encontrado para os filtros informados.</Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </>
      )}

      {view === 'related' && relatedProfessional && (
        <>
          {relatedError && <Alert severity="error" sx={{ mb: 2 }}>{relatedError}</Alert>}
          {relatedLoading && <Typography color="text.secondary">Carregando dados relacionados...</Typography>}
          {!relatedLoading && !relatedError && (
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2 }}>
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="h6" sx={{ mb: 1 }}>Pacientes ({relatedPatients.length})</Typography>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Nome</TableCell>
                      <TableCell>Nasc.</TableCell>
                      <TableCell>ID</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {relatedPatients.map((p) => (
                      <TableRow
                        key={p.id}
                        hover
                        onClick={() => handleSelectPatient({ id: p.id, full_name: p.full_name })}
                        sx={{ cursor: 'pointer' }}
                        selected={selectedPatient?.id === p.id}
                      >
                        <TableCell>{p.full_name}</TableCell>
                        <TableCell sx={{ whiteSpace: 'nowrap' }}>{p.birth_date ? new Date(String(p.birth_date)).toLocaleDateString('pt-BR') : '—'}</TableCell>
                        <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>{p.id}</TableCell>
                      </TableRow>
                    ))}
                    {relatedPatients.length === 0 && (
                      <TableRow><TableCell colSpan={3}><Typography color="text.secondary">Nenhum paciente.</Typography></TableCell></TableRow>
                    )}
                  </TableBody>
                </Table>
              </Paper>

              <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="h6" sx={{ mb: 1 }}>
                  {selectedPatient ? `Responsáveis legais do paciente` : 'Responsáveis legais'}
                </Typography>
                {selectedPatient && (
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    Paciente: <b>{selectedPatient.full_name}</b>
                  </Typography>
                )}
                {selectedPatientError && <Alert severity="error" sx={{ mb: 1.5 }}>{selectedPatientError}</Alert>}
                {selectedPatientLoading ? (
                  <Typography color="text.secondary">Carregando responsáveis...</Typography>
                ) : selectedPatient ? (
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Nome</TableCell>
                        <TableCell>E-mail</TableCell>
                        <TableCell>Relação</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {selectedPatientGuardians.map((g) => (
                        <TableRow key={g.id}>
                          <TableCell>{g.full_name}</TableCell>
                          <TableCell>{g.email}</TableCell>
                          <TableCell sx={{ whiteSpace: 'nowrap' }}>{g.relation}</TableCell>
                        </TableRow>
                      ))}
                      {selectedPatientGuardians.length === 0 && (
                        <TableRow><TableCell colSpan={3}><Typography color="text.secondary">Nenhum responsável vinculado a este paciente.</Typography></TableCell></TableRow>
                      )}
                    </TableBody>
                  </Table>
                ) : (
                  <Typography color="text.secondary">Selecione um paciente para ver o responsável legal.</Typography>
                )}

                <Box sx={{ mt: 2 }}>
                  <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                    Resumo de responsáveis na clínica ({relatedGuardians.length})
                  </Typography>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Nome</TableCell>
                        <TableCell>Pacientes</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {relatedGuardians.slice(0, 10).map((g) => (
                        <TableRow key={g.id}>
                          <TableCell>{g.full_name}</TableCell>
                          <TableCell sx={{ whiteSpace: 'nowrap' }}>{g.patients_count}</TableCell>
                        </TableRow>
                      ))}
                      {relatedGuardians.length === 0 && (
                        <TableRow><TableCell colSpan={2}><Typography color="text.secondary">Nenhum responsável.</Typography></TableCell></TableRow>
                      )}
                    </TableBody>
                  </Table>
                  {relatedGuardians.length > 10 && (
                    <Typography variant="caption" color="text.secondary">
                      Mostrando 10 de {relatedGuardians.length}. (Clique em um paciente para ver os responsáveis específicos.)
                    </Typography>
                  )}
                </Box>
              </Paper>
            </Box>
          )}
        </>
      )}

      <AppDialog open={!!impersonateTarget} onClose={() => { setImpersonateTarget(null); setImpersonateReason('') }} title="Impersonate – motivo obrigatório" actions={
        <>
          <Button onClick={() => { setImpersonateTarget(null); setImpersonateReason('') }} color="inherit">Cancelar</Button>
          <Button variant="contained" onClick={handleImpersonate} disabled={!impersonateReason.trim()}>Iniciar impersonate</Button>
        </>
      }>
        {impersonateTarget && (
          <>
            <Typography sx={{ mb: 1 }}>Alvo: {impersonateTarget.type} / {impersonateTarget.id}</Typography>
            <TextField fullWidth multiline rows={3} label="Motivo" placeholder="Ex.: Suporte ticket #123" value={impersonateReason} onChange={(e) => setImpersonateReason(e.target.value)} />
          </>
        )}
      </AppDialog>

      <AppDialog
        open={!!editingTarget}
        onClose={() => { setEditingTarget(null); setEditError(''); setShowCPF(false) }}
        title={editingTarget ? `Editar usuário (${editingTarget.type})` : 'Editar usuário'}
        maxWidth="sm"
        actions={
          <>
            <Button onClick={() => { setEditingTarget(null); setEditError(''); setShowCPF(false) }} color="inherit">Cancelar</Button>
            <Button variant="contained" onClick={handleSaveEdit} disabled={editLoading}>
              {editLoading ? 'Carregando...' : 'Salvar'}
            </Button>
          </>
        }
      >
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 0.5 }}>
          {editError && <Alert severity="error">{editError}</Alert>}
          <TextField label="E-mail" value={editEmail} onChange={(e) => setEditEmail(e.target.value)} fullWidth />
          <TextField label="Nome" value={editFullName} onChange={(e) => setEditFullName(e.target.value)} fullWidth />
          <FormControl fullWidth>
            <InputLabel id="status-label">Status</InputLabel>
            <Select labelId="status-label" label="Status" value={editStatus} onChange={(e) => setEditStatus(String(e.target.value))}>
              <MenuItem value="ACTIVE">ACTIVE</MenuItem>
              <MenuItem value="SUSPENDED">SUSPENDED</MenuItem>
              <MenuItem value="CANCELLED">CANCELLED</MenuItem>
            </Select>
          </FormControl>

          {editingTarget?.type === 'PROFESSIONAL' && (
            <>
              <TextField label="Nome fantasia" value={editTradeName} onChange={(e) => setEditTradeName(e.target.value)} fullWidth />
              <TextField label="Data de nascimento" type="date" value={editBirthDate} onChange={(e) => setEditBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
              <TextField
                label="CPF (opcional, para alterar)"
                type={showCPF ? 'text' : 'password'}
                value={editCPF}
                onChange={(e) => setEditCPF(e.target.value)}
                placeholder="Somente números"
                fullWidth
                autoComplete="off"
                inputMode="numeric"
                error={!!editCPF.trim() && !isValidCPF(editCPF)}
                helperText={editCPF.trim() && !isValidCPF(editCPF) ? 'CPF inválido.' : ' '}
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton onClick={() => setShowCPF((v) => !v)} edge="end" aria-label={showCPF ? 'Ocultar CPF' : 'Mostrar CPF'}>
                        {showCPF ? <VisibilityOffIcon /> : <VisibilityIcon />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
              <Typography variant="subtitle2" color="text.secondary">Endereço</Typography>
              <TextField label="Rua" value={editStreet} onChange={(e) => setEditStreet(e.target.value)} fullWidth />
              <TextField label="Número" value={editNumber} onChange={(e) => setEditNumber(e.target.value)} fullWidth />
              <TextField label="Complemento" value={editComplement} onChange={(e) => setEditComplement(e.target.value)} fullWidth />
              <TextField label="Bairro" value={editNeighborhood} onChange={(e) => setEditNeighborhood(e.target.value)} fullWidth />
              <TextField label="Cidade" value={editCity} onChange={(e) => setEditCity(e.target.value)} fullWidth />
              <TextField label="Estado (UF)" value={editState} onChange={(e) => setEditState(e.target.value)} placeholder="UF" inputProps={{ maxLength: 2 }} fullWidth />
              <TextField label="País" value={editCountry} onChange={(e) => setEditCountry(e.target.value)} fullWidth />
              <TextField label="CEP" value={editZip} onChange={(e) => setEditZip(e.target.value)} placeholder="00000000" inputProps={{ maxLength: 9 }} fullWidth />
              <TextField label="Estado civil" value={editMaritalStatus} onChange={(e) => setEditMaritalStatus(e.target.value)} fullWidth />
            </>
          )}

          {editingTarget?.type === 'LEGAL_GUARDIAN' && (
            <>
              <TextField label="Data de nascimento" type="date" value={editBirthDate} onChange={(e) => setEditBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
              <Typography variant="subtitle2" color="text.secondary">Endereço</Typography>
              <TextField label="Rua" value={editStreet} onChange={(e) => setEditStreet(e.target.value)} fullWidth />
              <TextField label="Número" value={editNumber} onChange={(e) => setEditNumber(e.target.value)} fullWidth />
              <TextField label="Complemento" value={editComplement} onChange={(e) => setEditComplement(e.target.value)} fullWidth />
              <TextField label="Bairro" value={editNeighborhood} onChange={(e) => setEditNeighborhood(e.target.value)} fullWidth />
              <TextField label="Cidade" value={editCity} onChange={(e) => setEditCity(e.target.value)} fullWidth />
              <TextField label="Estado (UF)" value={editState} onChange={(e) => setEditState(e.target.value)} placeholder="UF" inputProps={{ maxLength: 2 }} fullWidth />
              <TextField label="País" value={editCountry} onChange={(e) => setEditCountry(e.target.value)} fullWidth />
              <TextField label="CEP" value={editZip} onChange={(e) => setEditZip(e.target.value)} placeholder="00000000" inputProps={{ maxLength: 9 }} fullWidth />
              <TextField label="Telefone (WhatsApp)" value={editPhone} onChange={(e) => setEditPhone(e.target.value)} placeholder="+5511999999999" fullWidth />
              <TextField
                label="CPF (opcional, para alterar)"
                type={showCPF ? 'text' : 'password'}
                value={editCPF}
                onChange={(e) => setEditCPF(e.target.value)}
                placeholder="Somente números"
                fullWidth
                autoComplete="off"
                inputMode="numeric"
                error={!!editCPF.trim() && !isValidCPF(editCPF)}
                helperText={editCPF.trim() && !isValidCPF(editCPF) ? 'CPF inválido.' : ' '}
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton onClick={() => setShowCPF((v) => !v)} edge="end" aria-label={showCPF ? 'Ocultar CPF' : 'Mostrar CPF'}>
                        {showCPF ? <VisibilityOffIcon /> : <VisibilityIcon />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
            </>
          )}

          <TextField
            label="Nova senha (opcional)"
            type="password"
            value={editNewPassword}
            onChange={(e) => setEditNewPassword(e.target.value)}
            placeholder="mín. 8 caracteres"
            fullWidth
          />
        </Box>
      </AppDialog>
    </PageContainer>
  )
}
