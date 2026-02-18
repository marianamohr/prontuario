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
  const [street, setStreet] = useState('')
  const [number, setNumber] = useState('')
  const [complement, setComplement] = useState('')
  const [neighborhood, setNeighborhood] = useState('')
  const [city, setCity] = useState('')
  const [state, setState] = useState('')
  const [country, setCountry] = useState('')
  const [zip, setZip] = useState('')
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
        const addr = p.address
        if (addr && typeof addr === 'object') {
          setStreet(addr.street ?? '')
          setNumber(addr.number ?? '')
          setComplement(addr.complement ?? '')
          setNeighborhood(addr.neighborhood ?? '')
          setCity(addr.city ?? '')
          setState(addr.state ?? '')
          setCountry(addr.country ?? '')
          setZip(addr.zip ?? '')
        } else {
          setStreet('')
          setNumber('')
          setComplement('')
          setNeighborhood('')
          setCity('')
          setState('')
          setCountry('')
          setZip('')
        }
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
      const addressPayload =
        street.trim() || neighborhood.trim() || city.trim() || state.trim() || country.trim() || zip.trim()
          ? {
              street: street.trim(),
              number: number.trim() || undefined,
              complement: complement.trim() || undefined,
              neighborhood: neighborhood.trim(),
              city: city.trim(),
              state: state.trim(),
              country: country.trim(),
              zip: zip.replace(/\D/g, ''),
            }
          : undefined
      if (addressPayload && addressPayload.zip.length !== 8) {
        setError('CEP deve ter 8 dígitos.')
        setSaving(false)
        return
      }
      await api.patchMyProfile({
        full_name: fullName.trim(),
        trade_name: tradeName.trim() || undefined,
        birth_date: birthDate.trim() || undefined,
        address: addressPayload,
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
            <Typography variant="subtitle2" color="text.secondary">Endereço</Typography>
            <TextField label="Rua" value={street} onChange={(e) => setStreet(e.target.value)} fullWidth />
            <TextField label="Número" value={number} onChange={(e) => setNumber(e.target.value)} fullWidth />
            <TextField label="Complemento" value={complement} onChange={(e) => setComplement(e.target.value)} fullWidth />
            <TextField label="Bairro" value={neighborhood} onChange={(e) => setNeighborhood(e.target.value)} fullWidth />
            <TextField label="Cidade" value={city} onChange={(e) => setCity(e.target.value)} fullWidth />
            <TextField label="Estado (UF)" value={state} onChange={(e) => setState(e.target.value)} placeholder="UF" inputProps={{ maxLength: 2 }} fullWidth />
            <TextField label="País" value={country} onChange={(e) => setCountry(e.target.value)} fullWidth />
            <TextField label="CEP" value={zip} onChange={(e) => setZip(e.target.value)} placeholder="00000000" inputProps={{ maxLength: 9 }} fullWidth />
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

