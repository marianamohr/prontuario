import { useCallback, useEffect, useState } from 'react'
import { Alert, Box, Button, FormControl, InputLabel, MenuItem, Paper, Select, TextField, Typography } from '@mui/material'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'

const MARITAL_OPTIONS = [
  { value: '', label: 'Selecione' },
  { value: 'SOLTEIRO', label: 'Solteiro(a)' },
  { value: 'CASADO', label: 'Casado(a)' },
  { value: 'DIVORCIADO', label: 'Divorciado(a)' },
  { value: 'VIUVO', label: 'Viúvo(a)' },
  { value: 'OUTRO', label: 'Outro' },
]

export function Profile() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const [email, setEmail] = useState('')
  const [fullName, setFullName] = useState('')
  const [tradeName, setTradeName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [address, setAddress] = useState('')
  const [maritalStatus, setMaritalStatus] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    setError('')
    setSuccess('')
    api.getMyProfile()
      .then((p) => {
        setEmail(p.email ?? '')
        setFullName(p.full_name ?? '')
        setTradeName((p.trade_name ?? '') as string)
        setBirthDate((p.birth_date ?? '') as string)
        setAddress((p.address ?? '') as string)
        setMaritalStatus((p.marital_status ?? '') as string)
      })
      .catch(() => setError('Falha ao carregar seu perfil.'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    if (!fullName.trim()) {
      setError('Nome completo é obrigatório.')
      return
    }
    setSaving(true)
    try {
      await api.patchMyProfile({
        full_name: fullName.trim(),
        trade_name: tradeName.trim() || undefined,
        birth_date: birthDate.trim() || undefined,
        address: address || undefined,
        marital_status: maritalStatus || undefined,
      })
      setSuccess('Perfil atualizado.')
      load()
    } catch {
      setError('Falha ao atualizar perfil.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <PageContainer>
      <Typography variant="h4" sx={{ mb: 2 }}>Editar perfil</Typography>
      <Paper variant="outlined" sx={{ p: 2, maxWidth: 560 }}>
        {loading ? (
          <Typography color="text.secondary">Carregando...</Typography>
        ) : (
          <Box component="form" onSubmit={handleSave} sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            {error && <Alert severity="error">{error}</Alert>}
            {success && <Alert severity="success">{success}</Alert>}

            <TextField label="E-mail" value={email} InputProps={{ readOnly: true }} fullWidth />
            <TextField label="Nome completo" value={fullName} onChange={(e) => setFullName(e.target.value)} required fullWidth />
            <TextField label="Nome fantasia" value={tradeName} onChange={(e) => setTradeName(e.target.value)} fullWidth />
            <TextField label="Data de nascimento" type="date" value={birthDate} onChange={(e) => setBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
            <TextField label="Endereço" value={address} onChange={(e) => setAddress(e.target.value)} multiline minRows={2} fullWidth />
            <FormControl fullWidth>
              <InputLabel>Estado civil</InputLabel>
              <Select value={maritalStatus} label="Estado civil" onChange={(e) => setMaritalStatus(String(e.target.value))}>
                {MARITAL_OPTIONS.map((o) => (
                  <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>
                ))}
              </Select>
            </FormControl>

            <Button type="submit" variant="contained" disabled={saving} sx={{ alignSelf: 'flex-start' }}>
              {saving ? 'Salvando...' : 'Salvar'}
            </Button>
            <Typography variant="body2" color="text.secondary">
              CPF e e-mail não podem ser alterados por aqui.
            </Typography>
          </Box>
        )}
      </Paper>
    </PageContainer>
  )
}

