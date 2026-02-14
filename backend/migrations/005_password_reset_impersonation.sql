CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token TEXT NOT NULL UNIQUE,
  user_type TEXT NOT NULL CHECK (user_type IN ('PROFESSIONAL', 'SUPER_ADMIN', 'LEGAL_GUARDIAN')),
  user_id UUID NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at);

CREATE TABLE IF NOT EXISTS impersonation_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  admin_id UUID NOT NULL,
  target_user_type TEXT NOT NULL CHECK (target_user_type IN ('PROFESSIONAL', 'LEGAL_GUARDIAN')),
  target_user_id UUID NOT NULL,
  clinic_id UUID REFERENCES clinics(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at TIMESTAMPTZ
);

CREATE INDEX idx_impersonation_sessions_admin ON impersonation_sessions(admin_id);
CREATE INDEX idx_impersonation_sessions_ended ON impersonation_sessions(ended_at);
