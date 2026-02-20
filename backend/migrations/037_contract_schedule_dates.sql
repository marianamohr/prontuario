-- Datas específicas para pré-agendamento (consulta única), em contraste com contract_schedule_rules (recorrência semanal).
CREATE TABLE IF NOT EXISTS contract_schedule_dates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
  appointment_date DATE NOT NULL,
  slot_time TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_contract_schedule_dates_contract ON contract_schedule_dates(contract_id);
