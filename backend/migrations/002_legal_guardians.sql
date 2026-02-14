DO $$ BEGIN
  CREATE TYPE auth_provider_enum AS ENUM ('GOOGLE', 'LOCAL', 'HYBRID');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS legal_guardians (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  google_sub TEXT UNIQUE,
  password_hash TEXT,
  full_name TEXT NOT NULL,
  cpf_encrypted BYTEA,
  cpf_nonce BYTEA,
  cpf_key_version TEXT,
  cpf_hash TEXT UNIQUE,
  auth_provider auth_provider_enum NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'CANCELLED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_legal_guardians_email ON legal_guardians(email);
CREATE INDEX idx_legal_guardians_cpf_hash ON legal_guardians(cpf_hash);
CREATE INDEX idx_legal_guardians_google_sub ON legal_guardians(google_sub);
