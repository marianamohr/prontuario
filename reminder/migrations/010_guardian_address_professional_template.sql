-- Endereço do responsável legal (para cadastro)
ALTER TABLE legal_guardians ADD COLUMN IF NOT EXISTS address TEXT;

-- Modelo de contrato por profissional (cada profissional pode ter seu próprio)
ALTER TABLE contract_templates ADD COLUMN IF NOT EXISTS professional_id UUID REFERENCES professionals(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_contract_templates_professional ON contract_templates(professional_id);
