-- Imagem de assinatura do profissional (data URL base64)
ALTER TABLE professionals ADD COLUMN IF NOT EXISTS signature_image_data TEXT;

-- Contrato passa a registrar qual profissional disparou (para exibir assinatura no PDF)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS professional_id UUID REFERENCES professionals(id) ON DELETE SET NULL;
