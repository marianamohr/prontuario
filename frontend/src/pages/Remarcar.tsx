import { useEffect, useState } from 'react'
import { useSearchParams, Link } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  Alert,
  Paper,
  CircularProgress,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material'
import * as api from '../lib/api'

function formatDateBR(s: string) {
  const d = new Date(s + 'T12:00:00')
  return d.toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: '2-digit' })
}

function formatTime(s: string) {
  const [h, m] = s.split(':').map(Number)
  return `${String(h).padStart(2, '0')}:${String(m ?? 0).padStart(2, '0')}`
}

export function Remarcar() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const [loading, setLoading] = useState(!!token)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [data, setData] = useState<{
    appointment_id: string
    patient_name: string
    current_date: string
    current_start_time: string
    slots: { date: string; start_time: string }[]
  } | null>(null)
  const [selectedDate, setSelectedDate] = useState('')
  const [selectedTime, setSelectedTime] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [remarcando, setRemarcando] = useState(false)

  useEffect(() => {
    if (!token) {
      setLoading(false)
      setError('Link inválido.')
      return
    }
    api
      .getRemarcarByToken(token)
      .then((r) => {
        setData(r)
        const dates = [...new Set((r.slots || []).map((s) => s.date))].sort()
        if (dates.length) setSelectedDate(dates[0])
      })
      .catch(() => setError('Link inválido ou expirado.'))
      .finally(() => setLoading(false))
  }, [token])

  const slotsForDate = (data?.slots ?? []).filter((s) => s.date === selectedDate)
  const timesForDate = [...new Set(slotsForDate.map((s) => s.start_time))].sort()

  const handleConfirmar = async () => {
    if (!token) return
    setConfirming(true)
    setError('')
    setSuccess('')
    try {
      await api.confirmRemarcar(token)
      setSuccess('Presença confirmada. Até lá!')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro ao confirmar.')
    } finally {
      setConfirming(false)
    }
  }

  const handleRemarcar = async () => {
    if (!token || !selectedDate || !selectedTime) return
    setRemarcando(true)
    setError('')
    setSuccess('')
    try {
      await api.remarcarAppointment(token, selectedDate, selectedTime)
      setSuccess('Consulta remarcada com sucesso!')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro ao remarcar.')
    } finally {
      setRemarcando(false)
    }
  }

  if (!token) {
    return (
      <Box sx={{ maxWidth: 480, mx: 'auto', p: 2, textAlign: 'center' }}>
        <Typography variant="h5" sx={{ mb: 1 }}>Link inválido</Typography>
        <Typography color="text.secondary">Use o link recebido por WhatsApp.</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Ir para o login</Link>
        </Typography>
      </Box>
    )
  }

  if (loading || !data) {
    return (
      <Box sx={{ maxWidth: 480, mx: 'auto', p: 2, textAlign: 'center' }}>
        {loading && <CircularProgress sx={{ my: 3 }} />}
        {error && <Alert severity="error" sx={{ mt: 2 }}>{error}</Alert>}
      </Box>
    )
  }

  return (
    <Box sx={{ maxWidth: 520, mx: 'auto', p: 2 }}>
      <Typography variant="h5" sx={{ mb: 0.5 }}>Sua consulta</Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        Paciente: <b>{data.patient_name}</b>. Data atual: {formatDateBR(data.current_date)} às {formatTime(data.current_start_time)}.
      </Typography>

      {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
        <Typography variant="subtitle1" sx={{ mb: 1 }}>Confirmar presença</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          Confirme que comparecerá na data e horário atuais.
        </Typography>
        <Button variant="contained" onClick={handleConfirmar} disabled={confirming}>
          {confirming ? 'Confirmando...' : 'Confirmar presença'}
        </Button>
      </Paper>

      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="subtitle1" sx={{ mb: 1 }}>Remarcar consulta</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Escolha uma nova data e horário disponível.
        </Typography>
        {(data.slots || []).length === 0 && (
          <Alert severity="info" sx={{ mb: 2 }}>
            Nenhum horário disponível nas próximas 2 semanas. Entre em contato com a clínica.
          </Alert>
        )}
        <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
          <FormControl size="small" sx={{ minWidth: 180, flex: 1 }}>
            <InputLabel>Data</InputLabel>
            <Select
              value={selectedDate || ''}
              label="Data"
              onChange={(e) => {
                setSelectedDate(e.target.value)
                setSelectedTime('')
              }}
            >
              {[...new Set((data.slots || []).map((s) => s.date))].sort().map((d) => (
                <MenuItem key={d} value={d}>{formatDateBR(d)}</MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 120, flex: 1 }}>
            <InputLabel>Horário</InputLabel>
            <Select
              value={selectedTime || ''}
              label="Horário"
              onChange={(e) => setSelectedTime(e.target.value)}
            >
              {timesForDate.map((t) => (
                <MenuItem key={t} value={t}>{formatTime(t)}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </Box>
        <Button
          variant="outlined"
          onClick={handleRemarcar}
          disabled={remarcando || !selectedDate || !selectedTime}
          sx={{ mt: 2 }}
        >
          {remarcando ? 'Remarcando...' : 'Remarcar'}
        </Button>
      </Paper>

      <Typography sx={{ mt: 2, textAlign: 'center' }}>
        <Link to="/login" style={{ color: 'inherit', fontSize: 14 }}>Ir para o login</Link>
      </Typography>
    </Box>
  )
}
