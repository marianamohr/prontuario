-- Professional signature image (base64 data URL)
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS signature_image_data TEXT;

-- Contract now records which professional sent it (to show signature on PDF)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS professional_id UUID REFERENCES professionals(id) ON DELETE SET NULL;
