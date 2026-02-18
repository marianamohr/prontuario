import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  Chip,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'

function parseMessageHuman(msg: string | null | undefined): string {
  const m = String(msg || '').trim()
  if (!m) return 'Erro sem mensagem.'
  // backend muitas vezes retorna JSON {"error":"...","detail":"..."}
  if (m.startsWith('{') && m.endsWith('}')) {
    try {
      const o = JSON.parse(m)
      if (o?.detail && o?.error) return `${String(o.error)} (${String(o.detail)})`
      if (o?.error) return String(o.error)
    } catch {
      // ignore
    }
  }
  return m
}

/** Extrai o primeiro path de arquivo do repositório no stack (ex.: src/pages/BackofficeAudit.tsx). */
function extractFileFromStack(stack: string | null | undefined): string | null {
  if (!stack || typeof stack !== 'string') return null
  // Ex.: "at Component (webpack:///./src/pages/BackofficeAudit.tsx:45:12)" ou "http://.../src/pages/BackofficeAudit.tsx:45:12"
  const match = stack.match(/src\/[^)\s]+\.(tsx?|jsx?|js)(?::\d+:\d+)?/)
  if (!match) return null
  return match[0].replace(/:\d+:\d+$/, '')
}

function describeError(it: api.BackofficeErrorItem): string {
  const kind = String(it.kind || '').toUpperCase()
  const method = it.http_method ? String(it.http_method) : ''
  const path = it.path ? String(it.path) : ''
  const msg = parseMessageHuman(it.message)

  if (kind === 'FETCH_ERROR') {
    return `Falha de rede ao chamar ${method || 'HTTP'} ${path || '(endpoint desconhecido)'}: ${msg}`
  }
  if (kind === 'HTTP_ERROR') {
    const status = (it.metadata?.status ?? undefined) as number | undefined
    const st = status ? `HTTP ${status}` : 'HTTP error'
    return `${st} em ${method || 'HTTP'} ${path || '(endpoint desconhecido)'}: ${msg}`
  }
  if (kind === 'JSON_PARSE_ERROR') {
    return `Resposta inválida (JSON) em ${method || 'HTTP'} ${path || '(endpoint desconhecido)'}: ${msg}`
  }
  if (kind === 'REACT_ERROR_BOUNDARY') {
    return `Erro de renderização (React): ${msg}`
  }
  if (kind === 'WINDOW_ERROR') {
    return `Erro global do navegador: ${msg}`
  }
  if (kind === 'UNHANDLED_REJECTION') {
    return `Promise rejeitada sem tratamento: ${msg}`
  }
  if (it.pg_message) {
    return `Erro no banco: ${String(it.pg_message)}`
  }
  return `${kind || 'Erro'}: ${msg}`
}

export function BackofficeErrors() {
  const [searchParams] = useSearchParams()
  const requestIdFromURL = searchParams.get('request_id') || ''

  const [items, setItems] = useState<api.BackofficeErrorItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [offset, setOffset] = useState(0)

  const [requestId, setRequestId] = useState(requestIdFromURL)
  const [severity, setSeverity] = useState('')

  useEffect(() => {
    setRequestId(requestIdFromURL)
  }, [requestIdFromURL])

  const canLoadMore = useMemo(() => items.length > 0 && !loading, [items.length, loading])

  const load = useCallback((reset: boolean) => {
    setLoading(true)
    setError('')
    const nextOffset = reset ? 0 : offset
    api.listBackofficeErrors({
      limit: 50,
      offset: nextOffset,
      request_id: requestId.trim() || undefined,
      severity: severity || undefined,
    })
      .then((r) => {
        const list = r?.items ?? []
        setItems(reset ? list : [...items, ...list])
        setOffset(nextOffset + list.length)
      })
      .catch((e) => setError((e as Error)?.message || 'Falha ao carregar erros.'))
      .finally(() => setLoading(false))
  }, [offset, requestId, severity, items])

  useEffect(() => {
    load(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <PageContainer>
      <Typography variant="h4" sx={{ mb: 1.5 }}>Erros (bugs)</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Lista de erros do frontend e backend (sem PII). Clique no Request ID para ver a Auditoria correlacionada.
      </Typography>

      <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 2 }}>
        <TextField
          label="Request ID (opcional)"
          value={requestId}
          onChange={(e) => setRequestId(e.target.value)}
          size="small"
          sx={{ minWidth: 280 }}
        />
        <FormControl size="small" sx={{ minWidth: 180 }}>
          <InputLabel>Severidade</InputLabel>
          <Select value={severity} label="Severidade" onChange={(e) => setSeverity(String(e.target.value))}>
            <MenuItem value="">Todas</MenuItem>
            <MenuItem value="WARN">WARN</MenuItem>
            <MenuItem value="ERROR">ERROR</MenuItem>
          </Select>
        </FormControl>
        <Button variant="contained" onClick={() => load(true)} disabled={loading}>
          {loading ? 'Carregando...' : 'Buscar'}
        </Button>
        <Button variant="outlined" onClick={() => load(false)} disabled={!canLoadMore}>
          Carregar mais
        </Button>
      </Box>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <TableContainer component={Paper} variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Data/hora</TableCell>
              <TableCell>Sev</TableCell>
              <TableCell>Origem</TableCell>
              <TableCell>Path (página / endpoint)</TableCell>
              <TableCell>Descrição</TableCell>
              <TableCell>Request</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {items.map((it) => {
              const pageOrEndpoint = it.path?.trim() || null
              const fileInRepo = extractFileFromStack(it.stack ?? undefined)
              return (
              <TableRow key={it.id} hover>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{new Date(it.created_at).toLocaleString('pt-BR')}</TableCell>
                <TableCell><Chip size="small" label={it.severity} /></TableCell>
                <TableCell>{it.source}</TableCell>
                <TableCell sx={{ fontFamily: pageOrEndpoint?.startsWith('/') ? 'inherit' : 'monospace', fontSize: 12, maxWidth: 320 }}>
                  {pageOrEndpoint ? (
                    <Box component="span" title={pageOrEndpoint}>
                      {it.http_method ? `${it.http_method} ` : ''}{pageOrEndpoint}
                    </Box>
                  ) : fileInRepo ? (
                    <Box component="span" title="Arquivo no repositório (do stack)">
                      {fileInRepo}
                    </Box>
                  ) : (
                    '—'
                  )}
                </TableCell>
                <TableCell sx={{ maxWidth: 760 }}>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    {describeError(it)}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {it.action_name ? `ação: ${it.action_name}` : ''}
                    {it.kind ? ` • kind: ${it.kind}` : ''}
                  </Typography>
                </TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                  {it.request_id ? (
                    <Link to={`/backoffice/audit?request_id=${encodeURIComponent(it.request_id)}`} style={{ color: 'inherit' }}>
                      {it.request_id}
                    </Link>
                  ) : (
                    '—'
                  )}
                </TableCell>
              </TableRow>
            )})}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={6}>
                  <Typography color="text.secondary">Nenhum erro encontrado.</Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </PageContainer>
  )
}

