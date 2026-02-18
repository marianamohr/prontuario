-- Data de nascimento do responsável legal (para uso em contratos)
ALTER TABLE legal_guardians ADD COLUMN IF NOT EXISTS birth_date DATE;
