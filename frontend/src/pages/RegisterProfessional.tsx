import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Box, Typography, Button, TextField, Paper, FormControl, InputLabel, Select, MenuItem } from '@mui/material'
import * as api from '../lib/api'
import { isValidCPF } from '../lib/cpf'

const MARITAL_OPTIONS = [
  { value: '', label: 'Selecione' },
  { value: 'SOLTEIRO', label: 'Solteiro(a)' },
  { value: 'CASADO', label: 'Casado(a)' },
  { value: 'DIVORCIADO', label: 'Divorciado(a)' },
  { value: 'VIUVO', label: 'Viúvo(a)' },
  { value: 'OUTRO', label: 'Outro' },
]

export function RegisterProfessional() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const [invite, setInvite] = useState<{
    email: string
    full_name: string
    clinic_name: string
    expires_at: string
  } | null>(null)
  const [loading, setLoading] = useState(!!token)
  const [error, setError] = useState('')
  const [fullName, setFullName] = useState('')
  const [tradeName, setTradeName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [cpf, setCpf] = useState('')
  const [street, setStreet] = useState('')
  const [number, setNumber] = useState('')
  const [complement, setComplement] = useState('')
  const [neighborhood, setNeighborhood] = useState('')
  const [city, setCity] = useState('')
  const [state, setState] = useState('')
  const [country, setCountry] = useState('')
  const [zip, setZip] = useState('')
  const [maritalStatus, setMaritalStatus] = useState('')
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
      .getInviteByToken(token)
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
    if (!cpf.trim()) {
      setError('CPF é obrigatório.')
      return
    }
    if (!isValidCPF(cpf)) {
      setError('CPF inválido.')
      return
    }
    if (password.length < 8) {
      setError('A senha deve ter pelo menos 8 caracteres.')
      return
    }
    if (!street.trim() || !neighborhood.trim() || !city.trim() || !state.trim() || !country.trim() || !zip.trim()) {
      setError('Preencha o endereço: Rua, Bairro, Cidade, Estado, País e CEP.')
      return
    }
    const cepDigits = zip.replace(/\D/g, '')
    if (cepDigits.length !== 8) {
      setError('CEP deve ter 8 dígitos.')
      return
    }
    setSubmitting(true)
    try {
      await api.acceptInvite({
        token,
        password,
        full_name: fullName || undefined,
        trade_name: tradeName || undefined,
        birth_date: birthDate || undefined,
        cpf: cpf || undefined,
        address: {
          street: street.trim(),
          number: number.trim() || undefined,
          complement: complement.trim() || undefined,
          neighborhood: neighborhood.trim(),
          city: city.trim(),
          state: state.trim(),
          country: country.trim(),
          zip: cepDigits,
        },
        marital_status: maritalStatus || undefined,
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
        <Typography>Faça login na área do profissional com seu e-mail e senha.</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ fontWeight: 600, color: 'inherit' }}>Ir para o login</Link>
        </Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ maxWidth: 480, mx: 'auto', p: 2 }}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Concluir cadastro</Typography>
        <Typography color="text.secondary" sx={{ mb: 2, fontSize: 14 }}>Você foi convidado para {invite?.clinic_name}. Preencha os dados abaixo.</Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <TextField label="E-mail" type="email" fullWidth value={invite?.email ?? ''} InputProps={{ readOnly: true }} sx={{ mb: 1.5 }} />
          <TextField label="Nome completo" fullWidth required value={fullName} onChange={(e) => setFullName(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Nome fantasia" fullWidth value={tradeName} onChange={(e) => setTradeName(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField type="date" label="Data de nascimento" fullWidth value={birthDate} onChange={(e) => setBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} sx={{ mb: 1.5 }} />
          <TextField
            label="CPF"
            fullWidth
            required
            placeholder="000.000.000-00"
            inputProps={{ maxLength: 14 }}
            value={cpf}
            onChange={(e) => setCpf(e.target.value)}
            error={!!cpf.trim() && !isValidCPF(cpf)}
            helperText={cpf.trim() && !isValidCPF(cpf) ? 'CPF inválido.' : ' '}
            sx={{ mb: 1.5 }}
          />
          <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 0.5 }}>Endereço (obrigatório)</Typography>
          <TextField label="Rua" fullWidth required value={street} onChange={(e) => setStreet(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Número" fullWidth value={number} onChange={(e) => setNumber(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Complemento" fullWidth value={complement} onChange={(e) => setComplement(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Bairro" fullWidth required value={neighborhood} onChange={(e) => setNeighborhood(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Cidade" fullWidth required value={city} onChange={(e) => setCity(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="Estado (UF)" fullWidth required placeholder="UF" inputProps={{ maxLength: 2 }} value={state} onChange={(e) => setState(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="País" fullWidth required value={country} onChange={(e) => setCountry(e.target.value)} sx={{ mb: 1.5 }} />
          <TextField label="CEP" fullWidth required placeholder="00000000" inputProps={{ maxLength: 9 }} value={zip} onChange={(e) => setZip(e.target.value)} sx={{ mb: 1.5 }} />
          <FormControl fullWidth sx={{ mb: 1.5 }}>
            <InputLabel>Estado civil</InputLabel>
            <Select value={maritalStatus} label="Estado civil" onChange={(e) => setMaritalStatus(e.target.value)}>
              {MARITAL_OPTIONS.map((o) => (
                <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>
              ))}
            </Select>
          </FormControl>
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
