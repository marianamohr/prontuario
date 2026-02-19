-- Service value on contract (placeholder [VALOR], configured when sending)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS valor TEXT;
