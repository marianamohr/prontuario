-- Periodicity on contract (configured when sending, like value and dates)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS periodicidade TEXT;
