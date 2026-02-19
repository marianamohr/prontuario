const BASE = (import.meta.env.VITE_API_URL || '').replace(/\/$/, '')

function getToken(): string | null {
  return localStorage.getItem('token')
}

function getRequestId(): string {
  try {
    return (globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random()}`).toString()
  } catch {
    return `${Date.now()}-${Math.random()}`
  }
}

type FrontendErrorEvent = {
  request_id?: string
  severity: 'WARN' | 'ERROR'
  kind: string
  message: string
  stack?: string
  http_method?: string
  path?: string
  status?: number
  action_name?: string
  metadata?: Record<string, unknown>
}

async function postFrontendError(ev: FrontendErrorEvent) {
  // Evita recursão (logar o logger).
  if (ev.path?.includes('/api/errors/frontend')) return
  try {
    const headers: HeadersInit = { 'Content-Type': 'application/json' }
    const token = getToken()
    if (token) headers['Authorization'] = `Bearer ${token}`
    if (ev.request_id) headers['X-Request-ID'] = ev.request_id
    const url = `${BASE}/api/errors/frontend`
    // Fire-and-forget
    fetch(url, { method: 'POST', headers, body: JSON.stringify(ev) }).catch(() => {})
  } catch {
    // no-op
  }
}

export async function api<T>(
  path: string,
  opts: RequestInit & { json?: unknown } = {}
): Promise<T> {
  const { json, ...init } = opts
  const headers: HeadersInit = {
    ...(init.headers as Record<string, string>),
  }
  if (json !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  const requestId = (headers as Record<string, string>)['X-Request-ID'] || getRequestId()
  headers['X-Request-ID'] = requestId

  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const url = path.startsWith('http') ? path : `${BASE}${path}`
  let res: Response
  try {
    res = await fetch(url, {
      ...init,
      headers,
      body: json !== undefined ? JSON.stringify(json) : init.body,
    })
  } catch (e: unknown) {
    const err = e instanceof Error ? e : new Error('network error')
    postFrontendError({
      request_id: requestId,
      severity: 'ERROR',
      kind: 'FETCH_ERROR',
      message: err.message || 'fetch failed',
      stack: err.stack,
      http_method: String(init.method || 'GET'),
      path,
    })
    throw err
  }
  if (!res.ok) {
    const t = await res.text()
    const err: Error & { status?: number } = new Error(t || res.statusText)
    err.status = res.status
    const severity: 'WARN' | 'ERROR' = res.status >= 500 ? 'ERROR' : 'WARN'
    postFrontendError({
      request_id: requestId,
      severity,
      kind: 'HTTP_ERROR',
      message: t || res.statusText || 'http error',
      http_method: String(init.method || 'GET'),
      path,
      status: res.status,
    })
    throw err
  }
  const contentType = res.headers.get('content-type')
  if (contentType?.includes('application/json')) {
    try {
      return (await res.json()) as T
    } catch (e: unknown) {
      const err = e instanceof Error ? e : new Error('json parse error')
      postFrontendError({
        request_id: requestId,
        severity: 'ERROR',
        kind: 'JSON_PARSE_ERROR',
        message: err.message || 'json parse error',
        stack: err.stack,
        http_method: String(init.method || 'GET'),
        path,
      })
      throw err
    }
  }
  return undefined as unknown as T
}

export type Address = {
  street: string
  number?: string
  complement?: string
  neighborhood: string
  city: string
  state: string
  country: string
  zip: string
}

export type User = {
  id: string
  email?: string
  full_name?: string
  role: string
  clinic_id?: string
}

export type LoginRes = {
  token: string
  expires_at: string
  user: User
}

export function login(email: string, password: string) {
  return api<LoginRes>('/api/auth/login', {
    method: 'POST',
    json: { email, password },
  })
}

export function loginProfessional(email: string, password: string) {
  return login(email, password)
}

export function loginSuperAdmin(email: string, password: string) {
  return login(email, password)
}

export function forgotPassword(email: string) {
  return api<{ message: string }>('/api/auth/password/forgot', {
    method: 'POST',
    json: { email },
  })
}

export function resetPassword(token: string, new_password: string) {
  return api<{ message: string }>('/api/auth/password/reset', {
    method: 'POST',
    json: { token, new_password },
  })
}

export function me() {
  return api<User>('/api/me')
}

export function getMySignature() {
  return api<{ signature_image_data: string }>('/api/me/signature')
}

export function updateMySignature(signature_image_data: string) {
  return api<{ message: string }>('/api/me/signature', {
    method: 'PUT',
    json: { signature_image_data },
  })
}

export type Branding = {
  primary_color?: string | null
  background_color?: string | null
  home_label?: string | null
  home_image_url?: string | null
  action_button_color?: string | null   // botões de aceitar / submeter / ação principal
  negation_button_color?: string | null // botões de excluir / cancelar
}

export function getBranding() {
  return api<Branding>('/api/me/branding')
}

export function updateBranding(payload: Branding) {
  return api<{ message: string }>('/api/me/branding', {
    method: 'PUT',
    json: payload,
  })
}

export type MyProfile = {
  id: string
  email: string
  full_name: string
  trade_name?: string | null
  birth_date?: string | null
  address?: Address | null
  marital_status?: string | null
}

export function getMyProfile() {
  return api<MyProfile>('/api/me/profile')
}

export function patchMyProfile(payload: {
  full_name: string
  trade_name?: string
  birth_date?: string
  address?: Address
  marital_status?: string
}) {
  return api<{ message: string }>('/api/me/profile', { method: 'PATCH', json: payload })
}

export type ScheduleDay = {
  day_of_week: number
  enabled: boolean
  start_time: string | null
  end_time: string | null
  consultation_duration_minutes: number
  interval_minutes: number
  lunch_start: string | null
  lunch_end: string | null
}

export function getScheduleConfig() {
  return api<{ days: ScheduleDay[] }>('/api/me/schedule-config')
}

export function putScheduleConfig(days: { day_of_week: number; enabled?: boolean; start_time?: string | null; end_time?: string | null; consultation_duration_minutes?: number; interval_minutes?: number; lunch_start?: string | null; lunch_end?: string | null }[]) {
  return api<{ message: string }>('/api/me/schedule-config', { method: 'PUT', json: { days } })
}

export function copyScheduleConfigDay(from_day: number, to_day: number) {
  return api<{ message: string }>('/api/me/schedule-config/copy', { method: 'POST', json: { from_day, to_day } })
}

export type AppointmentItem = {
  id: string
  patient_id: string
  patient_name?: string
  contract_id: string
  appointment_date: string
  start_time: string
  end_time: string
  status: string
  notes: string
}

export function listAppointments(from: string, to: string) {
  return api<{ appointments: AppointmentItem[] }>(`/api/appointments?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`)
}

export type ContractForAgendaItem = { id: string; patient_id: string; patient_name: string; template_name: string }

export function listContractsForAgenda() {
  return api<{ contracts: ContractForAgendaItem[] }>('/api/contracts/for-agenda')
}

export type PendingContractItem = { id: string; patient_id: string; patient_name: string; template_name: string; guardian_name: string }

export function listPendingContracts() {
  return api<{ contracts: PendingContractItem[] }>('/api/contracts/pending')
}

export function createAppointments(contract_id: string, slots: { appointment_date: string; start_time: string }[]) {
  return api<{ message: string; created: number }>('/api/appointments', {
    method: 'POST',
    json: { contract_id, slots },
  })
}

export function patchAppointment(id: string, payload: { appointment_date?: string; start_time?: string; end_time?: string; status?: string; notes?: string }) {
  return api<{ message: string }>(`/api/appointments/${id}`, { method: 'PATCH', json: payload })
}

export function endContract(patientId: string, contractId: string, end_date: string) {
  return api<{ message: string }>(`/api/patients/${patientId}/contracts/${contractId}/end`, { method: 'PUT', json: { end_date } })
}

export type ListPatientsRes = {
  patients: { id: string; full_name: string; birth_date?: string }[]
  limit: number
  offset: number
  total: number
}

export function listPatients(opts?: { limit?: number; offset?: number }) {
  const params = new URLSearchParams()
  if (opts?.limit != null) params.set('limit', String(opts.limit))
  if (opts?.offset != null) params.set('offset', String(opts.offset))
  const q = params.toString() ? `?${params.toString()}` : ''
  return api<ListPatientsRes>(`/api/patients${q}`)
}

export type PatientDetail = {
  id: string
  full_name: string
  birth_date?: string | null
  email?: string | null
  cpf?: string | null
  guardian?: {
    id: string
    full_name: string
    email: string
    cpf?: string | null
    address?: Address | null
    birth_date?: string | null
    phone?: string | null
  }
}

export function getPatient(patientId: string) {
  return api<PatientDetail>(`/api/patients/${patientId}`)
}

export type UpdatePatientPayload = {
  full_name?: string
  birth_date?: string
  email?: string
  patient_cpf?: string
  patient_address?: Address
  guardian_full_name?: string
  guardian_email?: string
  guardian_address?: Address
  guardian_birth_date?: string
  guardian_phone?: string
  guardian_cpf?: string
}

export function updatePatient(patientId: string, payload: UpdatePatientPayload) {
  return api<{ message: string }>(`/api/patients/${patientId}`, { method: 'PATCH', json: payload })
}

export function softDeletePatient(patientId: string) {
  return api<{ message: string }>(`/api/patients/${patientId}`, { method: 'DELETE' })
}

export function softDeleteGuardian(patientId: string, guardianId: string) {
  return api<{ message: string }>(`/api/patients/${patientId}/guardians/${guardianId}`, { method: 'DELETE' })
}

export function softDeleteContract(patientId: string, contractId: string) {
  return api<{ message: string }>(`/api/patients/${patientId}/contracts/${contractId}`, { method: 'DELETE' })
}

export type CreatePatientPayload = {
  full_name?: string
  birth_date?: string
  patient_cpf?: string
  patient_address?: Address
  same_person?: boolean
  guardian_full_name?: string
  guardian_email?: string
  guardian_cpf?: string
  guardian_address?: Address
  guardian_birth_date?: string
  guardian_phone?: string
  patient_full_name?: string
}

export function createPatient(payload: CreatePatientPayload) {
  return api<{ id: string }>('/api/patients', {
    method: 'POST',
    json: payload,
  })
}

export type GuardianInfo = { id: string; full_name: string; email: string; phone?: string | null; relation: string }

export function listPatientGuardians(patientId: string) {
  return api<{ guardians: GuardianInfo[] }>(`/api/patients/${patientId}/guardians`)
}

export type ScheduleRule = { day_of_week: number; slot_time: string }

export function sendContractForPatient(
  patientId: string,
  guardian_id: string,
  template_id: string,
  data_inicio?: string,
  data_fim?: string,
  valor?: string,
  periodicidade?: string,
  schedule_rules?: ScheduleRule[],
  sign_place?: string,
  sign_date?: string,
  num_appointments?: number
) {
  const body: {
    guardian_id: string
    template_id: string
    data_inicio?: string
    data_fim?: string
    valor?: string
    periodicidade?: string
    schedule_rules?: ScheduleRule[]
    sign_place?: string
    sign_date?: string
    num_appointments?: number
  } = { guardian_id, template_id }
  if (data_inicio) body.data_inicio = data_inicio
  if (data_fim) body.data_fim = data_fim
  if (valor !== undefined && valor !== '') body.valor = valor
  if (periodicidade !== undefined && periodicidade !== '') body.periodicidade = periodicidade
  if (schedule_rules && schedule_rules.length > 0) body.schedule_rules = schedule_rules
  if (sign_place !== undefined && sign_place.trim() !== '') body.sign_place = sign_place.trim()
  if (sign_date !== undefined && sign_date.trim() !== '') body.sign_date = sign_date.trim()
  if (num_appointments !== undefined && num_appointments > 0) body.num_appointments = num_appointments
  return api<{ message: string; contract_id: string }>(`/api/patients/${patientId}/send-contract`, {
    method: 'POST',
    json: body,
  })
}

export function getContractPreview(
  patientId: string,
  guardian_id: string,
  template_id: string,
  data_inicio?: string,
  data_fim?: string,
  valor?: string,
  periodicidade?: string
) {
  const params = new URLSearchParams({ guardian_id, template_id })
  if (data_inicio) params.set('data_inicio', data_inicio)
  if (data_fim) params.set('data_fim', data_fim)
  if (valor !== undefined && valor !== '') params.set('valor', valor)
  if (periodicidade !== undefined && periodicidade !== '') params.set('periodicidade', periodicidade)
  return api<{ body_html: string }>(`/api/patients/${patientId}/contract-preview?${params.toString()}`)
}

export type PatientContractItem = {
  id: string
  legal_guardian_id: string
  guardian_name: string
  guardian_email: string
  template_name: string
  status: string
  signed_at?: string
  verify_url?: string
}

export type ListPatientContractsRes = {
  contracts: PatientContractItem[]
  limit: number
  offset: number
  total: number
}

export function listPatientContracts(patientId: string, opts?: { limit?: number; offset?: number }) {
  const params = new URLSearchParams()
  if (opts?.limit != null) params.set('limit', String(opts.limit))
  if (opts?.offset != null) params.set('offset', String(opts.offset))
  const q = params.toString() ? `?${params.toString()}` : ''
  return api<ListPatientContractsRes>(`/api/patients/${patientId}/contracts${q}`)
}

export function resendPatientContract(patientId: string, contractId: string) {
  return api<{ message: string }>(`/api/patients/${patientId}/contracts/${contractId}/resend`, {
    method: 'POST',
  })
}

export function cancelPatientContract(patientId: string, contractId: string) {
  return api<{ message: string }>(`/api/patients/${patientId}/contracts/${contractId}/cancel`, {
    method: 'POST',
  })
}

export type ContractTemplateItem = { id: string; name: string; version: number }

export function listContractTemplates() {
  return api<{ templates: ContractTemplateItem[] }>('/api/contract-templates')
}

export function getContractTemplate(id: string) {
  return api<{ id: string; name: string; body_html: string; version: number; tipo_servico?: string; periodicidade?: string }>(`/api/contract-templates/${id}`)
}

export function createContractTemplate(name: string, body_html: string, tipo_servico?: string, periodicidade?: string) {
  return api<{ id: string }>('/api/contract-templates', {
    method: 'POST',
    json: { name, body_html, tipo_servico: tipo_servico ?? '', periodicidade: periodicidade ?? '' },
  })
}

export function updateContractTemplate(id: string, name: string, body_html: string, version: number, tipo_servico?: string, periodicidade?: string) {
  return api<{ ok?: boolean }>(`/api/contract-templates/${id}`, {
    method: 'PUT',
    json: { name, body_html, version, tipo_servico: tipo_servico ?? '', periodicidade: periodicidade ?? '' },
  })
}

export function deleteContractTemplate(id: string) {
  return api<{ message: string }>(`/api/contract-templates/${id}`, { method: 'DELETE' })
}

export type ListBackofficeUsersRes = {
  users: { type: string; id: string; email: string; full_name: string; clinic_id?: string; status: string }[]
  limit: number
  offset: number
  total: number
}

export function listBackofficeUsers(clinicId?: string, opts?: { limit?: number; offset?: number }) {
  const params = new URLSearchParams()
  if (clinicId) params.set('clinic_id', clinicId)
  if (opts?.limit != null) params.set('limit', String(opts.limit))
  if (opts?.offset != null) params.set('offset', String(opts.offset))
  const q = params.toString() ? `?${params.toString()}` : ''
  return api<ListBackofficeUsersRes>(`/api/backoffice/users${q}`)
}

export function getBackofficeProfessionalRelated(professionalId: string) {
  return api<{
    professional_id: string
    clinic_id: string
    patients: { id: string; full_name: string; birth_date?: string | null }[]
    guardians: { id: string; full_name: string; email: string; status: string; patients_count: number }[]
  }>(`/api/backoffice/professionals/${encodeURIComponent(professionalId)}/related`)
}

export type BackofficeUserDetail = {
  type: string
  id: string
  email: string
  full_name: string
  trade_name?: string | null
  status: string
  clinic_id?: string
  birth_date?: string | null
  address?: Address | null
  phone?: string | null
  marital_status?: string | null
  cpf?: string
  auth_provider?: string
  has_google_sub?: boolean
}

export function getBackofficeUser(type: string, id: string) {
  return api<{ user: BackofficeUserDetail }>(`/api/backoffice/users/${encodeURIComponent(type)}/${encodeURIComponent(id)}`)
}

export function patchBackofficeUser(type: string, id: string, payload: {
  email?: string
  full_name?: string
  trade_name?: string
  status?: string
  clinic_id?: string
  birth_date?: string
  address?: Address
  phone?: string
  marital_status?: string
  cpf?: string
  new_password?: string
}) {
  return api<{ message: string }>(`/api/backoffice/users/${encodeURIComponent(type)}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    json: payload,
  })
}

