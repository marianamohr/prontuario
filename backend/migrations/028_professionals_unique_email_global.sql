-- Professional email must be globally unique (case-insensitive).
-- Note: if duplicate email exists (ignoring case), this migration will fail.

-- Remove old constraint (unique per clinic_id + email), if present.
ALTER TABLE professionals DROP CONSTRAINT IF EXISTS professionals_clinic_id_email_key;

-- Global unique index on email (case-insensitive).
CREATE UNIQUE INDEX IF NOT EXISTS ux_professionals_email_lower ON professionals (lower(email));

