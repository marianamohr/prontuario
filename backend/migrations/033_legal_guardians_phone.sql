-- Telefone do responsável legal para WhatsApp (E.164, opcional)
ALTER TABLE legal_guardians ADD COLUMN IF NOT EXISTS phone TEXT;
