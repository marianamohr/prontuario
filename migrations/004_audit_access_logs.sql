-- Access logs (LGPD - quem acessou o quê)
CREATE TABLE IF NOT EXISTS access_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID REFERENCES clinics(id) ON DELETE SET NULL,
  actor_type TEXT NOT NULL,
  actor_id UUID NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('READ', 'VIEW', 'DOWNLOAD', 'SIGN')),
  resource_type TEXT NOT NULL,
  resource_id UUID,
  patient_id UUID REFERENCES patients(id) ON DELETE SET NULL,
  ip TEXT,
  user_agent TEXT,
  request_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_logs_clinic ON access_logs(clinic_id);
CREATE INDEX idx_access_logs_actor ON access_logs(actor_type, actor_id);
CREATE INDEX idx_access_logs_created ON access_logs(created_at);

-- Audit events (ações sensíveis)
CREATE TABLE IF NOT EXISTS audit_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  action TEXT NOT NULL,
  actor_type TEXT NOT NULL,
  actor_id UUID,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_created ON audit_events(created_at);
