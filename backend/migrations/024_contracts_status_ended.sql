-- Add 'ENDED' to allowed contract statuses (closure: service provided up to date).
ALTER TABLE contracts DROP CONSTRAINT IF EXISTS contracts_status_check;
ALTER TABLE contracts ADD CONSTRAINT contracts_status_check CHECK (status IN ('PENDING', 'SIGNED', 'CANCELLED', 'ENDED'));
