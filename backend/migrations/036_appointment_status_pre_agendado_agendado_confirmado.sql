-- Novos status de appointment: PRE_AGENDADO, AGENDADO, CONFIRMADO (substituem PENDING_SIGNATURE e CONFIRMED)

-- Remover constraint antiga ANTES de migrar os dados (evita violar check ao fazer UPDATE)
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

-- Migrar dados existentes
UPDATE appointments SET status = 'AGENDADO' WHERE status = 'PENDING_SIGNATURE';
UPDATE appointments SET status = 'CONFIRMADO' WHERE status = 'CONFIRMED';

-- Nova constraint com os status válidos
ALTER TABLE appointments ADD CONSTRAINT appointments_status_check CHECK (
  status IN ('PRE_AGENDADO', 'AGENDADO', 'CONFIRMADO', 'CANCELLED', 'COMPLETED', 'SERIES_ENDED')
);

-- Default para novos registros (quando não informado)
ALTER TABLE appointments ALTER COLUMN status SET DEFAULT 'AGENDADO';
