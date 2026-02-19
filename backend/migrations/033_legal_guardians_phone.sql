-- Legal guardian phone for WhatsApp (E.164, optional)
ALTER TABLE legal_guardians ADD COLUMN IF NOT EXISTS phone TEXT;
