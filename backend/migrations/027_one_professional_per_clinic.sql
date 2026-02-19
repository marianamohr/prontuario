-- Enforce 1:1 relationship between clinic and professional (concept: professional's internal clinic).
-- Allows reuse of clinic_id only if the previous professional is CANCELLED.
--
-- IMPORTANT: if 2 non-cancelled professionals already exist for the same clinic_id, this migration will fail.
CREATE UNIQUE INDEX IF NOT EXISTS ux_professionals_one_active_per_clinic
ON professionals(clinic_id)
WHERE status != 'CANCELLED';

