-- Clinic schedule configuration by weekday (0=Sunday, 1=Monday, ..., 6=Saturday)
CREATE TABLE IF NOT EXISTS clinic_schedule_config (
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  day_of_week SMALLINT NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
  start_time TIME,
  end_time TIME,
  consultation_duration_minutes INT NOT NULL DEFAULT 50,
  interval_minutes INT NOT NULL DEFAULT 10,
  lunch_start TIME,
  lunch_end TIME,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (clinic_id, day_of_week)
);

CREATE INDEX idx_clinic_schedule_config_clinic ON clinic_schedule_config(clinic_id);

-- Pre-schedule rules on contract (e.g. every Tuesday 3pm) — shown on contract for guardian to sign
CREATE TABLE IF NOT EXISTS contract_schedule_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
  day_of_week SMALLINT NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
  slot_time TIME NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_contract_schedule_rules_contract ON contract_schedule_rules(contract_id);

-- Concrete appointments (created on contract sign or changed manually by the professional)
CREATE TABLE IF NOT EXISTS appointments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
  professional_id UUID NOT NULL REFERENCES professionals(id) ON DELETE CASCADE,
  patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
  contract_id UUID REFERENCES contracts(id) ON DELETE SET NULL,
  appointment_date DATE NOT NULL,
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,
  status TEXT NOT NULL DEFAULT 'CONFIRMED' CHECK (status IN ('PENDING_SIGNATURE', 'CONFIRMED', 'CANCELLED', 'COMPLETED', 'SERIES_ENDED')),
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_appointments_clinic ON appointments(clinic_id);
CREATE INDEX idx_appointments_professional ON appointments(professional_id);
CREATE INDEX idx_appointments_patient ON appointments(patient_id);
CREATE INDEX idx_appointments_contract ON appointments(contract_id);
CREATE INDEX idx_appointments_date ON appointments(appointment_date);
CREATE INDEX idx_appointments_clinic_date ON appointments(clinic_id, appointment_date);
