-- Number of appointments to create when signing the contract (null = no limit, uses contract period)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS num_appointments INT;
