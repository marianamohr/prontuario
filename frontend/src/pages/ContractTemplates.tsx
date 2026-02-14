import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import html2pdf from 'html2pdf.js'
import {
  Box,
  Typography,
  Button,
  Paper,
  TextField,
  Alert,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
} from '@mui/material'
import PreviewIcon from '@mui/icons-material/Preview'
import EditIcon from '@mui/icons-material/Edit'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import { useAuth } from '../contexts/AuthContext'
import { PageContainer } from '../components/ui/PageContainer'
import { AppDialog } from '../components/ui/AppDialog'
import * as api from '../lib/api'

const emptyForm = { name: '', body_html: '' }

const PRESET_PRESTACAO_SERVICOS = {
  name: 'Contrato de Prestação de Serviços',
  body_html: `<!DOCTYPE html>
<html>
<head><meta charset="utf-8">
<style>
  body { font-family: 'Georgia', 'Times New Roman', serif; margin: 0 auto; padding: 2rem 1.25rem; line-height: 1.65; color: #1f2937; font-size: 14px; }
  .doc-title { font-size: 1.1rem; font-weight: 700; letter-spacing: 0.02em; color: #111827; text-align: center; margin-bottom: 1.25rem; text-transform: uppercase; }
  .intro { margin-bottom: 1.5rem; color: #374151; }
  .block { margin-bottom: 1.25rem; }
  .label { font-size: 0.8rem; font-weight: 700; color: #374151; text-transform: uppercase; letter-spacing: 0.03em; margin-bottom: 0.25rem; }
  .value { color: #4b5563; }
  .clause { margin-bottom: 1.25rem; }
  .clause-title { font-size: 0.8rem; font-weight: 700; color: #111827; text-transform: uppercase; letter-spacing: 0.02em; margin-bottom: 0.35rem; }
  .clause-text { font-size: 0.9rem; color: #4b5563; line-height: 1.6; }
  .signatures { margin-top: 2rem; padding-top: 1.25rem; border-top: 1px solid #e5e7eb; }
  .sign-line { margin-bottom: 1rem; }
  .sign-label { font-size: 0.75rem; font-weight: 600; color: #6b7280; margin-bottom: 0.2rem; }
  .sign-value { min-height: 1.2em; color: #374151; }
</style>
</head>
<body>
<p class="doc-title">Contrato de Prestação de Serviços</p>
<p class="intro">Pelo presente instrumento particular, as partes abaixo qualificadas celebram o presente Contrato de Prestação de Serviços, que se regerá pelas cláusulas e condições a seguir:</p>
<div class="block"><p class="label">Contratado (prestador de serviços):</p><p class="value">[CONTRATADO]</p></div>
<div class="block"><p class="label">Contratante (responsável legal pelo paciente):</p><p class="value">[CONTRATANTE]</p></div>
<div class="block"><p class="label">Dados do paciente:</p><p class="value">Nome: [PACIENTE_NOME] &nbsp; Data de nascimento: [PACIENTE_NASCIMENTO]</p></div>
<div class="block"><p class="label">Dados do responsável legal:</p><p class="value">Nome: [RESPONSAVEL_NOME] | E-mail: [RESPONSAVEL_EMAIL] | CPF: [RESPONSAVEL_CPF]<br>Endereço: [RESPONSAVEL_ENDERECO] | Data de nascimento: [RESPONSAVEL_NASCIMENTO]</p></div>
<div class="clause"><p class="clause-title">Cláusula 1ª – Do objeto</p><p class="clause-text">O presente contrato tem por objeto a prestação de serviços de [TIPO_SERVICO], nas condições estabelecidas entre as partes.</p></div>
<div class="clause"><p class="clause-title">Cláusula 2ª – Da periodicidade</p><p class="clause-text">Os serviços serão prestados com periodicidade [PERIODICIDADE], conforme combinado entre as partes.</p></div>
<div class="clause"><p class="clause-title">Cláusula 3ª – Do prazo</p><p class="clause-text">O presente contrato tem data de início em [DATA_INICIO] e [DATA_FIM].</p></div>
<div class="clause"><p class="clause-title">Cláusula 4ª – Do valor</p><p class="clause-text">Os serviços têm [VALOR], conforme acordado entre as partes.</p></div>
<div class="clause"><p class="clause-title">Cláusula 5ª – Das condições gerais</p><p class="clause-text">Quaisquer alterações no presente contrato somente produzirão efeitos se formalizadas por escrito e assinadas pelas partes.</p></div>
<div class="signatures">
  <div class="sign-line"><p class="sign-label">Assinatura do profissional:</p><div class="sign-value">[ASSINATURA_PROFISSIONAL]</div></div>
  <p class="sign-label">Local: Joinville - SC — Data: [DATA]</p>
  <div class="sign-line" style="margin-top: 1rem;"><p class="sign-label">Responsável / Contratante</p><div class="sign-value">[ASSINATURA_RESPONSAVEL]</div></div>
</div>
</body>
</html>`,
}

