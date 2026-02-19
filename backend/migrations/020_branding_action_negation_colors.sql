-- Action (accept/submit) and negation (delete/cancel) button colors
ALTER TABLE clinics
  ADD COLUMN IF NOT EXISTS action_button_color TEXT,
  ADD COLUMN IF NOT EXISTS negation_button_color TEXT;

COMMENT ON COLUMN clinics.action_button_color IS 'Cor dos botões de ação positiva (aceitar, submeter, principal). Ex: #16a34a';
COMMENT ON COLUMN clinics.negation_button_color IS 'Cor dos botões de negação (excluir, cancelar). Ex: #dc2626';
