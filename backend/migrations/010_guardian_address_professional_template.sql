-- Legal guardian address (for registration)
ALTER TABLE legal_guardians ADD COLUMN IF NOT EXISTS address TEXT;

-- Contract template per professional (each professional can have their own)
ALTER TABLE contract_templates ADD COLUMN IF NOT EXISTS professional_id UUID REFERENCES professionals(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_contract_templates_professional ON contract_templates(professional_id);
