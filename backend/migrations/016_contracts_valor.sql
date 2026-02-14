-- Valor do serviço no contrato (placeholder [VALOR], configurado ao disparar)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS valor TEXT;
