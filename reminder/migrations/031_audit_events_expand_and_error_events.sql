-- Expande audit_events para suportar timeline completa (writes) com correlação por request_id
-- e cria error_events para logs de bugs/erros (frontend + backend), com retenção via job no backend.

ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS clinic_id UUID REFERENCES clinics(id) ON DELETE SET NULL;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS request_id TEXT;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS ip TEXT;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS user_agent TEXT;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS resource_type TEXT;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS resource_id UUID;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS patient_id UUID REFERENCES patients(id) ON DELETE SET NULL;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS is_impersonated BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS impersonation_session_id UUID;
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'USER';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS severity TEXT NOT NULL DEFAULT 'INFO';

CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_events_clinic_id ON audit_events(clinic_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor_type, actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource ON audit_events(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_patient_id ON audit_events(patient_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_request_id ON audit_events(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_source ON audit_events(source);
CREATE INDEX IF NOT EXISTS idx_audit_events_severity ON audit_events(severity);

-- Error logs (bugs / catches / backend errors / db errors)
CREATE TABLE IF NOT EXISTS error_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- Correlation
  request_id TEXT,
  source TEXT NOT NULL CHECK (source IN ('FRONTEND', 'BACKEND')),
  severity TEXT NOT NULL CHECK (severity IN ('WARN', 'ERROR')),

  -- Actor context (no PII)
  clinic_id UUID REFERENCES clinics(id) ON DELETE SET NULL,
  actor_type TEXT,
  actor_id UUID,
  is_impersonated BOOLEAN NOT NULL DEFAULT false,
  impersonation_session_id UUID,

  -- Trigger context
  http_method TEXT,
  path TEXT,
  action_name TEXT,

  -- Error info (sanitized)
  kind TEXT,
  message TEXT,
  stack TEXT,
  pg_code TEXT,
  pg_message TEXT,
  metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_error_events_created_at ON error_events(created_at);
CREATE INDEX IF NOT EXISTS idx_error_events_request_id ON error_events(request_id);
CREATE INDEX IF NOT EXISTS idx_error_events_clinic_id ON error_events(clinic_id);
CREATE INDEX IF NOT EXISTS idx_error_events_actor ON error_events(actor_type, actor_id);
CREATE INDEX IF NOT EXISTS idx_error_events_source ON error_events(source);
CREATE INDEX IF NOT EXISTS idx_error_events_severity ON error_events(severity);
CREATE INDEX IF NOT EXISTS idx_error_events_action_name ON error_events(action_name);

