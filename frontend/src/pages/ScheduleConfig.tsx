import { useEffect, useState } from 'react'
import { Box, Typography, Button, TextField, Paper, FormControlLabel, Checkbox, FormControl, InputLabel, Select, MenuItem } from '@mui/material'
import { useAuth } from '../contexts/AuthContext'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'

const DAY_NAMES = ['Domingo', 'Segunda', 'Terça', 'Quarta', 'Quinta', 'Sexta', 'Sábado']

export function ScheduleConfig() {
  const { user } = useAuth()
  const [days, setDays] = useState<api.ScheduleDay[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [copyFrom, setCopyFrom] = useState<number>(1)
  const [copyTo, setCopyTo] = useState<number>(2)

  useEffect(() => {
    api.getScheduleConfig()
      .then((r) => {
        const list = r.days || []
        if (list.length === 7) {
          setDays(list.map((d) => ({ ...d, enabled: !!d.enabled })))
        } else {
          setDays(
            Array.from({ length: 7 }, (_, i) => ({
              day_of_week: i,
              enabled: false,
              start_time: null as string | null,
              end_time: null as string | null,
              consultation_duration_minutes: 50,
              interval_minutes: 10,
              lunch_start: null as string | null,
              lunch_end: null as string | null,
            }))
          )
        }
      })
      .catch(() => {
        setDays(
          Array.from({ length: 7 }, (_, i) => ({
            day_of_week: i,
            enabled: false,
            start_time: null as string | null,
            end_time: null as string | null,
            consultation_duration_minutes: 50,
            interval_minutes: 10,
            lunch_start: null as string | null,
            lunch_end: null as string | null,
          }))
        )
      })
      .finally(() => setLoading(false))
  }, [])

  const handleSave = async () => {
    setSaving(true)
    setMessage('')
    const payload = days.map((d) => ({
      day_of_week: d.day_of_week,
      enabled: d.enabled,
      start_time: d.start_time || null,
      end_time: d.end_time || null,
      consultation_duration_minutes: d.consultation_duration_minutes,
      interval_minutes: d.interval_minutes,
      lunch_start: d.lunch_start || null,
      lunch_end: d.lunch_end || null,
    }))
    try {
      await api.putScheduleConfig(payload)
      setMessage('Configuração salva.')
    } catch {
      setMessage('Falha ao salvar.')
    } finally {
      setSaving(false)
    }
  }

  const handleCopy = async () => {
    if (copyFrom === copyTo) return
    setMessage('')
    try {
      await api.copyScheduleConfigDay(copyFrom, copyTo)
      setMessage(`Configuração do ${DAY_NAMES[copyFrom]} copiada para ${DAY_NAMES[copyTo]}.`)
      setDays((prev) => {
        const source = prev.find((d) => d.day_of_week === copyFrom)
        if (!source) return prev
        return prev.map((d) =>
          d.day_of_week === copyTo ? { ...source, day_of_week: copyTo, enabled: true } : d
        )
      })
    } catch {
      setMessage('Falha ao copiar.')
    }
  }

  const toggleDayEnabled = (dayOfWeek: number) => {
    setDays((prev) => prev.map((d) => (d.day_of_week === dayOfWeek ? { ...d, enabled: !d.enabled } : d)))
  }

  const updateDay = (dayOfWeek: number, field: string, value: string | number | boolean) => {
    setDays((prev) => prev.map((d) => (d.day_of_week === dayOfWeek ? { ...d, [field]: value } : d)))
  }

  if (user?.role !== 'PROFESSIONAL' && user?.role !== 'SUPER_ADMIN') {
    return (
      <PageContainer>
        <Typography>Apenas profissionais podem configurar a agenda.</Typography>
      </PageContainer>
    )
  }

  if (loading) return (
    <PageContainer>
      <Typography color="text.secondary">Carregando...</Typography>
    </PageContainer>
  )

  const enabledDays = days.filter((d) => d.enabled)

  return (
    <PageContainer>
      <Typography variant="h4" sx={{ mb: 2 }}>Configurar agenda</Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        Marque os dias da semana em que você atende. Só os dias marcados aparecem para configurar horários e na agenda.
      </Typography>

      <Box sx={{ mb: 2 }}>
        <Typography fontWeight={600} sx={{ mb: 0.5 }}>Dias de atendimento</Typography>
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
          {days.map((d) => (
            <FormControlLabel key={d.day_of_week} control={<Checkbox checked={d.enabled} onChange={() => toggleDayEnabled(d.day_of_week)} />} label={DAY_NAMES[d.day_of_week]} />
          ))}
        </Box>
      </Box>

      {enabledDays.length > 0 && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2, bgcolor: 'grey.50' }}>
          <Typography fontWeight={600}>Copiar um dia para outro</Typography>
          <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', flexWrap: 'wrap', mt: 0.5 }}>
            <FormControl size="small" sx={{ minWidth: 140 }}>
              <InputLabel>De</InputLabel>
              <Select value={copyFrom} label="De" onChange={(e) => setCopyFrom(Number(e.target.value))}>
                {DAY_NAMES.map((_, i) => (
                  <MenuItem key={i} value={i}>{DAY_NAMES[i]}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <Typography>→</Typography>
            <FormControl size="small" sx={{ minWidth: 140 }}>
              <InputLabel>Para</InputLabel>
              <Select value={copyTo} label="Para" onChange={(e) => setCopyTo(Number(e.target.value))}>
                {DAY_NAMES.map((_, i) => (
                  <MenuItem key={i} value={i}>{DAY_NAMES[i]}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <Button variant="contained" size="small" onClick={handleCopy}>Copiar</Button>
          </Box>
        </Paper>
      )}

      {message && <Typography sx={{ mb: 2, color: message.includes('Falha') ? 'error.main' : 'success.main' }}>{message}</Typography>}

      {enabledDays.length === 0 ? (
        <Typography color="text.secondary">Marque pelo menos um dia acima para configurar horários.</Typography>
      ) : (
        <>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {days.filter((d) => d.enabled).map((d) => (
              <Paper key={d.day_of_week} variant="outlined" sx={{ p: 2 }}>
                <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>{DAY_NAMES[d.day_of_week]}</Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 1 }}>
                  <TextField type="time" label="Início" size="small" value={d.start_time || ''} onChange={(e) => updateDay(d.day_of_week, 'start_time', e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
                  <TextField type="time" label="Fim" size="small" value={d.end_time || ''} onChange={(e) => updateDay(d.day_of_week, 'end_time', e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
                  <TextField type="number" label="Duração consulta (min)" size="small" inputProps={{ min: 5, max: 120 }} value={d.consultation_duration_minutes} onChange={(e) => updateDay(d.day_of_week, 'consultation_duration_minutes', parseInt(e.target.value, 10) || 50)} fullWidth />
                  <TextField type="number" label="Intervalo (min)" size="small" inputProps={{ min: 0, max: 60 }} value={d.interval_minutes} onChange={(e) => updateDay(d.day_of_week, 'interval_minutes', parseInt(e.target.value, 10) || 10)} fullWidth />
                  <TextField type="time" label="Almoço início" size="small" value={d.lunch_start || ''} onChange={(e) => updateDay(d.day_of_week, 'lunch_start', e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
                  <TextField type="time" label="Almoço fim" size="small" value={d.lunch_end || ''} onChange={(e) => updateDay(d.day_of_week, 'lunch_end', e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
                </Box>
              </Paper>
            ))}
          </Box>
          <Button variant="contained" onClick={handleSave} disabled={saving} sx={{ mt: 2 }}>
            {saving ? 'Salvando...' : 'Salvar configuração'}
          </Button>
        </>
      )}
    </PageContainer>
  )
}
