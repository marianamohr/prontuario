CREATE TABLE IF NOT EXISTS patients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  full_name TEXT NOT NULL,
  birth_date DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_patients_clinic ON patients(clinic_id);

CREATE TABLE IF NOT EXISTS patient_guardians (
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
