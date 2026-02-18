-- Professional invites (admin sends email + link; professional completes registration)
CREATE TABLE IF NOT EXISTS professional_invites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  full_name TEXT NOT NULL,
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED')),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_professional_invites_token ON professional_invites(token);
CREATE INDEX idx_professional_invites_email ON professional_invites(email);
CREATE INDEX idx_professional_invites_expires ON professional_invites(expires_at);

-- Extra fields for professionals (filled on invite acceptance)
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS birth_date DATE;
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS cpf_hash TEXT;
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS address TEXT;
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS marital_status TEXT;
