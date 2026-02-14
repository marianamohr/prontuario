-- Contract templates (por clínica)
CREATE TABLE IF NOT EXISTS contract_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  body_html TEXT NOT NULL,
  version INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_contract_templates_clinic ON contract_templates(clinic_id);

-- Contracts (patient_id + legal_guardian_id, signer_relation, signer_is_patient)
CREATE TABLE IF NOT EXISTS contracts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
  legal_guardian_id UUID NOT NULL REFERENCES legal_guardians(id) ON DELETE CASCADE,
  template_id UUID NOT NULL REFERENCES contract_templates(id) ON DELETE RESTRICT,
  signer_relation TEXT NOT NULL,
  signer_is_patient BOOLEAN NOT NULL DEFAULT false,
  status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'SIGNED', 'CANCELLED')),
  signed_at TIMESTAMPTZ,
  pdf_url TEXT,
  pdf_sha256 TEXT,
  audit_json JSONB,
  template_version INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_contracts_clinic ON contracts(clinic_id);
CREATE INDEX idx_contracts_patient ON contracts(patient_id);
CREATE INDEX idx_contracts_guardian ON contracts(legal_guardian_id);
CREATE INDEX idx_contracts_status ON contracts(status);

-- Tokens de acesso à página de assinatura (email)
CREATE TABLE IF NOT EXISTS contract_access_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_contract_access_tokens_token ON contract_access_tokens(token);
CREATE INDEX idx_contract_access_tokens_contract ON contract_access_tokens(contract_id);
