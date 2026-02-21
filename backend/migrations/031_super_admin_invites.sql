-- Invites for SUPER_ADMIN registration via link (invite-style flow).
-- Super admin sends email + name; invited admin sets password and completes registration.

CREATE TABLE IF NOT EXISTS super_admin_invites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  full_name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED')),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_super_admin_invites_token ON super_admin_invites(token);
CREATE INDEX IF NOT EXISTS idx_super_admin_invites_email ON super_admin_invites(email);
CREATE INDEX IF NOT EXISTS idx_super_admin_invites_expires ON super_admin_invites(expires_at);

