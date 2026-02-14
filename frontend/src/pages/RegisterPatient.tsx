import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Alert, Box, Button, Checkbox, FormControlLabel, Paper, TextField, Typography } from '@mui/material'
import * as api from '../lib/api'
import { isValidCPF } from '../lib/cpf'

/** Junta os 6 campos de endereço em uma única string (separador: newline). */
function buildAddress(rua: string, bairro: string, cidade: string, estado: string, pais: string, cep: string): string {
  return [rua, bairro, cidade, estado, pais, cep].join('\n')
}

export function RegisterPatient() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''

  const [invite, setInvite] = useState<{ email: string; full_name: string; clinic_name: string; expires_at: string } | null>(null)
  const [loading, setLoading] = useState(!!token)
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState('')

  const [samePerson, setSamePerson] = useState(false)
  const [guardianFullName, setGuardianFullName] = useState('')
  const [guardianCPF, setGuardianCPF] = useState('')
  const [guardianBirthDate, setGuardianBirthDate] = useState('')
  const [patientFullName, setPatientFullName] = useState('')
  const [patientBirthDate, setPatientBirthDate] = useState('')

  const [rua, setRua] = useState('')
  const [bairro, setBairro] = useState('')
  const [cidade, setCidade] = useState('')
  const [estado, setEstado] = useState('')
  const [pais, setPais] = useState('')
  const [cep, setCep] = useState('')

  useEffect(() => {
    if (!token) {
      setLoading(false)
      return
    }
    api.getPatientInviteByToken(token)
      .then((d) => {
        setInvite(d)
        setGuardianFullName(d.full_name || '')
      })
      .catch(() => setError('Link inválido ou expirado.'))
      .finally(() => setLoading(false))
  }, [token])

  const addressString = useMemo(() => buildAddress(rua.trim(), bairro.trim(), cidade.trim(), estado.trim(), pais.trim(), cep.replace(/\D/g, '')), [
    rua, bairro, cidade, estado, pais, cep,
  ])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!guardianFullName.trim()) {
      setError('Nome do responsável é obrigatório.')
      return
    }
    if (!guardianCPF.trim()) {
      setError('CPF do responsável é obrigatório.')
      return
    }
    if (!isValidCPF(guardianCPF)) {
      setError('CPF inválido.')
      return
    }
    if (!rua.trim() || !bairro.trim() || !cidade.trim() || !estado.trim() || !pais.trim() || !cep.trim()) {
      setError('Preencha o endereço completo: Rua, Bairro, Cidade, Estado, País e CEP.')
      return
    }
    const cepDigits = cep.replace(/\D/g, '')
    if (cepDigits.length !== 8) {
      setError('CEP deve ter 8 dígitos.')
      return
    }
    if (!guardianBirthDate.trim()) {
      setError('Data de nascimento do responsável é obrigatória.')
      return
    }
    if (!patientBirthDate.trim()) {
      setError('Data de nascimento do paciente é obrigatória.')
      return
    }

    setSubmitting(true)
    try {
      await api.acceptPatientInvite({
        token,
        same_person: samePerson,
        guardian_full_name: guardianFullName.trim(),
        guardian_cpf: guardianCPF.trim(),
        guardian_address: addressString,
        guardian_birth_date: guardianBirthDate.trim(),
        patient_full_name: samePerson ? '' : patientFullName.trim(),
        patient_birth_date: patientBirthDate.trim(),
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
      <Box sx={{ maxWidth: 520, mx: 'auto', p: 2, textAlign: 'center' }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Link inválido</Typography>
        <Typography>Use o link recebido por e-mail para acessar o formulário.</Typography>
        <Typography sx={{ mt: 2 }}>
          <Link to="/login" style={{ color: 'inherit' }}>Ir para o login</Link>
        </Typography>
      </Box>
    )
  }

  if (loading) {
    return (
      <Box sx={{ maxWidth: 520, mx: 'auto', p: 2 }}>
        <Typography color="text.secondary">Carregando...</Typography>
      </Box>
    )
  }

  if (error && !invite) {
    return (
      <Box sx={{ maxWidth: 520, mx: 'auto', p: 2, textAlign: 'center' }}>
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
      <Box sx={{ maxWidth: 520, mx: 'auto', p: 2, textAlign: 'center' }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Cadastro concluído</Typography>
        <Typography>Obrigado! O cadastro do paciente foi concluído.</Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ maxWidth: 560, mx: 'auto', p: 2 }}>
      <Paper variant="outlined" sx={{ p: 2 }}>
        <Typography variant="h5" sx={{ mb: 0.5 }}>Completar cadastro</Typography>
        <Typography color="text.secondary" sx={{ mb: 2, fontSize: 14 }}>
          Convite enviado por <b>{invite?.clinic_name}</b>. Preencha os dados abaixo.
        </Typography>

        <Box component="form" onSubmit={handleSubmit} sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {error && <Alert severity="error">{error}</Alert>}

          <TextField label="E-mail do responsável" value={invite?.email ?? ''} InputProps={{ readOnly: true }} fullWidth />

          <FormControlLabel
            control={<Checkbox checked={samePerson} onChange={(e) => setSamePerson(e.target.checked)} />}
            label="Paciente e responsável são a mesma pessoa"
          />

          <TextField label="Nome do responsável" value={guardianFullName} onChange={(e) => setGuardianFullName(e.target.value)} required fullWidth />
          <TextField
            label="CPF do responsável"
            value={guardianCPF}
            onChange={(e) => setGuardianCPF(e.target.value)}
            required
            fullWidth
            placeholder="000.000.000-00"
            inputProps={{ maxLength: 14 }}
            error={!!guardianCPF.trim() && !isValidCPF(guardianCPF)}
            helperText={guardianCPF.trim() && !isValidCPF(guardianCPF) ? 'CPF inválido.' : ' '}
          />

          <TextField label="Data de nascimento do responsável" type="date" value={guardianBirthDate} onChange={(e) => setGuardianBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} required fullWidth />

          <Typography variant="subtitle2" color="text.secondary" sx={{ mt: 0.5 }}>Endereço</Typography>
          <TextField label="Rua" value={rua} onChange={(e) => setRua(e.target.value)} required fullWidth />
          <TextField label="Bairro" value={bairro} onChange={(e) => setBairro(e.target.value)} required fullWidth />
          <TextField label="Cidade" value={cidade} onChange={(e) => setCidade(e.target.value)} required fullWidth />
          <TextField label="Estado" value={estado} onChange={(e) => setEstado(e.target.value)} required placeholder="UF" inputProps={{ maxLength: 2 }} fullWidth />
          <TextField label="País" value={pais} onChange={(e) => setPais(e.target.value)} required fullWidth />
          <TextField label="CEP" value={cep} onChange={(e) => setCep(e.target.value)} required placeholder="00000000" inputProps={{ maxLength: 9 }} fullWidth />

          {!samePerson && (
            <TextField label="Nome do paciente" value={patientFullName} onChange={(e) => setPatientFullName(e.target.value)} required fullWidth />
          )}
          <TextField label="Data de nascimento do paciente" type="date" value={patientBirthDate} onChange={(e) => setPatientBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} required fullWidth />

          <Button type="submit" variant="contained" disabled={submitting} sx={{ mt: 0.5 }}>
            {submitting ? 'Salvando...' : 'Concluir cadastro'}
          </Button>
        </Box>
      </Paper>
    </Box>
  )
}

