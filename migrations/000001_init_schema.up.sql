-- Clinics (tenant root)
CREATE TABLE clinics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Professionals (tenant admin)
CREATE TYPE user_status AS ENUM ('ACTIVE', 'SUSPENDED', 'CANCELLED');
CREATE TYPE auth_provider AS ENUM ('GOOGLE', 'LOCAL', 'HYBRID');
CREATE TYPE user_role AS ENUM ('PROFESSIONAL', 'LEGAL_GUARDIAN', 'SUPER_ADMIN');

CREATE TABLE professionals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  status user_status NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(clinic_id, email)
);

CREATE INDEX idx_professionals_clinic ON professionals(clinic_id);

-- Super admins (backoffice)
CREATE TABLE super_admins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL,
  status user_status NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Legal guardians (assinantes)
CREATE TABLE legal_guardians (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  google_sub TEXT UNIQUE,
  password_hash TEXT,
  full_name TEXT NOT NULL,
  cpf_encrypted BYTEA,
  cpf_nonce BYTEA,
  cpf_key_version TEXT,
  cpf_hash TEXT UNIQUE,
  auth_provider auth_provider NOT NULL,
  status user_status NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_legal_guardians_email ON legal_guardians(email);
CREATE INDEX idx_legal_guardians_cpf_hash ON legal_guardians(cpf_hash);

-- Patients (pessoa atendida, pode ser diferente do responsável)
CREATE TABLE patients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  full_name TEXT NOT NULL,
  birth_date DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_patients_clinic ON patients(clinic_id);

-- Vinculação paciente-responsável com permissões granulares
CREATE TABLE patient_guardians (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
  legal_guardian_id UUID NOT NULL REFERENCES legal_guardians(id) ON DELETE CASCADE,
  relation TEXT NOT NULL,
  can_view_medical_record BOOLEAN NOT NULL DEFAULT false,
  can_view_contracts BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(patient_id, legal_guardian_id)
);

CREATE INDEX idx_patient_guardians_patient ON patient_guardians(patient_id);
CREATE INDEX idx_patient_guardians_guardian ON patient_guardians(legal_guardian_id);

-- Audit events (LGPD)
CREATE TABLE audit_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  action TEXT NOT NULL,
  actor_type TEXT NOT NULL,
  actor_id TEXT,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_events_created ON audit_events(created_at);
CREATE INDEX idx_audit_events_actor ON audit_events(actor_type, actor_id);

-- Access logs (LGPD)
CREATE TABLE access_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID REFERENCES clinics(id),
  actor_type TEXT NOT NULL,
  actor_id TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT,
  patient_id UUID REFERENCES patients(id),
  ip TEXT,
  user_agent TEXT,
  request_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_logs_clinic ON access_logs(clinic_id);
CREATE INDEX idx_access_logs_created ON access_logs(created_at);
CREATE INDEX idx_access_logs_patient ON access_logs(patient_id);
