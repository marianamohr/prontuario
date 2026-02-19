-- Service type and periodicity on contract template (fill [TIPO_SERVICO], [PERIODICIDADE] and [OBJETO])
ALTER TABLE contract_templates ADD COLUMN IF NOT EXISTS tipo_servico TEXT;
ALTER TABLE contract_templates ADD COLUMN IF NOT EXISTS periodicidade TEXT;
