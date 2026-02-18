import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  IconButton,
  TextField,
  InputAdornment,
  FormControlLabel,
  Checkbox,
  Alert,
  CircularProgress,
} from '@mui/material'
import EditIcon from '@mui/icons-material/Edit'
import AssignmentIcon from '@mui/icons-material/Assignment'
import DescriptionIcon from '@mui/icons-material/Description'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import VisibilityIcon from '@mui/icons-material/Visibility'
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'
import { PageContainer } from '../components/ui/PageContainer'
import { AppDialog } from '../components/ui/AppDialog'
import * as api from '../lib/api'
import { isValidCPF, normalizeCPF } from '../lib/cpf'

const DEFAULT_ACTION = '#16a34a'
const DEFAULT_NEGATION = '#dc2626'

/** Valida formato de e-mail (uma @ e domínio com ponto). */
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** Monta objeto Address (8 campos) para a API. */
function buildAddressObject(
  street: string,
  number: string,
  complement: string,
  neighborhood: string,
  city: string,
  state: string,
  country: string,
  zip: string
): api.Address {
  return {
    street: street.trim(),
    number: number.trim() || undefined,
    complement: complement.trim() || undefined,
    neighborhood: neighborhood.trim(),
    city: city.trim(),
    state: state.trim(),
    country: country.trim(),
    zip: zip.replace(/\D/g, ''),
  }
}

/** Extrai 8 campos do endereço (objeto da API ou string legacy). */
function parseAddress(addr: api.Address | string | undefined): [string, string, string, string, string, string, string, string] {
  if (!addr) return ['', '', '', '', '', '', '', '']
  if (typeof addr === 'object') {
    return [
      addr.street ?? '',
      addr.number ?? '',
      addr.complement ?? '',
      addr.neighborhood ?? '',
      addr.city ?? '',
      addr.state ?? '',
      addr.country ?? '',
      addr.zip ?? '',
    ]
  }
  const parts = String(addr).split('\n')
  return [
    parts[0]?.trim() ?? '',
    parts[1]?.trim() ?? '',
    parts[2]?.trim() ?? '',
    parts[3]?.trim() ?? '',
    parts[4]?.trim() ?? '',
    parts[5]?.trim() ?? '',
    parts[6]?.trim() ?? '',
    parts[7]?.trim() ?? '',
  ]
}

function formatBirthDate(s: string | undefined) {
  if (!s) return '—'
  const d = new Date(s + 'T12:00:00')
  if (Number.isNaN(d.getTime())) return s
  return d.toLocaleDateString('pt-BR')
}

