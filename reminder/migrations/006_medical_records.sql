CREATE TABLE IF NOT EXISTS medical_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(patient_id)
);

CREATE INDEX idx_medical_records_patient ON medical_records(patient_id);

CREATE TABLE IF NOT EXISTS record_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  medical_record_id UUID NOT NULL REFERENCES medical_records(id) ON DELETE CASCADE,
  content_encrypted BYTEA NOT NULL,
  content_nonce BYTEA NOT NULL,
  content_key_version TEXT NOT NULL,
  entry_date DATE NOT NULL,
  author_id UUID NOT NULL,
  author_type TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_record_entries_medical_record ON record_entries(medical_record_id);
