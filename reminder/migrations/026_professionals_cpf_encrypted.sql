-- CPF do profissional: armazenar criptografado (reversível) + hash para busca/uniqueness.
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS cpf_encrypted BYTEA;
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS cpf_nonce BYTEA;
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS cpf_key_version TEXT;

CREATE INDEX IF NOT EXISTS idx_professionals_cpf_hash ON professionals(cpf_hash);
