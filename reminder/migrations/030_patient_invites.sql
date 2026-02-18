-- Convites para cadastro de paciente/responsável via link (fluxo tipo invite).
-- O profissional envia e-mail + nome; o responsável completa CPF/endereço/datas no link.

CREATE TABLE IF NOT EXISTS patient_invites (
  id uuid PRIMARY KEY,
  token text NOT NULL UNIQUE,
  clinic_id uuid NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  guardian_email text NOT NULL,
  guardian_full_name text NOT NULL,
  status text NOT NULL DEFAULT 'PENDING',
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_patient_invites_clinic_id ON patient_invites(clinic_id);
CREATE INDEX IF NOT EXISTS idx_patient_invites_token ON patient_invites(token);

