-- Weekdays the professional sees patients: only days with enabled = true appear for configuration and on the agenda
ALTER TABLE clinic_schedule_config
  ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true;