export function Patients() {
  const { user, isImpersonated } = useAuth()
  const branding = useBranding()?.branding ?? null
  const actionColor = user?.role === 'PROFESSIONAL' && branding?.action_button_color ? branding.action_button_color : DEFAULT_ACTION
  const negationColor = user?.role === 'PROFESSIONAL' && branding?.negation_button_color ? branding.negation_button_color : DEFAULT_NEGATION
  const [list, setList] = useState<{ id: string; full_name: string; birth_date?: string }[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [deletingPatientId, setDeletingPatientId] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [inviteModalOpen, setInviteModalOpen] = useState(false)
  const [inviteGuardianFullName, setInviteGuardianFullName] = useState('')
  const [inviteGuardianEmail, setInviteGuardianEmail] = useState('')
  const [inviteSubmitting, setInviteSubmitting] = useState(false)
  const [inviteSuccess, setInviteSuccess] = useState('')
  const [samePerson, setSamePerson] = useState(false)
  const [guardianFullName, setGuardianFullName] = useState('')
  const [guardianEmail, setGuardianEmail] = useState('')
  const [guardianCpf, setGuardianCpf] = useState('')
  const [patientCpf, setPatientCpf] = useState('')
  const [showPatientCPF, setShowPatientCPF] = useState(false)
  const [guardianRua, setGuardianRua] = useState('')
  const [guardianNumero, setGuardianNumero] = useState('')
  const [guardianComplemento, setGuardianComplemento] = useState('')
  const [guardianBairro, setGuardianBairro] = useState('')
  const [guardianCidade, setGuardianCidade] = useState('')
  const [guardianEstado, setGuardianEstado] = useState('')
  const [guardianPais, setGuardianPais] = useState('')
  const [guardianCep, setGuardianCep] = useState('')
  const [guardianBirthDate, setGuardianBirthDate] = useState('')
  const [guardianPhone, setGuardianPhone] = useState('')
  const [patientFullName, setPatientFullName] = useState('')
  const [newBirthDate, setNewBirthDate] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState('')
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editingPatientId, setEditingPatientId] = useState<string | null>(null)
  const [editFullName, setEditFullName] = useState('')
  const [editBirthDate, setEditBirthDate] = useState('')
  const [editEmail, setEditEmail] = useState('')
  const [editPatientCpf, setEditPatientCpf] = useState('')
  const [editPatientCpfInitial, setEditPatientCpfInitial] = useState('')
  const [showEditPatientCPF, setShowEditPatientCPF] = useState(false)
  const [editGuardianFullName, setEditGuardianFullName] = useState('')
  const [editGuardianEmail, setEditGuardianEmail] = useState('')
  const [editGuardianRua, setEditGuardianRua] = useState('')
  const [editGuardianNumero, setEditGuardianNumero] = useState('')
  const [editGuardianComplemento, setEditGuardianComplemento] = useState('')
  const [editGuardianBairro, setEditGuardianBairro] = useState('')
  const [editGuardianCidade, setEditGuardianCidade] = useState('')
  const [editGuardianEstado, setEditGuardianEstado] = useState('')
  const [editGuardianPais, setEditGuardianPais] = useState('')
  const [editGuardianCep, setEditGuardianCep] = useState('')
  const [editGuardianBirthDate, setEditGuardianBirthDate] = useState('')
  const [editGuardianPhone, setEditGuardianPhone] = useState('')
  const [editGuardianCpf, setEditGuardianCpf] = useState('')
  const [editGuardianCpfInitial, setEditGuardianCpfInitial] = useState('')
  const [editSubmitting, setEditSubmitting] = useState(false)
  const [editError, setEditError] = useState('')
  const [editHasGuardian, setEditHasGuardian] = useState(false)
  const [editGuardianId, setEditGuardianId] = useState<string | null>(null)
  const [showEditGuardianCPF, setShowEditGuardianCPF] = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    api
      .listPatients()
      .then((r) => setList(r.patients))
      .catch(() => setError('Falha ao carregar pacientes.'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const resetForm = () => {
    setSamePerson(false)
    setGuardianFullName('')
    setGuardianEmail('')
    setGuardianCpf('')
    setPatientCpf('')
    setShowPatientCPF(false)
    setGuardianRua('')
    setGuardianNumero('')
    setGuardianComplemento('')
    setGuardianBairro('')
    setGuardianCidade('')
    setGuardianEstado('')
    setGuardianPais('')
    setGuardianCep('')
    setGuardianBirthDate('')
    setGuardianPhone('')
    setPatientFullName('')
    setNewBirthDate('')
    setFormError('')
  }

  const resetInviteForm = () => {
    setInviteGuardianFullName('')
    setInviteGuardianEmail('')
    setInviteSubmitting(false)
    setInviteSuccess('')
    setFormError('')
  }

  const handleCreatePatient = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    const hasGuardian = guardianEmail.trim() !== ''
    if (hasGuardian) {
      if (!guardianFullName.trim()) {
        setFormError('Nome do responsável é obrigatório.')
        return
      }
      if (!EMAIL_REGEX.test(guardianEmail.trim())) {
        setFormError('E-mail do responsável inválido.')
        return
      }
      if (!guardianCpf.trim()) {
        setFormError('CPF do responsável é obrigatório.')
        return
      }
      if (!isValidCPF(guardianCpf)) {
        setFormError('CPF do responsável inválido.')
        return
      }
      if (!guardianRua.trim() || !guardianBairro.trim() || !guardianCidade.trim() || !guardianEstado.trim() || !guardianPais.trim() || !guardianCep.trim()) {
        setFormError('Preencha todos os campos do endereço: Rua, Bairro, Cidade, Estado, País e CEP.')
        return
      }
      const cepDigits = guardianCep.replace(/\D/g, '')
      if (cepDigits.length !== 8) {
        setFormError('CEP deve ter 8 dígitos.')
        return
      }
      if (!guardianBirthDate.trim()) {
        setFormError('Data de nascimento do responsável é obrigatória.')
        return
      }
      if (!newBirthDate.trim()) {
        setFormError('Data de nascimento do paciente é obrigatória.')
        return
      }
    } else {
      if (!guardianFullName.trim() && !patientFullName.trim()) {
        setFormError('Informe o nome do responsável ou do paciente.')
        return
      }
    }
    if (patientCpf.trim() && !isValidCPF(patientCpf)) {
      setFormError('CPF do paciente inválido.')
      return
    }
    setSubmitting(true)
    try {
      const birth = newBirthDate.trim() || undefined
      if (hasGuardian) {
        await api.createPatient({
          full_name: guardianFullName.trim(),
          birth_date: birth,
          patient_cpf: patientCpf.trim() || undefined,
          same_person: samePerson,
          guardian_full_name: guardianFullName.trim(),
          guardian_email: guardianEmail.trim(),
          guardian_cpf: guardianCpf.trim(),
          guardian_address: buildAddressObject(guardianRua, guardianNumero, guardianComplemento, guardianBairro, guardianCidade, guardianEstado, guardianPais, guardianCep),
          guardian_birth_date: guardianBirthDate.trim(),
          guardian_phone: guardianPhone.trim() || undefined,
          patient_full_name: samePerson ? '' : patientFullName.trim(),
        })
      } else {
        const name = (samePerson ? guardianFullName : patientFullName) || guardianFullName || patientFullName
        await api.createPatient({ full_name: name.trim(), birth_date: birth, patient_cpf: patientCpf.trim() || undefined })
      }
      resetForm()
      setModalOpen(false)
      load()
    } catch {
      setFormError('Falha ao cadastrar paciente.')
    } finally {
      setSubmitting(false)
    }
  }

  const handleOpenEdit = (p: { id: string }) => {
    setEditingPatientId(p.id)
    setEditError('')
    setEditModalOpen(true)
    setEditFullName('')
    setEditBirthDate('')
    setEditEmail('')
    setEditPatientCpf('')
    setEditPatientCpfInitial('')
    setShowEditPatientCPF(false)
    setEditGuardianFullName('')
    setEditGuardianEmail('')
    setEditGuardianRua('')
    setEditGuardianNumero('')
    setEditGuardianComplemento('')
    setEditGuardianBairro('')
    setEditGuardianCidade('')
    setEditGuardianEstado('')
    setEditGuardianPais('')
    setEditGuardianCep('')
    setEditGuardianBirthDate('')
    setEditGuardianPhone('')
    setEditGuardianCpf('')
    setEditGuardianCpfInitial('')
    setEditHasGuardian(false)
    setEditGuardianId(null)
    setShowEditGuardianCPF(false)
    api.getPatient(p.id).then((data) => {
      setEditFullName(data.full_name ?? '')
      setEditBirthDate(data.birth_date ?? '')
      setEditEmail(data.email ?? '')
      const cpfPaciente = data.cpf ? String(data.cpf) : ''
      setEditPatientCpf(cpfPaciente)
      setEditPatientCpfInitial(normalizeCPF(cpfPaciente))
      if (data.guardian) {
        setEditHasGuardian(true)
        setEditGuardianId(data.guardian.id ?? null)
        setEditGuardianFullName(data.guardian.full_name ?? '')
        setEditGuardianEmail(data.guardian.email ?? '')
        const [rua, numero, complemento, bairro, cidade, estado, pais, cep] = parseAddress(data.guardian.address ?? undefined)
        setEditGuardianRua(rua)
        setEditGuardianNumero(numero)
        setEditGuardianComplemento(complemento)
        setEditGuardianBairro(bairro)
        setEditGuardianCidade(cidade)
        setEditGuardianEstado(estado)
        setEditGuardianPais(pais)
        setEditGuardianCep(cep)
        setEditGuardianBirthDate(data.guardian.birth_date ?? '')
        setEditGuardianPhone(data.guardian.phone ?? '')
        const cpf = data.guardian.cpf ? String(data.guardian.cpf) : ''
        setEditGuardianCpf(cpf)
        setEditGuardianCpfInitial(normalizeCPF(cpf))
      }
    }).catch(() => setEditError('Falha ao carregar paciente.'))
  }

  const handleCloseEdit = () => {
    setEditModalOpen(false)
    setEditingPatientId(null)
    setEditError('')
  }

  const handleSaveEdit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingPatientId) return
    setEditError('')
    if (!editFullName.trim()) {
      setEditError('Nome do paciente é obrigatório.')
      return
    }
    setEditSubmitting(true)
    try {
      const payload: api.UpdatePatientPayload = {
        full_name: editFullName.trim(),
        birth_date: editBirthDate.trim() || undefined,
        email: editEmail.trim() || undefined,
      }
      if (normalizeCPF(editPatientCpf) !== editPatientCpfInitial) {
        // Enviar string vazia permite limpar CPF no backend.
        payload.patient_cpf = editPatientCpf.trim()
        if (payload.patient_cpf && !isValidCPF(payload.patient_cpf)) {
          setEditError('CPF do paciente inválido.')
          setEditSubmitting(false)
          return
        }
      }
      if (editHasGuardian) {
        if (editGuardianEmail.trim() && !EMAIL_REGEX.test(editGuardianEmail.trim())) {
          setEditError('E-mail do responsável inválido.')
          setEditSubmitting(false)
          return
        }
        const hasAnyAddress = editGuardianRua.trim() || editGuardianBairro.trim() || editGuardianCidade.trim() || editGuardianEstado.trim() || editGuardianPais.trim() || editGuardianCep.trim()
        if (hasAnyAddress) {
          if (!editGuardianRua.trim() || !editGuardianBairro.trim() || !editGuardianCidade.trim() || !editGuardianEstado.trim() || !editGuardianPais.trim() || !editGuardianCep.trim()) {
            setEditError('Preencha todos os campos do endereço: Rua, Bairro, Cidade, Estado, País e CEP.')
            setEditSubmitting(false)
            return
          }
          const cepDigits = editGuardianCep.replace(/\D/g, '')
          if (cepDigits.length !== 8) {
            setEditError('CEP deve ter 8 dígitos.')
            setEditSubmitting(false)
            return
          }
        }
        payload.guardian_full_name = editGuardianFullName.trim() || undefined
        payload.guardian_email = editGuardianEmail.trim() || undefined
        payload.guardian_address = hasAnyAddress ? buildAddressObject(editGuardianRua, editGuardianNumero, editGuardianComplemento, editGuardianBairro, editGuardianCidade, editGuardianEstado, editGuardianPais, editGuardianCep) : undefined
        payload.guardian_birth_date = editGuardianBirthDate.trim() || undefined
        payload.guardian_phone = editGuardianPhone.trim() || undefined
        if (editGuardianCpf.trim()) {
          if (!isValidCPF(editGuardianCpf)) {
            setEditError('CPF do responsável inválido.')
            setEditSubmitting(false)
            return
          }
          const curCpf = normalizeCPF(editGuardianCpf)
          if (curCpf && curCpf !== editGuardianCpfInitial) {
            payload.guardian_cpf = editGuardianCpf.trim()
          }
        }
      }
      await api.updatePatient(editingPatientId, payload)
      handleCloseEdit()
      load()
    } catch {
      setEditError('Falha ao atualizar paciente.')
    } finally {
      setEditSubmitting(false)
    }
  }

  const handleSoftDeletePatient = async (patientId: string) => {
    const ok = window.confirm('Tem certeza que deseja excluir (soft delete) este paciente? Ele vai sumir do front para profissionais.')
    if (!ok) return
    setDeletingPatientId(patientId)
    try {
      await api.softDeletePatient(patientId)
      load()
    } catch {
      setError('Falha ao excluir paciente.')
    } finally {
      setDeletingPatientId(null)
    }
  }

  const handleSoftDeleteGuardian = async () => {
    if (!editingPatientId || !editGuardianId) return
    const ok = window.confirm('Tem certeza que deseja excluir (soft delete) este responsável? Ele vai sumir do front para profissionais.')
    if (!ok) return
    try {
      await api.softDeleteGuardian(editingPatientId, editGuardianId)
      handleCloseEdit()
      load()
    } catch {
      setEditError('Falha ao excluir responsável.')
    }
  }

  if (user?.role === 'SUPER_ADMIN') {
    return (
      <PageContainer>
        <Typography sx={{ mb: 2 }}>
          Como super admin, use o backoffice para gestão. Pacientes são listados por clínica no contexto do profissional.
        </Typography>
        <Button component={Link} to="/backoffice" variant="contained">
          Ir para Backoffice
        </Button>
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 1, mb: 2 }}>
        <Typography variant="h1" sx={{ margin: 0 }}>
          Pacientes
        </Typography>
        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          <Button
            variant="outlined"
            onClick={() => { setInviteModalOpen(true); setInviteSuccess(''); setFormError('') }}
          >
            Enviar invite para paciente
          </Button>
          <Button
            variant="contained"
            onClick={() => setModalOpen(true)}
            sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}
          >
            Novo paciente
          </Button>
        </Box>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>
          {error}
        </Alert>
      )}
      {loading && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 3 }}>
          <CircularProgress size={20} />
          <Typography color="text.secondary">Carregando...</Typography>
        </Box>
      )}

      {!loading && !error && (
        <TableContainer component={Paper} sx={{ overflowX: 'auto' }}>
          <Table size="medium" sx={{ minWidth: 320 }}>
            <TableHead>
              <TableRow sx={{ bgcolor: 'grey.50' }}>
                <TableCell sx={{ fontWeight: 600, color: 'text.secondary' }}>Paciente</TableCell>
                <TableCell sx={{ fontWeight: 600, color: 'text.secondary' }}>Data de nascimento</TableCell>
                <TableCell align="right" sx={{ fontWeight: 600, color: 'text.secondary' }}>
                  Ações
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {list.map((p) => (
                <TableRow key={p.id} hover>
                  <TableCell sx={{ fontWeight: 500 }}>{p.full_name}</TableCell>
                  <TableCell color="text.secondary">{formatBirthDate(p.birth_date)}</TableCell>
                  <TableCell align="right">
                    <IconButton size="small" onClick={() => handleOpenEdit(p)} title="Editar paciente" aria-label="Editar paciente">
                      <EditIcon fontSize="small" />
                    </IconButton>
                    <IconButton size="small" component={Link} to={`/patients/${p.id}/contracts`} title="Gerenciar contratos" aria-label="Gerenciar contratos">
                      <AssignmentIcon fontSize="small" />
                    </IconButton>
                    <IconButton
                      size="small"
                      component={Link}
                      to={`/patients/${p.id}/prontuario`}
                      title="Prontuário"
                      aria-label="Prontuário"
                      sx={{ bgcolor: actionColor, color: 'white', '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}
                    >
                      <DescriptionIcon fontSize="small" />
                    </IconButton>
                    {isImpersonated && (
                      <IconButton
                        size="small"
                        title="Excluir paciente"
                        aria-label="Excluir paciente"
                        color="error"
                        disabled={deletingPatientId === p.id}
                        onClick={() => handleSoftDeletePatient(p.id)}
                      >
                        <DeleteOutlineIcon fontSize="small" />
                      </IconButton>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {list.length === 0 && (
            <Box sx={{ py: 4, textAlign: 'center' }}>
              <Typography color="text.secondary">Nenhum paciente cadastrado.</Typography>
            </Box>
          )}
        </TableContainer>
      )}

      <AppDialog
        open={modalOpen}
        onClose={() => { setModalOpen(false); resetForm() }}
        title="Novo paciente"
        actions={
          <>
            <Button onClick={() => { setModalOpen(false); resetForm() }} color="inherit" sx={{ color: negationColor }}>
              Cancelar
            </Button>
            <Button variant="contained" type="submit" form="new-patient-form" disabled={submitting} sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>
              {submitting ? 'Salvando...' : 'Cadastrar'}
            </Button>
          </>
        }
      >
        <form id="new-patient-form" onSubmit={handleCreatePatient}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 0.5 }}>
            <FormControlLabel
              control={<Checkbox checked={samePerson} onChange={(e) => setSamePerson(e.target.checked)} />}
              label="Paciente e responsável são a mesma pessoa"
            />
            <TextField label="Nome do responsável (guardião legal)" value={guardianFullName} onChange={(e) => setGuardianFullName(e.target.value)} placeholder="Nome completo do responsável" fullWidth />
            <TextField label="Nome do paciente" value={patientFullName} onChange={(e) => setPatientFullName(e.target.value)} disabled={samePerson} placeholder={samePerson ? 'Igual ao responsável' : 'Nome completo do paciente'} fullWidth />
            <TextField label="E-mail do responsável (opcional)" type="email" value={guardianEmail} onChange={(e) => setGuardianEmail(e.target.value)} placeholder="email@exemplo.com" fullWidth />
            <TextField
              label="CPF do responsável (obrigatório)"
              value={guardianCpf}
              onChange={(e) => setGuardianCpf(e.target.value)}
              placeholder="Somente números (11 dígitos)"
              fullWidth
              required={guardianEmail.trim() !== ''}
              error={guardianEmail.trim() !== '' && !!guardianCpf.trim() && !isValidCPF(guardianCpf)}
              helperText={guardianEmail.trim() !== '' && guardianCpf.trim() && !isValidCPF(guardianCpf) ? 'CPF inválido.' : ' '}
            />
            <TextField
              label="CPF do paciente (opcional)"
              type={showPatientCPF ? 'text' : 'password'}
              value={patientCpf}
              onChange={(e) => setPatientCpf(e.target.value)}
              placeholder="Somente números (11 dígitos)"
              fullWidth
              error={!!patientCpf.trim() && !isValidCPF(patientCpf)}
              helperText={patientCpf.trim() && !isValidCPF(patientCpf) ? 'CPF inválido.' : ' '}
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton onClick={() => setShowPatientCPF((v) => !v)} edge="end" aria-label={showPatientCPF ? 'Ocultar CPF' : 'Mostrar CPF'}>
                      {showPatientCPF ? <VisibilityOffIcon /> : <VisibilityIcon />}
                    </IconButton>
                  </InputAdornment>
                ),
              }}
            />
            <Typography variant="subtitle2" color="text.secondary">Endereço do responsável (obrigatório se e-mail preenchido)</Typography>
            <TextField label="Rua" value={guardianRua} onChange={(e) => setGuardianRua(e.target.value)} fullWidth />
            <TextField label="Número" value={guardianNumero} onChange={(e) => setGuardianNumero(e.target.value)} fullWidth />
            <TextField label="Complemento" value={guardianComplemento} onChange={(e) => setGuardianComplemento(e.target.value)} fullWidth />
            <TextField label="Bairro" value={guardianBairro} onChange={(e) => setGuardianBairro(e.target.value)} fullWidth />
            <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
              <TextField label="Cidade" value={guardianCidade} onChange={(e) => setGuardianCidade(e.target.value)} sx={{ flex: 1, minWidth: 120 }} />
              <TextField label="Estado" value={guardianEstado} onChange={(e) => setGuardianEstado(e.target.value)} placeholder="UF" inputProps={{ maxLength: 2 }} sx={{ width: 80 }} />
              <TextField label="País" value={guardianPais} onChange={(e) => setGuardianPais(e.target.value)} sx={{ flex: 1, minWidth: 120 }} />
              <TextField label="CEP" value={guardianCep} onChange={(e) => setGuardianCep(e.target.value)} placeholder="00000000" inputProps={{ maxLength: 9 }} sx={{ width: 120 }} />
            </Box>
            <TextField label="Data de nascimento do responsável (obrigatório se e-mail preenchido)" type="date" value={guardianBirthDate} onChange={(e) => setGuardianBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
            <TextField label="Telefone (WhatsApp)" value={guardianPhone} onChange={(e) => setGuardianPhone(e.target.value)} placeholder="+5511999999999" fullWidth />
            <TextField label="Data de nascimento do paciente (obrigatório se e-mail do responsável preenchido)" type="date" value={newBirthDate} onChange={(e) => setNewBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
            {formError && <Alert severity="error">{formError}</Alert>}
          </Box>
        </form>
      </AppDialog>

      <AppDialog
        open={inviteModalOpen}
        onClose={() => { setInviteModalOpen(false); resetInviteForm() }}
        title="Enviar invite para paciente"
        actions={
          <>
            <Button onClick={() => { setInviteModalOpen(false); resetInviteForm() }} color="inherit" sx={{ color: negationColor }}>
              Cancelar
            </Button>
            <Button
              variant="contained"
              onClick={async () => {
                setFormError('')
                setInviteSuccess('')
                const email = inviteGuardianEmail.trim()
                const name = inviteGuardianFullName.trim()
                if (!name) {
                  setFormError('Nome do responsável é obrigatório.')
                  return
                }
                if (!email) {
                  setFormError('E-mail do responsável é obrigatório.')
                  return
                }
                if (!EMAIL_REGEX.test(email)) {
                  setFormError('E-mail do responsável inválido.')
                  return
                }
                setInviteSubmitting(true)
                try {
                  const res = await api.createPatientInvite(email, name)
                  setInviteSuccess(res.message || 'Convite enviado por e-mail.')
                } catch (e) {
                  setFormError((e as Error)?.message || 'Falha ao enviar convite.')
                } finally {
                  setInviteSubmitting(false)
                }
              }}
              disabled={inviteSubmitting}
              sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}
            >
              {inviteSubmitting ? 'Enviando...' : 'Enviar convite'}
            </Button>
          </>
        }
      >
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 0.5 }}>
          {formError && <Alert severity="error">{formError}</Alert>}
          {inviteSuccess && <Alert severity="success">{inviteSuccess}</Alert>}
          <Typography variant="body2" color="text.secondary">
            Informe o nome e o e-mail do responsável legal. Ele receberá um link para completar CPF, endereço e datas.
          </Typography>
          <TextField
            label="Nome completo do responsável"
            value={inviteGuardianFullName}
            onChange={(e) => setInviteGuardianFullName(e.target.value)}
            fullWidth
          />
          <TextField
            label="E-mail do responsável"
            type="email"
            value={inviteGuardianEmail}
            onChange={(e) => setInviteGuardianEmail(e.target.value)}
            fullWidth
          />
        </Box>
      </AppDialog>

      <AppDialog
        open={editModalOpen}
        onClose={handleCloseEdit}
        title="Editar paciente"
        actions={
          <>
            {isImpersonated && editHasGuardian && editGuardianId && (
              <Button variant="outlined" color="error" onClick={handleSoftDeleteGuardian}>
                Excluir responsável
              </Button>
            )}
            <Button onClick={handleCloseEdit} color="inherit" sx={{ color: negationColor }}>
              Cancelar
            </Button>
            <Button variant="contained" type="submit" form="edit-patient-form" disabled={editSubmitting}>
              {editSubmitting ? 'Salvando...' : 'Salvar'}
            </Button>
          </>
        }
      >
        <form id="edit-patient-form" onSubmit={handleSaveEdit}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 0.5 }}>
            <TextField label="Nome do paciente" value={editFullName} onChange={(e) => setEditFullName(e.target.value)} required fullWidth />
            <TextField label="Data de nascimento" type="date" value={editBirthDate} onChange={(e) => setEditBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
            <TextField label="E-mail do paciente (opcional)" type="email" value={editEmail} onChange={(e) => setEditEmail(e.target.value)} fullWidth />
            <TextField
              label="CPF do paciente (opcional)"
              type={showEditPatientCPF ? 'text' : 'password'}
              value={editPatientCpf}
              onChange={(e) => setEditPatientCpf(e.target.value)}
              placeholder="Somente números (11 dígitos)"
              fullWidth
              error={!!editPatientCpf.trim() && !isValidCPF(editPatientCpf)}
              helperText={editPatientCpf.trim() && !isValidCPF(editPatientCpf) ? 'CPF inválido.' : ' '}
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton onClick={() => setShowEditPatientCPF((v) => !v)} edge="end" aria-label={showEditPatientCPF ? 'Ocultar CPF' : 'Mostrar CPF'}>
                      {showEditPatientCPF ? <VisibilityOffIcon /> : <VisibilityIcon />}
                    </IconButton>
                  </InputAdornment>
                ),
              }}
            />
            {editHasGuardian && (
              <>
                <Typography variant="subtitle2" sx={{ mt: 1 }}>Responsável legal</Typography>
                <TextField label="Nome do responsável" value={editGuardianFullName} onChange={(e) => setEditGuardianFullName(e.target.value)} fullWidth />
                <TextField label="E-mail do responsável" type="email" value={editGuardianEmail} onChange={(e) => setEditGuardianEmail(e.target.value)} fullWidth />
                <TextField label="Data de nascimento do responsável" type="date" value={editGuardianBirthDate} onChange={(e) => setEditGuardianBirthDate(e.target.value)} InputLabelProps={{ shrink: true }} fullWidth />
                <TextField label="Telefone (WhatsApp)" value={editGuardianPhone} onChange={(e) => setEditGuardianPhone(e.target.value)} placeholder="+5511999999999" fullWidth />
                <Typography variant="subtitle2" color="text.secondary">Endereço</Typography>
                <TextField label="Rua" value={editGuardianRua} onChange={(e) => setEditGuardianRua(e.target.value)} fullWidth />
                <TextField label="Número" value={editGuardianNumero} onChange={(e) => setEditGuardianNumero(e.target.value)} fullWidth />
                <TextField label="Complemento" value={editGuardianComplemento} onChange={(e) => setEditGuardianComplemento(e.target.value)} fullWidth />
                <TextField label="Bairro" value={editGuardianBairro} onChange={(e) => setEditGuardianBairro(e.target.value)} fullWidth />
                <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                  <TextField label="Cidade" value={editGuardianCidade} onChange={(e) => setEditGuardianCidade(e.target.value)} sx={{ flex: 1, minWidth: 120 }} />
                  <TextField label="Estado" value={editGuardianEstado} onChange={(e) => setEditGuardianEstado(e.target.value)} placeholder="UF" inputProps={{ maxLength: 2 }} sx={{ width: 80 }} />
                  <TextField label="País" value={editGuardianPais} onChange={(e) => setEditGuardianPais(e.target.value)} sx={{ flex: 1, minWidth: 120 }} />
                  <TextField label="CEP" value={editGuardianCep} onChange={(e) => setEditGuardianCep(e.target.value)} placeholder="00000000" inputProps={{ maxLength: 9 }} sx={{ width: 120 }} />
                </Box>
                <TextField
                  label="CPF do responsável (deixe em branco para não alterar)"
                  type={showEditGuardianCPF ? 'text' : 'password'}
                  value={editGuardianCpf}
                  onChange={(e) => setEditGuardianCpf(e.target.value)}
                  placeholder="Somente números (11 dígitos)"
                  fullWidth
                  error={!!editGuardianCpf.trim() && !isValidCPF(editGuardianCpf)}
                  helperText={editGuardianCpf.trim() && !isValidCPF(editGuardianCpf) ? 'CPF inválido.' : ' '}
                  InputProps={{
                    endAdornment: (
                      <InputAdornment position="end">
                        <IconButton onClick={() => setShowEditGuardianCPF((v) => !v)} edge="end" aria-label={showEditGuardianCPF ? 'Ocultar CPF' : 'Mostrar CPF'}>
                          {showEditGuardianCPF ? <VisibilityOffIcon /> : <VisibilityIcon />}
                        </IconButton>
                      </InputAdornment>
                    ),
                  }}
                />
              </>
            )}
            {editError && <Alert severity="error">{editError}</Alert>}
          </Box>
        </form>
      </AppDialog>
    </PageContainer>
  )
}
