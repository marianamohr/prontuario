-- Tabela de endereços (rua, numero, complemento, bairro, cidade, estado, pais, cep)
CREATE TABLE IF NOT EXISTS addresses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  street TEXT,
  number TEXT,
  complement TEXT,
  neighborhood TEXT,
  city TEXT,
  state TEXT,
  country TEXT,
  zip TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- FKs em legal_guardians, professionals e patients
ALTER TABLE legal_guardians ADD COLUMN IF NOT EXISTS address_id UUID REFERENCES addresses(id) ON DELETE SET NULL;
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS address_id UUID REFERENCES addresses(id) ON DELETE SET NULL;
ALTER TABLE patients ADD COLUMN IF NOT EXISTS address_id UUID REFERENCES addresses(id) ON DELETE SET NULL;

-- Migração de dados: legal_guardians (formato atual 6 linhas: rua, bairro, cidade, estado, pais, cep)
DO $$
DECLARE
  r RECORD;
  aid UUID;
BEGIN
  FOR r IN SELECT id, address FROM legal_guardians WHERE address IS NOT NULL AND trim(address) != ''
  LOOP
    INSERT INTO addresses (street, number, complement, neighborhood, city, state, country, zip)
    VALUES (
      nullif(trim(split_part(r.address, E'\n', 1)), ''),
      NULL,
      NULL,
      nullif(trim(split_part(r.address, E'\n', 2)), ''),
      nullif(trim(split_part(r.address, E'\n', 3)), ''),
      nullif(trim(split_part(r.address, E'\n', 4)), ''),
      nullif(trim(split_part(r.address, E'\n', 5)), ''),
      nullif(trim(split_part(r.address, E'\n', 6)), '')
    )
    RETURNING id INTO aid;
    UPDATE legal_guardians SET address_id = aid WHERE id = r.id;
  END LOOP;
END $$;

-- Migração de dados: professionals
DO $$
DECLARE
  r RECORD;
  aid UUID;
BEGIN
  FOR r IN SELECT id, address FROM professionals WHERE address IS NOT NULL AND trim(address) != ''
  LOOP
    INSERT INTO addresses (street, number, complement, neighborhood, city, state, country, zip)
    VALUES (
      nullif(trim(split_part(r.address, E'\n', 1)), ''),
      NULL,
      NULL,
      nullif(trim(split_part(r.address, E'\n', 2)), ''),
      nullif(trim(split_part(r.address, E'\n', 3)), ''),
      nullif(trim(split_part(r.address, E'\n', 4)), ''),
      nullif(trim(split_part(r.address, E'\n', 5)), ''),
      nullif(trim(split_part(r.address, E'\n', 6)), '')
    )
    RETURNING id INTO aid;
    UPDATE professionals SET address_id = aid WHERE id = r.id;
  END LOOP;
END $$;

-- Remover coluna address
ALTER TABLE legal_guardians DROP COLUMN IF EXISTS address;
ALTER TABLE professionals DROP COLUMN IF EXISTS address;

-- Índices para JOINs
CREATE INDEX IF NOT EXISTS idx_legal_guardians_address_id ON legal_guardians(address_id);
CREATE INDEX IF NOT EXISTS idx_professionals_address_id ON professionals(address_id);
CREATE INDEX IF NOT EXISTS idx_patients_address_id ON patients(address_id);
