import { useCallback, useEffect, useMemo, useState } from 'react'
import { Navigate } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Paper,
  IconButton,
} from '@mui/material'
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'
import { useTheme, alpha } from '@mui/material/styles'
import { PageContainer } from '../components/ui/PageContainer'
import { AppDialog } from '../components/ui/AppDialog'
import * as api from '../lib/api'

const DAY_NAMES = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']
const HOUR_START = 7
const HOUR_END = 20
const PX_PER_HOUR = 52

function formatDateBR(s: string) {
  const d = new Date(s + 'T12:00:00')
  return d.toLocaleDateString('pt-BR')
}

function getMonday(date: Date): Date {
  const d = new Date(date)
  const day = d.getDay()
  const diff = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + diff)
  return d
}

function toYYYYMMDD(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function addDays(d: Date, n: number): Date {
  const r = new Date(d)
  r.setDate(r.getDate() + n)
  return r
}

function timeToMinutes(t: string): number {
  const [h, m] = t.split(':').map(Number)
  return (h ?? 0) * 60 + (m ?? 0)
}

export function Agenda() {
  const theme = useTheme()
  const { user, isImpersonated } = useAuth()
  const branding = useBranding()?.branding ?? null
  const actionColor = user?.role === 'PROFESSIONAL' && branding?.action_button_color ? branding.action_button_color : theme.palette.primary.main
  const [appointments, setAppointments] = useState<api.AppointmentItem[]>([])
  const [loading, setLoading] = useState(true)
  const [weekStart, setWeekStart] = useState(() => toYYYYMMDD(getMonday(new Date())))
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editDate, setEditDate] = useState('')
  const [editTime, setEditTime] = useState('')
  const [editStatus, setEditStatus] = useState('')
  const [cancellingId, setCancellingId] = useState<string | null>(null)
  const [message, setMessage] = useState('')
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [contractsForAgenda, setContractsForAgenda] = useState<api.ContractForAgendaItem[]>([])
  const [selectedContractId, setSelectedContractId] = useState('')
  const [newSlots, setNewSlots] = useState<{ appointment_date: string; start_time: string }[]>([{ appointment_date: '', start_time: '09:00' }])
  const [createAvailableSlots, setCreateAvailableSlots] = useState<api.AvailableSlotItem[]>([])
  const [creating, setCreating] = useState(false)
  const [createMessage, setCreateMessage] = useState('')
  const [popoverId, setPopoverId] = useState<string | null>(null)

  const from = weekStart
  const to = toYYYYMMDD(addDays(new Date(weekStart + 'T12:00:00'), 6))

  const load = useCallback(() => {
    api.listAppointments(from, to)
      .then((r) => setAppointments(r.appointments || []))
      .catch(() => setAppointments([]))
      .finally(() => setLoading(false))
  }, [from, to])

  useEffect(() => {
    setLoading(true)
    load()
  }, [load])

  const startEdit = (a: api.AppointmentItem) => {
    setEditingId(a.id)
    setEditDate(a.appointment_date)
    setEditTime(a.start_time)
    setEditStatus(a.status)
    setPopoverId(null)
  }

  const saveEdit = async () => {
    if (!editingId) return
    setMessage('')
    try {
      await api.patchAppointment(editingId, { appointment_date: editDate, start_time: editTime, status: editStatus })
      setMessage('Compromisso atualizado.')
      setEditingId(null)
      load()
    } catch {
      setMessage('Falha ao atualizar.')
    }
  }

  const cancelEdit = () => {
    setEditingId(null)
  }

  const handleCancelAppointment = async (id: string) => {
    setMessage('')
    setCancellingId(id)
    setPopoverId(null)
    try {
      await api.patchAppointment(id, { status: 'CANCELLED' })
      setMessage('Agendamento cancelado.')
      load()
    } catch {
      setMessage('Falha ao cancelar agendamento.')
    } finally {
      setCancellingId(null)
    }
  }

  useEffect(() => {
    if (createModalOpen) {
      api.listContractsForAgenda()
        .then((r) => {
          setContractsForAgenda(r.contracts || [])
          if (r.contracts?.length && !selectedContractId) setSelectedContractId(r.contracts[0].id)
        })
        .catch(() => setContractsForAgenda([]))
      const from = toYYYYMMDD(new Date())
      const toDate = new Date()
      toDate.setDate(toDate.getDate() + 12 * 7)
      const to = toYYYYMMDD(toDate)
      api.listAvailableSlots(from, to)
        .then((r) => setCreateAvailableSlots(r.slots || []))
        .catch(() => setCreateAvailableSlots([]))
    }
  }, [createModalOpen, selectedContractId])

  const createModalUniqueDates = useMemo(() => {
    const dates = new Set<string>()
    for (const s of createAvailableSlots) dates.add(s.date)
    return Array.from(dates).sort()
  }, [createAvailableSlots])

  const getTimesForDate = (date: string) => {
    const times = createAvailableSlots.filter((s) => s.date === date).map((s) => s.start_time)
    return Array.from(new Set(times)).sort()
  }

  const handleOpenCreateModal = () => {
    setCreateMessage('')
    setSelectedContractId(contractsForAgenda[0]?.id || '')
    setNewSlots([{ appointment_date: '', start_time: '' }])
    setCreateModalOpen(true)
  }

  const handleAddSlot = () => {
    const first = createAvailableSlots[0]
    setNewSlots((prev) => [...prev, first ? { appointment_date: first.date, start_time: first.start_time } : { appointment_date: '', start_time: '' }])
  }

  const handleRemoveSlot = (i: number) => {
    setNewSlots((prev) => prev.filter((_, j) => j !== i))
  }

  const handleCreateAppointments = async () => {
    if (!selectedContractId) return
    const valid = newSlots.filter((s) => s.appointment_date && s.start_time)
    if (valid.length === 0) {
      setCreateMessage('Adicione ao menos uma data e horário.')
      return
    }
    setCreating(true)
    setCreateMessage('')
    try {
      const res = await api.createAppointments(selectedContractId, valid)
      setCreateMessage(res.created ? `${res.created} agendamento(s) criado(s).` : 'Nenhum agendamento criado.')
      if (res.created) {
        setCreateModalOpen(false)
        load()
      }
    } catch {
      setCreateMessage('Falha ao criar agendamentos.')
    } finally {
      setCreating(false)
    }
  }

  const goPrevWeek = () => setWeekStart(toYYYYMMDD(addDays(new Date(weekStart + 'T12:00:00'), -7)))
  const goNextWeek = () => setWeekStart(toYYYYMMDD(addDays(new Date(weekStart + 'T12:00:00'), 7)))
  const goToday = () => setWeekStart(toYYYYMMDD(getMonday(new Date())))

  const weekDates = Array.from({ length: 7 }, (_, i) => addDays(new Date(weekStart + 'T12:00:00'), i))
  const totalHours = HOUR_END - HOUR_START
  const gridHeight = totalHours * PX_PER_HOUR

  const appointmentsByDay = weekDates.map((d) => {
    const key = toYYYYMMDD(d)
    return appointments.filter((a) => a.appointment_date === key && a.status !== 'CANCELLED')
  })

  if (user?.role === 'SUPER_ADMIN' && !isImpersonated) {
    return <Navigate to="/backoffice" replace />
  }
  if (user?.role !== 'PROFESSIONAL' && user?.role !== 'SUPER_ADMIN') {
    return (
      <PageContainer>
        <Typography>Apenas profissionais podem acessar a agenda.</Typography>
      </PageContainer>
    )
  }

  const weekLabel = `${formatDateBR(weekStart)} – ${formatDateBR(to)}`

  const statusColor = (status: string) => {
    if (status === 'CONFIRMADO') return 'success.light'
    if (status === 'COMPLETED') return 'info.light'
    if (status === 'PRE_AGENDADO') return 'warning.light'
    if (status === 'AGENDADO') return 'primary.light'
    if (status === 'CANCELLED' || status === 'SERIES_ENDED') return 'error.light'
    return 'grey.300'
  }

  return (
    <PageContainer>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1, mb: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.25 }}>
          <IconButton onClick={goPrevWeek} aria-label="Semana anterior" size="small"><ChevronLeftIcon /></IconButton>
          <IconButton onClick={goNextWeek} aria-label="Próxima semana" size="small"><ChevronRightIcon /></IconButton>
          <Button variant="outlined" size="small" onClick={goToday} sx={{ ml: 0.5 }}>Hoje</Button>
        </Box>
        <Typography variant="h5" fontWeight={600} sx={{ flex: 1 }}>{weekLabel}</Typography>
        <Button variant="contained" onClick={handleOpenCreateModal} sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>Criar agendamentos</Button>
      </Box>
      {message && (
        <Typography sx={{ mb: 1, color: message.includes('Falha') ? 'error.main' : 'success.main', fontSize: 14 }}>{message}</Typography>
      )}

      {loading ? (
        <Typography color="text.secondary">Carregando...</Typography>
      ) : (
        <Paper variant="outlined" sx={{ display: 'grid', gridTemplateColumns: '48px repeat(7, minmax(0, 1fr))', gridTemplateRows: `auto ${gridHeight}px`, overflow: 'hidden', minHeight: 400 }}>
          <Box sx={{ gridColumn: 1, gridRow: 1, borderRight: 1, borderBottom: 1, borderColor: 'divider', bgcolor: 'grey.50' }} />
          {weekDates.map((d, j) => (
            <Box key={j} sx={{ gridColumn: j + 2, gridRow: 1, py: 0.5, textAlign: 'center', borderBottom: 1, borderRight: j < 6 ? 1 : 0, borderColor: 'divider', bgcolor: 'grey.50', fontSize: 13, fontWeight: 500 }}>
              <Typography variant="caption" color="text.secondary">{DAY_NAMES[d.getDay()]}</Typography>
              <Typography variant="body2">{d.getDate()}</Typography>
            </Box>
          ))}
          <Box sx={{ gridColumn: 1, gridRow: 2, display: 'flex', flexDirection: 'column', borderRight: 1, borderColor: 'divider', bgcolor: 'grey.50' }}>
            {Array.from({ length: totalHours }, (_, i) => (
              <Box key={i} sx={{ height: PX_PER_HOUR, pr: 0.35, textAlign: 'right', fontSize: 11, color: 'text.secondary', lineHeight: `${PX_PER_HOUR}px` }}>
                {String(HOUR_START + i).padStart(2, '0')}:00
              </Box>
            ))}
          </Box>
          {weekDates.map((_, dayIndex) => (
            <Box
              key={dayIndex}
              role="gridcell"
              sx={{
                gridColumn: dayIndex + 2,
                gridRow: 2,
                position: 'relative',
                borderRight: dayIndex < 6 ? 1 : 0,
                borderColor: 'divider',
                background: `repeating-linear-gradient( transparent, transparent ${PX_PER_HOUR - 1}px, ${theme.palette.grey[200]} ${PX_PER_HOUR - 1}px, ${theme.palette.grey[200]} ${PX_PER_HOUR}px)`,
              }}
              onClick={() => setPopoverId(null)}
            >
              {appointmentsByDay[dayIndex].map((a) => {
                const startM = timeToMinutes(a.start_time)
                const endM = timeToMinutes(a.end_time)
                const baseM = HOUR_START * 60
                const topPx = Math.max(0, (startM - baseM) / 60 * PX_PER_HOUR)
                const heightPx = Math.max(20, (endM - startM) / 60 * PX_PER_HOUR)
                const showPopover = popoverId === a.id
                return (
                  <Paper
                    key={a.id}
                    elevation={0}
                    sx={(theme) => ({
                      position: 'absolute',
                      left: 4,
                      right: 4,
                      top: topPx + 2,
                      height: heightPx - 4,
                      bgcolor: a.status === 'COMPLETED'
                        ? alpha(theme.palette.info.main, 0.14)
                        : a.status === 'CANCELLED'
                          ? alpha(theme.palette.error.main, 0.1)
                          : alpha(theme.palette.primary.main, 0.14),
                      border: '1px solid',
                      borderColor: a.status === 'CANCELLED' ? 'error.light' : 'primary.light',
                      borderRadius: 0.75,
                      overflow: 'hidden',
                      cursor: 'pointer',
                      fontSize: 12,
                    })}
                    onClick={(e) => {
                      e.stopPropagation()
                      setPopoverId(showPopover ? null : a.id)
                    }}
                    onDoubleClick={(e) => {
                      e.stopPropagation()
                      setPopoverId(null)
                      startEdit(a)
                    }}
                  >
                    <Box sx={{ px: 0.75, py: 0.25, fontWeight: 500 }}>{a.start_time} – {a.end_time}</Box>
                    <Box sx={{ px: 0.75, pb: 0.5, color: 'text.secondary', fontSize: 11 }}>{a.patient_name || `Paciente ${a.patient_id.slice(0, 8)}…`}</Box>
                    <Box sx={{ px: 0.75, pb: 0.5 }}>
                      <Typography component="span" variant="caption" sx={{ px: 0.35, py: 0.1, borderRadius: 0.5, bgcolor: statusColor(a.status) }}>{a.status}</Typography>
                    </Box>
                    {showPopover && (
                      <Box sx={{ p: 0.5, borderTop: 1, borderColor: 'divider', display: 'flex', gap: 0.25, flexWrap: 'wrap' }} onClick={(e) => e.stopPropagation()}>
                        <Button size="small" variant="contained" onClick={() => startEdit(a)} sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 }, minWidth: 'auto', py: 0.25, px: 0.5 }}>Alterar</Button>
                        {a.status !== 'CANCELLED' && a.status !== 'COMPLETED' && (
                          <Button size="small" variant="outlined" color="error" onClick={() => handleCancelAppointment(a.id)} disabled={cancellingId === a.id} sx={{ minWidth: 'auto', py: 0.25, px: 0.5 }}>{cancellingId === a.id ? '...' : 'Cancelar'}</Button>
                        )}
                      </Box>
                    )}
                  </Paper>
                )
              })}
            </Box>
          ))}
        </Paper>
      )}

      <AppDialog open={!!editingId} onClose={cancelEdit} title="Editar compromisso" actions={
        <>
          <Button onClick={saveEdit} variant="contained" sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>Salvar</Button>
          <Button onClick={cancelEdit} color="inherit">Fechar</Button>
          {editStatus !== 'CANCELLED' && editStatus !== 'COMPLETED' && editingId && (
            <Button variant="outlined" color="error" onClick={async () => { await handleCancelAppointment(editingId); setEditingId(null) }} disabled={cancellingId === editingId}>
              {cancellingId === editingId ? 'Cancelando...' : 'Cancelar agendamento'}
            </Button>
          )}
        </>
      }>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mb: 2 }}>
          <TextField type="date" label="Data" size="small" fullWidth value={editDate} onChange={(e) => setEditDate(e.target.value)} InputLabelProps={{ shrink: true }} />
          <TextField type="time" label="Horário" size="small" fullWidth value={editTime} onChange={(e) => setEditTime(e.target.value)} InputLabelProps={{ shrink: true }} />
          <FormControl fullWidth size="small">
            <InputLabel>Status</InputLabel>
            <Select value={editStatus} label="Status" onChange={(e) => setEditStatus(e.target.value)}>
              <MenuItem value="PRE_AGENDADO">Pré-agendado</MenuItem>
              <MenuItem value="AGENDADO">Agendado</MenuItem>
              <MenuItem value="CONFIRMADO">Confirmado</MenuItem>
              <MenuItem value="CANCELLED">Cancelado</MenuItem>
              <MenuItem value="COMPLETED">Realizado</MenuItem>
              <MenuItem value="SERIES_ENDED">Série encerrada</MenuItem>
            </Select>
          </FormControl>
        </Box>
      </AppDialog>

      <AppDialog open={createModalOpen} onClose={() => setCreateModalOpen(false)} title="Criar agendamentos" actions={
        <>
          <Button onClick={() => setCreateModalOpen(false)} color="inherit">Fechar</Button>
          <Button variant="contained" onClick={handleCreateAppointments} disabled={creating || !selectedContractId || contractsForAgenda.length === 0} sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>
            {creating ? 'Criando...' : 'Criar agendamentos'}
          </Button>
        </>
      }>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>Vincule os novos compromissos a um contrato assinado.</Typography>
        <FormControl fullWidth size="small" sx={{ mb: 2 }}>
          <InputLabel>Contrato</InputLabel>
          <Select value={selectedContractId} label="Contrato" onChange={(e) => setSelectedContractId(e.target.value)}>
            {contractsForAgenda.length === 0 && <MenuItem value="">Nenhum contrato assinado</MenuItem>}
            {contractsForAgenda.map((c) => (
              <MenuItem key={c.id} value={c.id}>{c.patient_name} – {c.template_name}</MenuItem>
            ))}
          </Select>
        </FormControl>
        <Typography variant="subtitle2" sx={{ mb: 0.5 }}>Data e horário (apenas slots disponíveis na configuração da agenda)</Typography>
        {createModalUniqueDates.length === 0 && createModalOpen && (
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>Carregando slots...</Typography>
        )}
        {newSlots.map((slot, i) => {
          const timesForDate = getTimesForDate(slot.appointment_date)
          const timeValid = timesForDate.includes(slot.start_time)
          return (
            <Box key={i} sx={{ display: 'flex', gap: 0.5, alignItems: 'center', mb: 0.5 }}>
              <FormControl size="small" sx={{ minWidth: 140 }}>
                <InputLabel>Data</InputLabel>
                <Select
                  value={slot.appointment_date}
                  label="Data"
                  onChange={(e) => {
                    const date = e.target.value
                    const times = getTimesForDate(date)
                    setNewSlots((prev) => prev.map((s, j) => j === i ? { appointment_date: date, start_time: times[0] || s.start_time } : s))
                  }}
                  displayEmpty
                >
                  <MenuItem value="">Selecione</MenuItem>
                  {createModalUniqueDates.map((d) => (
                    <MenuItem key={d} value={d}>{formatDateBR(d)}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <FormControl size="small" sx={{ minWidth: 100 }}>
                <InputLabel>Horário</InputLabel>
                <Select
                  value={timeValid ? slot.start_time : (timesForDate[0] ?? '')}
                  label="Horário"
                  onChange={(e) => setNewSlots((prev) => prev.map((s, j) => j === i ? { ...s, start_time: e.target.value } : s))}
                  displayEmpty
                >
                  <MenuItem value="">-</MenuItem>
                  {timesForDate.map((t) => (
                    <MenuItem key={t} value={t}>{t}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <IconButton size="small" color="error" onClick={() => handleRemoveSlot(i)} aria-label="Remover"><DeleteOutlineIcon fontSize="small" /></IconButton>
            </Box>
          )
        })}
        <Button size="small" variant="outlined" onClick={handleAddSlot} sx={{ mb: 1 }}>+ Adicionar horário</Button>
        {createMessage && <Typography sx={{ color: createMessage.includes('Falha') ? 'error.main' : 'success.main', fontSize: 14 }}>{createMessage}</Typography>}
      </AppDialog>
    </PageContainer>
  )
}
