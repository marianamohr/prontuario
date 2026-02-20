import { useCallback, useEffect, useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import { Box, Typography, Button, Paper, CircularProgress } from '@mui/material'
import { useAuth } from '../contexts/AuthContext'
import { useBranding } from '../contexts/BrandingContext'
import { PageContainer } from '../components/ui/PageContainer'
import * as api from '../lib/api'
import { toBrazilYYYYMMDD } from '../lib/date'

function formatTime(t: string): string {
  if (!t) return '—'
  const [h, m] = t.split(':')
  return `${h ?? '00'}:${m ?? '00'}`
}

export function Home() {
  const { user, isImpersonated } = useAuth()
  const branding = useBranding()?.branding ?? null
  const isProfessional = user?.role === 'PROFESSIONAL'
  const primaryColor = isProfessional && branding?.primary_color ? branding?.primary_color : undefined

  const [pendingContracts, setPendingContracts] = useState<api.PendingContractItem[]>([])
  const [todayAppointments, setTodayAppointments] = useState<api.AppointmentItem[]>([])
  const [loadingContracts, setLoadingContracts] = useState(true)
  const [loadingAgenda, setLoadingAgenda] = useState(true)

  const load = useCallback(() => {
    if (user?.role !== 'PROFESSIONAL' && user?.role !== 'SUPER_ADMIN') {
      setLoadingContracts(false)
      setLoadingAgenda(false)
      return
    }
    setLoadingContracts(true)
    api.listPendingContracts()
      .then((r) => setPendingContracts(r.contracts || []))
      .catch(() => setPendingContracts([]))
      .finally(() => setLoadingContracts(false))
    const today = toBrazilYYYYMMDD(new Date())
    setLoadingAgenda(true)
    api.listAppointments(today, today)
      .then((r) => setTodayAppointments(r.appointments || []))
      .catch(() => setTodayAppointments([]))
      .finally(() => setLoadingAgenda(false))
  }, [user?.role])

  useEffect(() => {
    load()
  }, [load])

  const canShowContent = user?.role === 'PROFESSIONAL' || user?.role === 'SUPER_ADMIN'

  if (user?.role === 'SUPER_ADMIN' && !isImpersonated) {
    return <Navigate to="/backoffice/audit" replace />
  }

  return (
    <PageContainer>
      <Typography variant="h4" sx={{ mb: 2, color: primaryColor }}>
        {isProfessional && branding?.home_label ? `Bem-vindo ao ${branding.home_label}` : 'Bem-vindo ao Prontuário Saúde'}
      </Typography>

      {canShowContent && (
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2 }}>
          {/* Esquerda: contratos que faltam assinar */}
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="h6" sx={{ mb: 1.5 }}>Contratos pendentes de assinatura</Typography>
            {loadingContracts ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
                <CircularProgress size={24} />
              </Box>
            ) : pendingContracts.length === 0 ? (
              <Typography variant="body2" color="text.secondary">Nenhum contrato pendente.</Typography>
            ) : (
              <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
                {pendingContracts.map((c) => (
                  <Box key={c.id} component="li" sx={{ py: 0.75, borderBottom: 1, borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                    <Typography variant="body2"><strong>{c.template_name}</strong> — {c.patient_name} ({c.guardian_name})</Typography>
                    <Button component={Link} to={`/patients/${c.patient_id}/contracts`} size="small" variant="outlined" sx={{ mt: 0.5 }}>Ver contratos</Button>
                  </Box>
                ))}
              </Box>
            )}
          </Paper>

          {/* Direita: agenda do dia */}
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="h6" sx={{ mb: 1.5 }}>Agenda do dia</Typography>
            {loadingAgenda ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
                <CircularProgress size={24} />
              </Box>
            ) : todayAppointments.length === 0 ? (
              <Typography variant="body2" color="text.secondary">Nenhum agendamento para hoje.</Typography>
            ) : (
              <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
                {todayAppointments
                  .filter((a) => a.status !== 'CANCELLED')
                  .sort((a, b) => (a.start_time || '').localeCompare(b.start_time || ''))
                  .map((a) => (
                    <Box key={a.id} component="li" sx={{ py: 0.75, borderBottom: 1, borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                      <Typography variant="body2">
                        <strong>{formatTime(a.start_time)}</strong> — {a.patient_name || 'Paciente'}
                        {a.status && a.status !== 'CONFIRMED' && (
                          <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>
                            ({a.status === 'COMPLETED' ? 'Concluído' : a.status})
                          </Typography>
                        )}
                      </Typography>
                    </Box>
                  ))}
              </Box>
            )}
            <Button component={Link} to="/agenda" variant="outlined" size="small" sx={{ mt: 1.5 }}>Ver agenda completa</Button>
          </Paper>
        </Box>
      )}

      {!canShowContent && (
        <Typography color="text.secondary">Acesse o menu para navegar.</Typography>
      )}
    </PageContainer>
  )
}
