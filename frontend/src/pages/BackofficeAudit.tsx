import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
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

export function BackofficeAudit() {
  const [searchParams] = useSearchParams()
  const requestIdFromURL = searchParams.get('request_id') || ''

  const [items, setItems] = useState<api.BackofficeTimelineItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [offset, setOffset] = useState(0)

  const [requestId, setRequestId] = useState(requestIdFromURL)
  const [severity, setSeverity] = useState('')

  useEffect(() => {
    setRequestId(requestIdFromURL)
    // quando vier request_id na URL, já busca automaticamente
    if (requestIdFromURL) {
      setOffset(0)
      setItems([])
      // não await: useCallback abaixo fará o fetch
    }
  }, [requestIdFromURL])

  const canLoadMore = useMemo(() => items.length > 0 && !loading, [items.length, loading])

  const load = useCallback((reset: boolean) => {
    setLoading(true)
    setError('')
    const nextOffset = reset ? 0 : offset
    api.listBackofficeTimeline({
      limit: 50,
      offset: nextOffset,
      request_id: requestId.trim() || undefined,
      severity: severity || undefined,
    })
      .then((r) => {
        setItems(reset ? r.items : [...items, ...r.items])
        setOffset(nextOffset + r.items.length)
      })
      .catch((e) => setError((e as Error)?.message || 'Falha ao carregar auditoria.'))
      .finally(() => setLoading(false))
  }, [offset, requestId, severity, items])

  // auto-load quando entrar na página
  useEffect(() => {
    load(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <PageContainer>
      <Typography variant="h4" sx={{ mb: 1.5 }}>Auditoria</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Linha do tempo unificada (eventos + acessos). Sem PII; use o Request ID para correlacionar com erros.
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
            <MenuItem value="INFO">INFO</MenuItem>
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
              <TableCell>Tipo</TableCell>
              <TableCell>Ação</TableCell>
              <TableCell>Ator</TableCell>
              <TableCell>Recurso</TableCell>
              <TableCell>Request</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {items.map((it) => (
              <TableRow key={`${it.kind}-${it.id}`} hover>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{new Date(it.created_at).toLocaleString('pt-BR')}</TableCell>
                <TableCell><Chip size="small" label={it.kind} /></TableCell>
                <TableCell>
                  <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', alignItems: 'center' }}>
                    <Chip size="small" label={it.severity} />
                    {it.is_impersonated && <Chip size="small" color="warning" label="impersonate" />}
                    <span>{it.action}</span>
                  </Box>
                </TableCell>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{it.actor_type}{it.actor_id ? `/${it.actor_id}` : ''}</TableCell>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>
                  {it.resource_type ? `${it.resource_type}${it.resource_id ? `/${it.resource_id}` : ''}` : '—'}
                </TableCell>
                <TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>{it.request_id || '—'}</TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={6}>
                  <Typography color="text.secondary">Nenhum item encontrado.</Typography>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </PageContainer>
  )
}

