-- E-mail do profissional deve ser único globalmente (case-insensitive).
-- Observação: se existir duplicidade de e-mail (ignorando maiúsculas/minúsculas), esta migration vai falhar.

-- Remove o constraint antigo (unique por clinic_id + email), se existir.
ALTER TABLE professionals DROP CONSTRAINT IF EXISTS professionals_clinic_id_email_key;

-- Índice único global por e-mail (case-insensitive).
CREATE UNIQUE INDEX IF NOT EXISTS ux_professionals_email_lower ON professionals (lower(email));

