-- Periodicidade no contrato (configurada ao disparar, como valor e datas)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS periodicidade TEXT;
