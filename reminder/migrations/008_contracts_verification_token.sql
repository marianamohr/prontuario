ALTER TABLE contracts ADD COLUMN IF NOT EXISTS verification_token TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_contracts_verification_token ON contracts(verification_token);
