-- Dias da semana que o profissional atende: só dias com enabled = true aparecem para configurar e na agenda
ALTER TABLE clinic_schedule_config
  ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT true;
