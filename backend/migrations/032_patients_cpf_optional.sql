-- CPF opcional do paciente: armazenar criptografado (reversível) + hash para busca/uniqueness por clínica.
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_encrypted BYTEA;
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_nonce BYTEA;
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_key_version TEXT;
ALTER TABLE patients ADD COLUMN IF NOT EXISTS cpf_hash TEXT;

-- CPF deve ser único por clínica apenas quando preenchido e quando o paciente não está soft-deletado.
CREATE UNIQUE INDEX IF NOT EXISTS ux_patients_clinic_cpf_hash_active
ON patients(clinic_id, cpf_hash)
WHERE deleted_at IS NULL AND cpf_hash IS NOT NULL;

