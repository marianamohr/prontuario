import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Box, Typography, Button, FormControl, InputLabel, Select, MenuItem, FormControlLabel, Checkbox, AppBar, Toolbar } from '@mui/material'

const BASE = (import.meta.env.VITE_API_URL || '').replace(/\/$/, '')

const SIGNATURE_FONTS = [
  { value: 'cursive', label: 'Cursiva' },
  { value: 'brush', label: 'Brush Script' },
  { value: 'dancing', label: 'Dancing Script' },
] as const

function getSignatureFontFamily(fontKey: string): string {
  switch (fontKey) {
    case 'brush':
      return "'Brush Script MT', cursive"
    case 'dancing':
      return "'Dancing Script', cursive"
    case 'cursive':
    default:
      return "'Segoe Script', 'Dancing Script', cursive"
  }
}

/** Formata data no padrão DD/MM/AAAA (com zero à esquerda). */
function formatDateDDMMAAAA(date: Date): string {
  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const year = date.getFullYear()
  return `${day}/${month}/${year}`
}

function buildGuardianSignatureHTML(guardianName: string, fontKey: string): string {
  const safeName = guardianName.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  const fontFamily = getSignatureFontFamily(fontKey)
  return `<span style="font-family: ${fontFamily}; font-size: 1.25em;">${safeName}</span>`
}

type ContractInfo = {
  contract_id: string
  patient_name: string
  guardian_name: string
  body_html: string
  signer_relation: string
  signer_is_patient: boolean
  status: string
  clinic_name: string
}

