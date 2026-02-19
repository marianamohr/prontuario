-- Optional patient CPF: store encrypted (reversible) + hash for search/uniqueness per clinic.
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_encrypted BYTEA;
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_nonce BYTEA;
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_key_version TEXT;
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_hash TEXT;

-- CPF must be unique per clinic only when filled and when patient is not soft-deleted.
CREATE UNIQUE INDEX IF NOT EXISTS ux_patients_clinic_cpf_hash_active
ON patients(clinic_id, cpf_hash)
WHERE deleted_at IS NULL AND cpf_hash IS NOT NULL;

