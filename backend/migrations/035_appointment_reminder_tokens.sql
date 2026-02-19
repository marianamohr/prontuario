-- Tokens for attendance confirmation and reschedule (sent in WhatsApp reminder)
CREATE TABLE IF NOT EXISTS appointment_reminder_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  appointment_id UUID NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
  guardian_id UUID NOT NULL REFERENCES legal_guardians(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_appointment_reminder_tokens_token ON appointment_reminder_tokens(token);
CREATE INDEX IF NOT EXISTS idx_appointment_reminder_tokens_expires ON appointment_reminder_tokens(expires_at);
