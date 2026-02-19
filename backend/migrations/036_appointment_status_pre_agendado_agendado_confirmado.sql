-- New appointment statuses: PRE_AGENDADO, AGENDADO, CONFIRMADO (replace PENDING_SIGNATURE and CONFIRMED)

-- Drop old constraint BEFORE migrating data (avoids check violation on UPDATE)
DO $$
DECLARE
  r RECORD;
BEGIN
  FOR r IN (SELECT conname FROM pg_constraint c
            JOIN pg_class t ON t.oid = c.conrelid
            WHERE t.relname = 'appointments' AND c.contype = 'c'
              AND pg_get_constraintdef(c.oid) LIKE '%status%')
  LOOP
    EXECUTE format('ALTER TABLE appointments DROP CONSTRAINT %I', r.conname);
    EXIT;
  END LOOP;
END $$;

-- Migrate existing data
UPDATE appointments SET status = 'AGENDADO' WHERE status = 'PENDING_SIGNATURE';
UPDATE appointments SET status = 'CONFIRMADO' WHERE status = 'CONFIRMED';

-- New constraint with valid statuses
ALTER TABLE appointments ADD CONSTRAINT appointments_status_check CHECK (
  status IN ('PRE_AGENDADO', 'AGENDADO', 'CONFIRMADO', 'CANCELLED', 'COMPLETED', 'SERIES_ENDED')
);

-- Default for new records (when not provided)
ALTER TABLE appointments ALTER COLUMN status SET DEFAULT 'AGENDADO';
