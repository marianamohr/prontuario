-- Legal guardian birth date (for use in contracts)
ALTER TABLE legal_guardians ADD COLUMN IF NOT EXISTS birth_date DATE;