export function SignContract() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') || ''
  const [contract, setContract] = useState<ContractInfo | null>(null)
  const [accepted, setAccepted] = useState(false)
  const [signatureFont, setSignatureFont] = useState<string>('cursive')
  const [loading, setLoading] = useState(!!token)
  const [signing, setSigning] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    if (!token) {
      setError('Token não informado.')
      setLoading(false)
      return
    }
    fetch(`${BASE}/api/contracts/by-token?token=${encodeURIComponent(token)}`)
      .then((r) => {
        if (!r.ok) throw new Error('Token inválido ou expirado.')
        return r.json()
      })
      .then((data: unknown) => {
        if (!data || typeof data !== 'object' || Array.isArray(data)) {
          setError('Resposta inválida do servidor.')
          return
        }
        const o = data as Record<string, unknown>
        const contractId = typeof o.contract_id === 'string' ? o.contract_id : ''
        const bodyHtml = typeof o.body_html === 'string' ? o.body_html : ''
        if (!contractId || bodyHtml === undefined) {
          setError('Dados do contrato incompletos.')
          return
        }
        setContract({
          contract_id: contractId,
          patient_name: typeof o.patient_name === 'string' ? o.patient_name : '',
          guardian_name: typeof o.guardian_name === 'string' ? o.guardian_name : '',
          body_html: bodyHtml,
          signer_relation: typeof o.signer_relation === 'string' ? o.signer_relation : '',
          signer_is_patient: o.signer_is_patient === true,
          status: typeof o.status === 'string' ? o.status : '',
          clinic_name: typeof o.clinic_name === 'string' ? o.clinic_name : '',
        })
      })
      .catch(() => setError('Token inválido ou expirado.'))
      .finally(() => setLoading(false))
  }, [token])

  const bodyHtmlWithSignature = useMemo(() => {
    let html = contract?.body_html ?? ''
    const name = contract?.guardian_name ?? ''
    if (!html) return html
    // Assinatura do responsável (fonte escolhida)
    if (name) {
      const sigHTML = buildGuardianSignatureHTML(name, signatureFont)
      html = html.replace(/\[ASSINATURA_RESPONSAVEL\]/g, sigHTML)
    }
    // Apenas [DATA] é substituída (dia em que a pessoa abriu o contrato, DD/MM/AAAA). Local já vem escrito no template.
    const dataHoje = formatDateDDMMAAAA(new Date())
    if (html.includes('[DATA]')) html = html.replace(/\[DATA\]/g, dataHoje)
    return html
  }, [contract?.body_html, contract?.guardian_name, signatureFont])

  const handleSign = async () => {
    if (!accepted || !token) return
    setSigning(true)
    setError('')
    const headers: HeadersInit = { 'Content-Type': 'application/json' }
    const stored = localStorage.getItem('token')
    if (stored) (headers as Record<string, string>)['Authorization'] = `Bearer ${stored}`
    try {
      const res = await fetch(`${BASE}/api/contracts/sign`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ token, accepted_terms: true, signature_font: signatureFont }),
      })
      if (!res.ok) {
        const t = await res.text()
        throw new Error(t || 'Falha ao assinar')
      }
      setSuccess(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Falha ao assinar.')
    } finally {
      setSigning(false)
    }
  }

  const pageClass = 'sign-contract-page'
  const headerTitle = contract?.clinic_name?.trim() || 'Contrato'

  const header = (
    <AppBar position="static" elevation={0} sx={{ bgcolor: 'background.paper', color: 'text.primary', borderBottom: 1, borderColor: 'divider' }}>
      <Toolbar>
        <Typography variant="h6" component="h1">{headerTitle}</Typography>
      </Toolbar>
    </AppBar>
  )

  if (loading) {
    return (
      <Box className={pageClass} sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
        {header}
        <Box className="sign-contract-loading" sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <Typography color="text.secondary">Carregando contrato...</Typography>
        </Box>
      </Box>
    )
  }
  if (error && !contract) {
    return (
      <Box className={pageClass} sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
        {header}
        <Box className="sign-contract-error-only" sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <Typography color="error">{error}</Typography>
        </Box>
      </Box>
    )
  }
  if (success) {
    return (
      <Box className={pageClass} sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
        {header}
        <Box className="sign-contract-success" sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <Box>
            <Typography variant="h5" component="h2" sx={{ mb: 0.5 }}>Contrato assinado com sucesso</Typography>
            <Typography color="text.secondary">Uma cópia foi enviada ao seu e-mail.</Typography>
          </Box>
        </Box>
      </Box>
    )
  }
  if (!contract) {
    return (
      <Box className={pageClass} sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
        {header}
        <Box className="sign-contract-error-only" sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
          <Typography color="text.secondary">Não foi possível carregar o contrato. Verifique o link ou tente novamente.</Typography>
        </Box>
      </Box>
    )
  }

  return (
    <Box className={pageClass} sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {header}
      <Box sx={{ flex: 1, width: '100%', maxWidth: '100%', boxSizing: 'border-box', p: 2 }}>
      <Typography variant="h4" sx={{ mb: 2 }}>Assinar contrato</Typography>
      <Box sx={{ mb: 2, p: 1.5, bgcolor: 'grey.50', borderRadius: 1 }}>
        <Typography><strong>Paciente:</strong> {contract?.patient_name ?? '—'}</Typography>
        <Typography><strong>Responsável/Assinante:</strong> {contract?.guardian_name ?? '—'}</Typography>
        <Typography><strong>Relação:</strong> {contract?.signer_relation ?? '—'}</Typography>
      </Box>
      <FormControl fullWidth size="small" sx={{ mb: 1 }}>
        <InputLabel>Fonte da sua assinatura</InputLabel>
        <Select value={signatureFont} label="Fonte da sua assinatura" onChange={(e) => setSignatureFont(e.target.value)}>
          {SIGNATURE_FONTS.map((f) => (
            <MenuItem key={f.value} value={f.value}>{f.label}</MenuItem>
          ))}
        </Select>
      </FormControl>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>Sua assinatura aparecerá no contrato com a fonte escolhida.</Typography>
      <Box className="sign-contract-preview" dangerouslySetInnerHTML={{ __html: bodyHtmlWithSignature }} sx={{ mb: 2 }} />
      <FormControlLabel control={<Checkbox checked={accepted} onChange={(e) => setAccepted(e.target.checked)} />} label="Li e concordo com os termos do contrato." sx={{ display: 'block', mb: 1 }} />
      {error && <Typography color="error" sx={{ mb: 2 }}>{error}</Typography>}
      <Button variant="contained" disabled={!accepted || signing} onClick={handleSign} sx={{ bgcolor: accepted && !signing ? undefined : 'grey.400' }}>
        {signing ? 'Assinando...' : 'Assinar'}
      </Button>
      </Box>
    </Box>
  )
}
