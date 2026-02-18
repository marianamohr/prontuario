-- Tipo de serviço e periodicidade no modelo de contrato (preenchimento de [TIPO_SERVICO], [PERIODICIDADE] e [OBJETO])
ALTER TABLE contract_templates ADD COLUMN IF NOT EXISTS tipo_servico TEXT;
ALTER TABLE contract_templates ADD COLUMN IF NOT EXISTS periodicidade TEXT;