export function impersonateStart(target_user_type: string, target_user_id: string, reason: string) {
  return api<{ token: string; session_id: string; expires_in_seconds: number }>('/api/backoffice/impersonate/start', {
    method: 'POST',
    json: { target_user_type, target_user_id, reason },
  })
}

export function impersonateEnd() {
  return api<{ message: string }>('/api/backoffice/impersonate/end', { method: 'POST' })
}

export type BackofficeInviteItem = {
  id: string
  email: string
  full_name: string
  status: string
  expires_at: string
  created_at: string
}

export type ListInvitesRes = {
  items: BackofficeInviteItem[]
  limit: number
  offset: number
  total: number
}

export function listInvites(opts?: { limit?: number; offset?: number }) {
  const params = new URLSearchParams()
  if (opts?.limit != null) params.set('limit', String(opts.limit))
  if (opts?.offset != null) params.set('offset', String(opts.offset))
  const q = params.toString() ? `?${params.toString()}` : ''
  return api<ListInvitesRes>(`/api/backoffice/invites${q}`)
}

export function createInvite(email: string, full_name: string) {
  return api<{ message: string }>('/api/backoffice/invites', {
    method: 'POST',
    json: { email, full_name },
  })
}

export function deleteInvite(id: string) {
  return api<{ message: string }>(`/api/backoffice/invites/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function resendInvite(id: string) {
  return api<{ message: string }>(`/api/backoffice/invites/${encodeURIComponent(id)}/resend`, {
    method: 'POST',
  })
}

export function getRemarcarByToken(token: string) {
  return api<{
    appointment_id: string
    patient_name: string
    current_date: string
    current_start_time: string
    slots: { date: string; start_time: string }[]
  }>(`/api/appointments/remarcar/${encodeURIComponent(token)}`)
}

export function confirmRemarcar(token: string) {
  return api<{ message: string }>(`/api/appointments/remarcar/${encodeURIComponent(token)}/confirm`, {
    method: 'POST',
  })
}

export function remarcarAppointment(token: string, appointment_date: string, start_time: string) {
  return api<{ message: string }>(`/api/appointments/remarcar/${encodeURIComponent(token)}`, {
    method: 'PATCH',
    json: { appointment_date, start_time },
  })
}

export function triggerReminder(professionalId?: string) {
  const q = professionalId ? `?professional_id=${encodeURIComponent(professionalId)}` : ''
  return api<{ sent: number; skipped: number; date: string }>(`/api/backoffice/reminder/trigger${q}`, {
    method: 'POST',
  })
}

export type BackofficeTimelineItem = {
  kind: 'AUDIT' | 'ACCESS'
  id: string
  action: string
  actor_type: string
  actor_id?: string | null
  clinic_id?: string | null
  request_id?: string | null
  ip?: string | null
  user_agent?: string | null
  resource_type?: string | null
  resource_id?: string | null
  patient_id?: string | null
  is_impersonated: boolean
  impersonation_session_id?: string | null
  source: string
  severity: string
  metadata?: Record<string, unknown>
  created_at: string
}

export function listBackofficeTimeline(params: {
  limit?: number
  offset?: number
  from?: string
  to?: string
  request_id?: string
  severity?: string
  source?: string
} = {}) {
  const q = new URLSearchParams()
  if (params.limit) q.set('limit', String(params.limit))
  if (params.offset) q.set('offset', String(params.offset))
  if (params.from) q.set('from', params.from)
  if (params.to) q.set('to', params.to)
  if (params.request_id) q.set('request_id', params.request_id)
  if (params.severity) q.set('severity', params.severity)
  if (params.source) q.set('source', params.source)
  const qs = q.toString()
  return api<{ items: BackofficeTimelineItem[]; limit: number; offset: number }>(`/api/backoffice/timeline${qs ? `?${qs}` : ''}`)
}

export type BackofficeErrorItem = {
  id: string
  created_at: string
  request_id?: string | null
  source: string
  severity: string
  clinic_id?: string | null
  actor_type?: string | null
  actor_id?: string | null
  path?: string | null
  http_method?: string | null
  action_name?: string | null
  kind?: string | null
  message?: string | null
  stack?: string | null
  pg_code?: string | null
  pg_message?: string | null
  metadata?: Record<string, unknown>
}

export function listBackofficeErrors(params: {
  limit?: number
  offset?: number
  from?: string
  to?: string
  request_id?: string
  severity?: string
  source?: string
} = {}) {
  const q = new URLSearchParams()
  if (params.limit) q.set('limit', String(params.limit))
  if (params.offset) q.set('offset', String(params.offset))
  if (params.from) q.set('from', params.from)
  if (params.to) q.set('to', params.to)
  if (params.request_id) q.set('request_id', params.request_id)
  if (params.severity) q.set('severity', params.severity)
  if (params.source) q.set('source', params.source)
  const qs = q.toString()
  return api<{ items: BackofficeErrorItem[]; limit: number; offset: number }>(`/api/backoffice/errors${qs ? `?${qs}` : ''}`)
}

export function getInviteByToken(token: string) {
  return api<{ email: string; full_name: string; clinic_name: string; expires_at: string }>(
    `/api/invites/by-token?token=${encodeURIComponent(token)}`
  )
}

export function acceptInvite(data: {
  token: string
  password: string
  full_name?: string
  trade_name?: string
  birth_date?: string
  cpf?: string
  address?: Address
  marital_status?: string
}) {
  return api<{ message: string }>('/api/invites/accept', {
    method: 'POST',
    json: data,
  })
}

export function createPatientInvite(email: string, full_name: string) {
  return api<{ message: string; invite_id?: string; expires_at?: string }>('/api/patient-invites', {
    method: 'POST',
    json: { email, full_name },
  })
}

export function getPatientInviteByToken(token: string) {
  return api<{ email: string; full_name: string; clinic_name: string; expires_at: string }>(
    `/api/patient-invites/by-token?token=${encodeURIComponent(token)}`
  )
}

export function acceptPatientInvite(data: {
  token: string
  same_person: boolean
  guardian_full_name: string
  guardian_cpf: string
  guardian_address: Address
  guardian_birth_date: string
  patient_full_name: string
  patient_birth_date: string
}) {
  return api<{ message: string }>('/api/patient-invites/accept', {
    method: 'POST',
    json: data,
  })
}

export type ListRecordEntriesRes = {
  entries: { id: string; content: string; entry_date: string; author_id: string; author_type: string; created_at: string }[]
  limit: number
  offset: number
  total: number
}

export function listRecordEntries(patientId: string, opts?: { limit?: number; offset?: number }) {
  const params = new URLSearchParams()
  if (opts?.limit != null) params.set('limit', String(opts.limit))
  if (opts?.offset != null) params.set('offset', String(opts.offset))
  const q = params.toString() ? `?${params.toString()}` : ''
  return api<ListRecordEntriesRes>(`/api/patients/${patientId}/record-entries${q}`)
}

export function createRecordEntry(patientId: string, content: string, entry_date?: string) {
  return api<{ id: string }>(`/api/patients/${patientId}/record-entries`, {
    method: 'POST',
    json: entry_date ? { content, entry_date } : { content },
  })
}