export function ContractTemplates() {
  const [list, setList] = useState<api.ContractTemplateItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [presetFields, setPresetFields] = useState({ tipoServico: '' })
  const [submitting, setSubmitting] = useState(false)
  const [pdfLoading, setPdfLoading] = useState(false)
  const [previewLoadingId, setPreviewLoadingId] = useState<string | null>(null)
  const [formError, setFormError] = useState('')
  const { user } = useAuth()
  const [signatureImageData, setSignatureImageData] = useState<string | null>(null)
  const [signatureLoading, setSignatureLoading] = useState(false)
  const [signatureSaving, setSignatureSaving] = useState(false)
  const [signatureError, setSignatureError] = useState('')
  const signatureInputRef = useRef<HTMLInputElement>(null)

  const loadSignature = useCallback(() => {
    if (user?.role !== 'PROFESSIONAL') return
    setSignatureLoading(true)
    setSignatureError('')
    api
      .getMySignature()
      .then((r) => setSignatureImageData(r.signature_image_data || null))
      .catch(() => setSignatureError('Falha ao carregar assinatura.'))
      .finally(() => setSignatureLoading(false))
  }, [user?.role])

  useEffect(() => {
    loadSignature()
  }, [loadSignature])

  const handleSignatureFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !file.type.startsWith('image/')) return
    setSignatureError('')
    setSignatureSaving(true)
    const reader = new FileReader()
    reader.onload = () => {
      const dataUrl = reader.result as string
      api
        .updateMySignature(dataUrl)
        .then(() => {
          setSignatureImageData(dataUrl)
          signatureInputRef.current?.form?.reset()
        })
        .catch(() => setSignatureError('Falha ao enviar imagem.'))
        .finally(() => setSignatureSaving(false))
    }
    reader.readAsDataURL(file)
  }

  const handleRemoveSignature = () => {
    if (!window.confirm('Remover sua assinatura dos contratos?')) return
    setSignatureError('')
    setSignatureSaving(true)
    api
      .updateMySignature('')
      .then(() => setSignatureImageData(null))
      .catch(() => setSignatureError('Falha ao remover.'))
      .finally(() => setSignatureSaving(false))
  }

  const load = useCallback(() => {
    setLoading(true)
    api
      .listContractTemplates()
      .then((r) => setList(r.templates))
      .catch(() => setError('Falha ao carregar modelos de contrato.'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const openCreate = () => {
    setEditingId(null)
    setForm({ name: PRESET_PRESTACAO_SERVICOS.name, body_html: PRESET_PRESTACAO_SERVICOS.body_html })
    setPresetFields({ tipoServico: '' })
    setFormError('')
    setModalOpen(true)
  }

  const openEdit = (id: string) => {
    setEditingId(id)
    setPresetFields({ tipoServico: '' })
    setFormError('')
    setModalOpen(true)
    api
      .getContractTemplate(id)
      .then((t) => {
        setForm({ name: t.name, body_html: t.body_html })
        setPresetFields({ tipoServico: t.tipo_servico ?? '' })
      })
      .catch(() => setFormError('Falha ao carregar modelo.'))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    if (!form.name.trim()) {
      setFormError('Nome é obrigatório.')
      return
    }
    setSubmitting(true)
    try {
      const tipoServico = presetFields.tipoServico.trim()
      if (editingId) {
        const t = list.find((x) => x.id === editingId)
        await api.updateContractTemplate(editingId, form.name.trim(), form.body_html, t?.version ?? 1, tipoServico, '')
      } else {
        await api.createContractTemplate(form.name.trim(), form.body_html, tipoServico, '')
      }
      setModalOpen(false)
      load()
    } catch {
      setFormError('Falha ao salvar.')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm('Excluir este modelo de contrato?')) return
    try {
      await api.deleteContractTemplate(id)
      setError('')
      load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Falha ao excluir.'
      try {
        const parsed = JSON.parse(msg)
        if (parsed?.error) setError(parsed.error)
        else setError('Falha ao excluir.')
      } catch {
        setError(msg || 'Falha ao excluir.')
      }
    }
  }

  function getProcessedHtml(): string {
    let html = form.body_html
      .replace(/\[TIPO_SERVICO\]/g, presetFields.tipoServico.trim() || '[não informado]')
      .replace(/\[PERIODICIDADE\]/g, '[definido ao disparar o contrato]')
    const sigPlaceholder = '[ASSINATURA_PROFISSIONAL]'
    if (html.includes(sigPlaceholder)) {
      let sigHtml = ''
      if (signatureImageData) {
        sigHtml = `<img src="${signatureImageData}" alt="Assinatura do profissional" style="max-height:56px;max-width:200px;display:block;" />`
      } else if (user?.full_name) {
        sigHtml = `<span style="font-family: 'Brush Script MT', 'Segoe Script', 'Dancing Script', cursive; font-size: 1.25em;">${user.full_name}</span>`
      }
      html = html.replace(new RegExp(sigPlaceholder.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), sigHtml || sigPlaceholder)
    }
    return html
  }

  const handlePreviewPdf = () => {
    setFormError('')
    setPdfLoading(true)
    const processedHtml = getProcessedHtml()
    const iframe = document.createElement('iframe')
    iframe.setAttribute('style', 'position:absolute;width:210mm;height:297mm;left:-9999px;top:0;border:none;')
    document.body.appendChild(iframe)
    const doc = iframe.contentDocument
    if (!doc) {
      setPdfLoading(false)
      document.body.removeChild(iframe)
      setFormError('Não foi possível gerar o preview.')
      return
    }
    doc.open()
    doc.write(processedHtml)
    doc.close()
    iframe.onload = () => {
      setTimeout(() => {
        const body = iframe.contentDocument?.body
        if (!body) {
          setPdfLoading(false)
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
        html2pdf().set(opt).from(body).outputPdf('blob').then(
          (blob) => {
            const url = URL.createObjectURL(blob)
            window.open(url, '_blank', 'noopener,noreferrer')
            URL.revokeObjectURL(url)
            document.body.removeChild(iframe)
            setPdfLoading(false)
          },
          () => {
            if (document.body.contains(iframe)) document.body.removeChild(iframe)
            setPdfLoading(false)
            setFormError('Falha ao gerar o PDF. Tente novamente.')
          }
        )
      }, 100)
    }
  }

  const runPdfFromHtml = (html: string, onDone: () => void) => {
    const iframe = document.createElement('iframe')
    iframe.setAttribute('style', 'position:absolute;width:210mm;height:297mm;left:-9999px;top:0;border:none;')
    document.body.appendChild(iframe)
    const doc = iframe.contentDocument
    if (!doc) {
      onDone()
      document.body.removeChild(iframe)
      return
    }
    doc.open()
    doc.write(html)
    doc.close()
    iframe.onload = () => {
      setTimeout(() => {
        const body = iframe.contentDocument?.body
        if (!body) {
          onDone()
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
        html2pdf().set(opt).from(body).outputPdf('blob').then(
          (blob) => {
            const url = URL.createObjectURL(blob)
            window.open(url, '_blank', 'noopener,noreferrer')
            URL.revokeObjectURL(url)
            document.body.removeChild(iframe)
            onDone()
          },
          () => {
            if (document.body.contains(iframe)) document.body.removeChild(iframe)
            onDone()
          }
        )
      }, 100)
    }
  }

  const handlePreviewPdfFromList = (templateId: string) => {
    setPreviewLoadingId(templateId)
    api.getContractTemplate(templateId).then(
      (t) => runPdfFromHtml(t.body_html, () => setPreviewLoadingId(null)),
      () => setPreviewLoadingId(null)
    )
  }

  return (
    <PageContainer maxWidth="md">
      <Box sx={{ mb: 1 }}>
        <Typography component={Link} to="/" sx={{ color: 'primary.main', textDecoration: 'none', fontSize: 14 }}>← Início</Typography>
      </Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 1, mb: 2 }}>
        <Typography variant="h4" fontWeight={700}>Modelos de contrato</Typography>
        <Button variant="contained" onClick={openCreate}>Novo modelo</Button>
      </Box>
      <Typography color="text.secondary" sx={{ mb: 2, fontSize: 14 }}>
        Cada profissional pode ter seus próprios modelos. Use-os ao disparar contratos para assinatura na tela do paciente.
      </Typography>
      {user?.role === 'PROFESSIONAL' && (
        <Paper variant="outlined" sx={{ p: 2, mb: 2, bgcolor: 'grey.50' }}>
          <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>Minha assinatura</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>A imagem será exibida nos contratos enviados por você. Use PNG ou JPEG (até 500 KB).</Typography>
          {signatureLoading && <Typography variant="body2" color="text.secondary">Carregando...</Typography>}
          {!signatureLoading && (
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, alignItems: 'center' }}>
              {signatureImageData && (
                <Box sx={{ mb: 1 }}>
                  <Box component="img" src={signatureImageData} alt="Sua assinatura" sx={{ maxHeight: 56, maxWidth: 200, border: '1px solid', borderColor: 'divider', borderRadius: 0.5 }} />
                </Box>
              )}
              <Button variant="outlined" component="label" disabled={signatureSaving} size="small">
                <input ref={signatureInputRef} type="file" accept="image/*" onChange={handleSignatureFile} hidden />
                {signatureSaving ? 'Enviando...' : 'Escolher imagem da assinatura'}
              </Button>
              {signatureImageData && (
                <Button variant="outlined" color="error" size="small" disabled={signatureSaving} onClick={handleRemoveSignature}>Remover assinatura</Button>
              )}
            </Box>
          )}
          {signatureError && <Alert severity="error" sx={{ mt: 0.5 }}>{signatureError}</Alert>}
        </Paper>
      )}
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {loading && <Typography color="text.secondary">Carregando...</Typography>}
      {!loading && (
        <List disablePadding>
          {list.map((t) => (
            <ListItem key={t.id} component={Paper} variant="outlined" sx={{ mb: 0.75, borderRadius: 1 }}>
              <ListItemText primary={t.name} secondary={`v${t.version}`} primaryTypographyProps={{ fontWeight: 600 }} />
              <ListItemSecondaryAction>
                <Button size="small" variant="contained" startIcon={<PreviewIcon />} disabled={previewLoadingId === t.id} onClick={() => handlePreviewPdfFromList(t.id)} sx={{ mr: 0.5 }}>
                  {previewLoadingId === t.id ? 'Gerando...' : 'Preview'}
                </Button>
                <Button size="small" variant="outlined" startIcon={<EditIcon />} onClick={() => openEdit(t.id)} sx={{ mr: 0.5 }}>Editar</Button>
                <Button size="small" variant="outlined" color="error" startIcon={<DeleteOutlineIcon />} onClick={() => handleDelete(t.id)}>Excluir</Button>
              </ListItemSecondaryAction>
            </ListItem>
          ))}
          {list.length === 0 && (
            <Paper variant="outlined" sx={{ p: 2, textAlign: 'center', borderStyle: 'dashed' }}>
              <Typography color="text.secondary">Nenhum modelo cadastrado. Crie um para enviar contratos aos responsáveis.</Typography>
            </Paper>
          )}
        </List>
      )}

      <AppDialog open={modalOpen} onClose={() => setModalOpen(false)} title={editingId ? 'Editar modelo' : 'Novo modelo'} maxWidth="md">
        <Box component="form" onSubmit={handleSubmit}>
          <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ display: 'block', mb: 0.75, textTransform: 'uppercase' }}>Dados do modelo</Typography>
          <TextField label="Nome do modelo" fullWidth value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} placeholder="Ex.: Contrato de Prestação de Serviços" sx={{ mb: 1 }} />
          <TextField label="Tipo de serviço" fullWidth value={presetFields.tipoServico} onChange={(e) => setPresetFields((p) => ({ ...p, tipoServico: e.target.value }))} placeholder="Ex.: Fonoaudiologia" sx={{ mb: 1.5 }} />
          <Typography variant="body2" color="text.secondary" sx={{ mb: 0.75 }}>Periodicidade, datas e valor são definidos na tela de disparar o contrato (por cliente).</Typography>
          <Typography variant="caption" color="text.secondary" fontWeight={600} sx={{ display: 'block', mb: 0.75, textTransform: 'uppercase' }}>Conteúdo (HTML)</Typography>
          <TextField label="HTML do contrato" fullWidth multiline rows={14} value={form.body_html} onChange={(e) => setForm((f) => ({ ...f, body_html: e.target.value }))} sx={{ fontFamily: 'monospace', fontSize: 13, mb: 0.5 }} />
          <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mb: 1 }}>Placeholders: [CONTRATADO], [CONTRATANTE], [PACIENTE_NOME], [TIPO_SERVICO], [PERIODICIDADE], [VALOR], [ASSINATURA_PROFISSIONAL], [ASSINATURA_RESPONSAVEL], [DATA]. Escreva o local diretamente no texto (ex.: Joinville - SC). Apenas [DATA] é preenchida automaticamente na tela de assinatura (dia em que o responsável abriu o link, DD/MM/AAAA).</Typography>
          {formError && <Alert severity="error" sx={{ mb: 1 }}>{formError}</Alert>}
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
            <Button variant="outlined" onClick={handlePreviewPdf} disabled={pdfLoading}>{pdfLoading ? 'Gerando PDF...' : 'Preview em PDF'}</Button>
            <Button type="submit" variant="contained" color="primary" disabled={submitting}>{submitting ? 'Salvando...' : 'Salvar'}</Button>
            <Button type="button" onClick={() => setModalOpen(false)}>Cancelar</Button>
          </Box>
        </Box>
      </AppDialog>
    </PageContainer>
  )
}
