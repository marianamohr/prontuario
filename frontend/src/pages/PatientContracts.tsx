import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Box,
  Typography,
  Button,
  Paper,
  Alert,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  TextField,
  IconButton,
} from '@mui/material'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'
import { useTheme } from '@mui/material/styles'
import { PageContainer } from '../components/ui/PageContainer'
import { AppDialog } from '../components/ui/AppDialog'
import * as api from '../lib/api'

export function PatientContracts() {
  const theme = useTheme()
  const { patientId } = useParams<{ patientId: string }>()
  const { user, isImpersonated } = useAuth()
  const branding = useBranding()?.branding ?? null
  const actionColor = user?.role === 'PROFESSIONAL' && branding?.action_button_color ? branding.action_button_color : theme.palette.primary.main
  const canSendContract = user?.role === 'PROFESSIONAL' || user?.role === 'SUPER_ADMIN'
  const [guardians, setGuardians] = useState<api.GuardianInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [contractModalOpen, setContractModalOpen] = useState(false)
  const [templates, setTemplates] = useState<api.ContractTemplateItem[]>([])
  const [selectedGuardianId, setSelectedGuardianId] = useState('')
  const [selectedTemplateId, setSelectedTemplateId] = useState('')
  const [contractDataInicio, setContractDataInicio] = useState('')
  const [contractDataFim, setContractDataFim] = useState('')
  const [contractValor, setContractValor] = useState('')
  const [contractPeriodicidade, setContractPeriodicidade] = useState('')
  const [sendingContract, setSendingContract] = useState(false)
  const [previewPdfLoading, setPreviewPdfLoading] = useState(false)
  const [contractMessage, setContractMessage] = useState('')
  const [patientContracts, setPatientContracts] = useState<api.PatientContractItem[]>([])
  const [resendingId, setResendingId] = useState<string | null>(null)
  const [cancellingId, setCancellingId] = useState<string | null>(null)
  const [deletingContractId, setDeletingContractId] = useState<string | null>(null)
  const [scheduleRules, setScheduleRules] = useState<api.ScheduleRule[]>([])
  const [contractNumAppointments, setContractNumAppointments] = useState<number | ''>('')
  const [endingContractId, setEndingContractId] = useState<string | null>(null)
  const [endContractDate, setEndContractDate] = useState('')
  const [showAllContracts, setShowAllContracts] = useState(false)
  const INITIAL_CONTRACTS_COUNT = 5

  const load = useCallback(() => {
    if (!patientId) return
    setError('')
    setLoading(true)
    api
      .listPatientGuardians(patientId)
      .then((r) => setGuardians(r.guardians))
      .catch(() => {
        setGuardians([])
        setError('Falha ao carregar responsáveis.')
      })
      .finally(() => setLoading(false))
    if (canSendContract) {
      api
        .listPatientContracts(patientId)
        .then((r) => setPatientContracts(r.contracts))
        .catch(() => {
          setPatientContracts([])
          setError('Falha ao carregar contratos.')
        })
    }
  }, [patientId, canSendContract])

  useEffect(() => {
    load()
  }, [load])

  const openContractModal = () => {
    setContractModalOpen(true)
    setContractMessage('')
    setSelectedGuardianId(guardians[0]?.id ?? '')
    setSelectedTemplateId('')
    setContractDataInicio('')
    setContractDataFim('')
    setContractValor('')
    setContractPeriodicidade('')
    setScheduleRules([])
    setContractNumAppointments('')
    api.listContractTemplates().then((r) => {
      setTemplates(r.templates)
      if (r.templates.length > 0) setSelectedTemplateId(r.templates[0].id)
    }).catch(() => setTemplates([]))
  }

  const formatValorPorSessao = (input: string): string => {
    const cleaned = input.replace(/\s/g, '').replace(/[R$\s]/gi, '').replace(',', '.')
    const num = parseFloat(cleaned)
    if (Number.isNaN(num) || num < 0) return ''
    const formatted = num.toFixed(2).replace('.', ',')
    return `valor de R$ ${formatted} por sessão`
  }

  const handleSendContract = async () => {
    if (!patientId || !selectedGuardianId || !selectedTemplateId) return
    const valorTrim = contractValor.trim()
    if (!valorTrim) {
      setContractMessage('Informe o valor por sessão.')
      return
    }
    const valorFormatado = formatValorPorSessao(valorTrim)
    if (!valorFormatado) {
      setContractMessage('Informe um valor numérico válido (ex.: 150 ou 150,50).')
      return
    }
    setSendingContract(true)
    setContractMessage('')
    try {
      await api.sendContractForPatient(
        patientId,
        selectedGuardianId,
        selectedTemplateId,
        contractDataInicio || undefined,
        contractDataFim || undefined,
        valorFormatado,
        contractPeriodicidade.trim() || undefined,
        scheduleRules.length > 0 ? scheduleRules : undefined,
        undefined,
        undefined,
        contractNumAppointments !== '' && Number(contractNumAppointments) > 0 ? Number(contractNumAppointments) : undefined
      )
      setContractMessage('Contrato enviado por e-mail para assinatura.')
      api.listPatientContracts(patientId).then((r) => setPatientContracts(r.contracts)).catch(() => {})
      setTimeout(() => setContractModalOpen(false), 1500)
    } catch {
      setContractMessage('Falha ao enviar contrato.')
    } finally {
      setSendingContract(false)
    }
  }

  const handleResendContract = async (contractId: string) => {
    if (!patientId) return
    setResendingId(contractId)
    setContractMessage('')
    try {
      await api.resendPatientContract(patientId, contractId)
      setContractMessage('Contrato reenviado por e-mail.')
      api.listPatientContracts(patientId).then((r) => setPatientContracts(r.contracts))
    } catch {
      setContractMessage('Falha ao reenviar contrato.')
    } finally {
      setResendingId(null)
    }
  }

  const handleEndContract = async () => {
    if (!patientId || !endingContractId || !endContractDate) return
    setContractMessage('')
    try {
      await api.endContract(patientId, endingContractId, endContractDate)
      setContractMessage('Contrato encerrado. Agendamentos a partir da data foram finalizados.')
      setEndingContractId(null)
      setEndContractDate('')
      api.listPatientContracts(patientId).then((r) => setPatientContracts(r.contracts))
    } catch (e) {
      let msg = 'Falha ao encerrar contrato.'
      if (e instanceof Error && e.message) {
        try {
          const parsed = JSON.parse(e.message) as { error?: string }
          if (parsed?.error) msg = parsed.error
        } catch {
          if (e.message.length < 120) msg = e.message
        }
      }
      setContractMessage(msg)
    }
  }

  const handleCancelContract = async (contractId: string) => {
    if (!patientId) return
    if (!window.confirm('Cancelar este contrato? O contrato será tornado ineligível e os agendamentos vinculados serão cancelados. O responsável será notificado por e-mail.')) return
    setCancellingId(contractId)
    setContractMessage('')
    try {
      const res = await api.cancelPatientContract(patientId, contractId)
      setContractMessage(res.message ?? 'Contrato cancelado. O responsável foi notificado por e-mail.')
      api.listPatientContracts(patientId).then((r) => setPatientContracts(r.contracts))
    } catch {
      setContractMessage('Falha ao cancelar contrato.')
    } finally {
      setCancellingId(null)
    }
  }

  const handleSoftDeleteContract = async (contractId: string) => {
    if (!patientId) return
    const ok = window.confirm('Tem certeza que deseja excluir (soft delete) este contrato? Ele vai sumir do front para profissionais.')
    if (!ok) return
    setDeletingContractId(contractId)
    try {
      await api.softDeleteContract(patientId, contractId)
      setContractMessage('Contrato excluído.')
      load()
    } catch {
      setContractMessage('Falha ao excluir contrato.')
    } finally {
      setDeletingContractId(null)
    }
  }

  const handlePreviewContract = () => {
    if (!patientId || !selectedGuardianId || !selectedTemplateId) return
    const valorTrim = contractValor.trim()
    const valorFormatado = valorTrim ? formatValorPorSessao(valorTrim) : undefined
    if (valorTrim && !valorFormatado) {
      setContractMessage('Informe um valor numérico válido para o preview.')
      return
    }
    setContractMessage('')
    setPreviewPdfLoading(true)
    api
      .getContractPreview(
        patientId,
        selectedGuardianId,
        selectedTemplateId,
        contractDataInicio || undefined,
        contractDataFim || undefined,
        valorFormatado,
        contractPeriodicidade.trim() || undefined
      )
      .then(({ body_html }) => {
        const iframe = document.createElement('iframe')
        iframe.setAttribute('style', 'position:absolute;width:210mm;height:297mm;left:-9999px;top:0;border:none;')
        document.body.appendChild(iframe)
        const doc = iframe.contentDocument
        if (!doc) {
          setPreviewPdfLoading(false)
          document.body.removeChild(iframe)
          setContractMessage('Não foi possível gerar o preview.')
          return
        }
        doc.open()
        doc.write(body_html)
        doc.close()
        iframe.onload = () => {
          setTimeout(() => {
            const body = iframe.contentDocument?.body
            if (!body) {
              setPreviewPdfLoading(false)
              document.body.removeChild(iframe)
              return
            }
            const opt = {
              margin: 10,
              filename: 'preview-contrato.pdf',
              image: { type: 'jpeg' as const, quality: 0.98 },
              html2canvas: { scale: 2, useCORS: true },
              jsPDF: { unit: 'mm' as const, format: 'a4' as const, orientation: 'portrait' as const },
            }
            import('html2pdf.js').then(({ default: html2pdf }) =>
              html2pdf()
                .set(opt)
                .from(body)
                .outputPdf('blob')
                .then(
                  (blob: Blob) => {
                    const url = URL.createObjectURL(blob)
                    window.open(url, '_blank', 'noopener,noreferrer')
                    URL.revokeObjectURL(url)
                    document.body.removeChild(iframe)
                    setPreviewPdfLoading(false)
                  },
                  () => {
                    if (document.body.contains(iframe)) document.body.removeChild(iframe)
                    setPreviewPdfLoading(false)
                  }
                )
            )
          }, 100)
        }
      })
      .catch(() => {
        setPreviewPdfLoading(false)
        setContractMessage('Falha ao carregar preview.')
      })
  }

  if (!patientId) return null

  const DAY_NAMES = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

  return (
    <PageContainer>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, alignItems: 'center', mb: 2 }}>
        <Typography component={Link} to="/patients" sx={{ color: 'primary.main', textDecoration: 'none' }}>← Pacientes</Typography>
        <Typography component="span" color="text.secondary">·</Typography>
        <Typography component={Link} to={`/patients/${patientId}/prontuario`} sx={{ color: 'primary.main', textDecoration: 'none' }}>Prontuário</Typography>
      </Box>
      <Typography variant="h4" sx={{ mb: 2 }}>Contratos do paciente</Typography>
      {!loading && guardians.length > 0 && (
        <Alert severity="info" sx={{ mb: 2 }}>
          <strong>Responsável(is) legal(is):</strong>{' '}
          {guardians.map((g) => `${g.full_name}${g.relation ? ` (${g.relation})` : ''}`).join('; ')}
        </Alert>
      )}
      {loading && <Typography color="text.secondary">Carregando...</Typography>}
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {canSendContract && !loading && guardians.length > 0 && (
        <Box sx={{ mb: 2 }}>
          <Button variant="contained" onClick={openContractModal} sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>
            Disparar contrato para assinar
          </Button>
        </Box>
      )}
      {canSendContract && patientContracts.length > 0 && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2, bgcolor: 'grey.50' }}>
          <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1.5 }}>Contratos enviados</Typography>
          <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
            {(showAllContracts ? patientContracts : patientContracts.slice(0, INITIAL_CONTRACTS_COUNT)).map((c) => (
              <Paper key={c.id} variant="outlined" sx={{ p: 1.5, mb: 0.75, display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 0.5, justifyContent: 'space-between' }}>
                <Box sx={{ flex: '1 1 200px' }}>
                  <Typography fontWeight={600}>{c.template_name} — {c.guardian_name}</Typography>
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 0.25 }}>
                    {c.status === 'CANCELLED' ? (
                      <Typography component="span" color="text.secondary">Contrato cancelado</Typography>
                    ) : c.status === 'ENDED' ? (
                      <Typography component="span" color="text.secondary">Contrato encerrado</Typography>
                    ) : c.status === 'SIGNED' ? (
                      <Typography component="span" color="success.main">Assinado{c.signed_at ? ` em ${new Date(c.signed_at).toLocaleDateString('pt-BR')}` : ''}</Typography>
                    ) : (
                      <Typography component="span" color="warning.main">Pendente de assinatura</Typography>
                    )}
                  </Typography>
                </Box>
                <Box sx={{ display: 'flex', gap: 0.5, flexShrink: 0, flexWrap: 'wrap', alignItems: 'center' }}>
                  {isImpersonated && (
                    <IconButton
                      size="small"
                      title="Excluir contrato"
                      aria-label="Excluir contrato"
                      color="error"
                      disabled={deletingContractId === c.id}
                      onClick={() => handleSoftDeleteContract(c.id)}
                    >
                      <DeleteOutlineIcon fontSize="small" />
                    </IconButton>
                  )}
                  {c.status === 'CANCELLED' ? (
                    <Typography variant="body2" color="text.secondary">Contrato cancelado</Typography>
                  ) : c.status === 'ENDED' ? (
                    c.verify_url ? (
                      <Button size="small" variant="contained" href={c.verify_url} target="_blank" rel="noopener noreferrer" sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>
                        Ver contrato assinado
                      </Button>
                    ) : null
                  ) : (
                    <>
                      {c.status === 'PENDING' && (
                        <Button size="small" variant="outlined" color="warning" disabled={resendingId === c.id} onClick={() => handleResendContract(c.id)}>
                          {resendingId === c.id ? 'Reenviando...' : 'Reenviar'}
                        </Button>
                      )}
                      {c.status === 'SIGNED' && c.verify_url && (
                        <Button size="small" variant="contained" href={c.verify_url} target="_blank" rel="noopener noreferrer" sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>
                          Ver contrato assinado
                        </Button>
                      )}
                      {c.status === 'SIGNED' && (
                        <Button size="small" variant="outlined" onClick={() => { setEndingContractId(c.id); setEndContractDate(''); setContractMessage('') }}>
                          Encerrar contrato
                        </Button>
                      )}
                      {(c.status === 'PENDING' || c.status === 'SIGNED') && (
                        <Button size="small" variant="outlined" color="error" disabled={cancellingId === c.id} onClick={() => handleCancelContract(c.id)}>
                          {cancellingId === c.id ? 'Cancelando...' : 'Cancelar'}
                        </Button>
                      )}
                    </>
                  )}
                </Box>
              </Paper>
            ))}
          </Box>
          {patientContracts.length > INITIAL_CONTRACTS_COUNT && (
            <Button size="small" onClick={() => setShowAllContracts((v) => !v)} sx={{ mt: 1 }}>
              {showAllContracts ? 'Ver menos' : `Ver mais (${patientContracts.length - INITIAL_CONTRACTS_COUNT} restantes)`}
            </Button>
          )}
          {contractMessage && patientContracts.length > 0 && (
            <Alert severity={contractMessage.includes('Falha') ? 'error' : 'success'} sx={{ mt: 0.5, fontSize: 14 }}>{contractMessage}</Alert>
          )}
        </Paper>
      )}
      {!loading && canSendContract && guardians.length === 0 && (
        <Typography color="text.secondary">Nenhum responsável com e-mail cadastrado. Cadastre um responsável com e-mail para enviar contratos.</Typography>
      )}
      {!loading && !canSendContract && (
        <Typography color="text.secondary">Apenas profissionais podem gerenciar contratos.</Typography>
      )}

      <AppDialog
        open={contractModalOpen}
        onClose={() => setContractModalOpen(false)}
        title="Enviar contrato para assinatura"
        maxWidth="md"
        actions={
          <>
            <Button variant="outlined" onClick={handlePreviewContract} disabled={previewPdfLoading || !selectedGuardianId || !selectedTemplateId || !contractValor.trim()}>
              {previewPdfLoading ? 'Gerando PDF...' : 'Preview em PDF'}
            </Button>
            <Button onClick={() => setContractModalOpen(false)} color="inherit">Fechar</Button>
            <Button variant="contained" onClick={handleSendContract} disabled={sendingContract || !selectedGuardianId || !selectedTemplateId || !contractValor.trim()} sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>
              {sendingContract ? 'Enviando...' : 'Enviar por e-mail'}
            </Button>
          </>
        }
      >
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2, maxHeight: '60vh', overflow: 'auto' }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <FormControl fullWidth size="small">
              <InputLabel>Responsável</InputLabel>
              <Select value={selectedGuardianId} label="Responsável" onChange={(e) => setSelectedGuardianId(e.target.value)}>
                {guardians.map((g) => (
                  <MenuItem key={g.id} value={g.id}>{g.full_name} – {g.email}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl fullWidth size="small">
              <InputLabel>Modelo de contrato</InputLabel>
              <Select value={selectedTemplateId} label="Modelo de contrato" onChange={(e) => setSelectedTemplateId(e.target.value)}>
                <MenuItem value="">Selecione...</MenuItem>
                {templates.map((t) => (
                  <MenuItem key={t.id} value={t.id}>{t.name}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <TextField type="date" label="Data de início" size="small" fullWidth value={contractDataInicio} onChange={(e) => setContractDataInicio(e.target.value)} InputLabelProps={{ shrink: true }} />
            <TextField type="date" label="Data de término" size="small" fullWidth value={contractDataFim} onChange={(e) => setContractDataFim(e.target.value)} InputLabelProps={{ shrink: true }} />
            <TextField label="Valor por sessão (R$) *" size="small" fullWidth value={contractValor} onChange={(e) => { setContractValor(e.target.value); setContractMessage('') }} placeholder="Ex.: 150 ou 150,50" />
            <Typography variant="caption" color="text.secondary">Exibido no contrato como &quot;valor de R$ XX,XX por sessão&quot;</Typography>
            <TextField label="Periodicidade (opcional)" size="small" fullWidth value={contractPeriodicidade} onChange={(e) => setContractPeriodicidade(e.target.value)} placeholder="Ex.: semanal, quinzenal, mensal" />
          </Box>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <TextField type="number" label="Quantidade de agendamentos (opcional)" size="small" fullWidth value={contractNumAppointments === '' ? '' : contractNumAppointments} onChange={(e) => { const v = e.target.value; setContractNumAppointments(v === '' ? '' : Math.max(1, parseInt(v, 10) || 1)) }} placeholder="Ex.: 4" inputProps={{ min: 1 }} />
            <Typography variant="caption" color="text.secondary">Pré-agendar consultas (opcional)</Typography>
            {scheduleRules.map((r, i) => (
              <Box key={i} sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
                <FormControl size="small" sx={{ minWidth: 100 }}>
                  <Select value={r.day_of_week} onChange={(e) => setScheduleRules((prev) => prev.map((x, j) => j === i ? { ...x, day_of_week: Number(e.target.value) } : x))}>
                    {DAY_NAMES.map((name, d) => (
                      <MenuItem key={d} value={d}>{name}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <TextField type="time" size="small" value={r.slot_time} onChange={(e) => setScheduleRules((prev) => prev.map((x, j) => j === i ? { ...x, slot_time: e.target.value } : x))} sx={{ width: 110 }} InputLabelProps={{ shrink: true }} />
                <IconButton size="small" color="error" onClick={() => setScheduleRules((prev) => prev.filter((_, j) => j !== i))} aria-label="Remover"><DeleteOutlineIcon fontSize="small" /></IconButton>
              </Box>
            ))}
            <Button size="small" variant="outlined" onClick={() => setScheduleRules((prev) => [...prev, { day_of_week: 1, slot_time: '09:00' }])}>+ Adicionar horário</Button>
          </Box>
        </Box>
        {contractMessage && <Alert severity={contractMessage.includes('Falha') ? 'error' : 'success'} sx={{ mt: 1 }}>{contractMessage}</Alert>}
      </AppDialog>

      <AppDialog
        open={!!endingContractId}
        onClose={() => { setEndingContractId(null); setContractMessage('') }}
        title="Encerrar contrato"
        actions={
          <>
            <Button onClick={() => { setEndingContractId(null); setContractMessage('') }} color="inherit">Cancelar</Button>
            <Button variant="contained" onClick={handleEndContract} disabled={!endContractDate} sx={{ bgcolor: actionColor, '&:hover': { bgcolor: actionColor, opacity: 0.9 } }}>Confirmar encerramento</Button>
          </>
        }
      >
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>Informe a data de término. Os agendamentos a partir dessa data serão finalizados na agenda.</Typography>
        <TextField type="date" label="Data de término" fullWidth value={endContractDate} onChange={(e) => setEndContractDate(e.target.value)} InputLabelProps={{ shrink: true }} sx={{ mb: 1 }} />
        {contractMessage && <Alert severity={contractMessage.includes('Falha') ? 'error' : 'success'} sx={{ mb: 1 }}>{contractMessage}</Alert>}
      </AppDialog>
    </PageContainer>
  )
}
