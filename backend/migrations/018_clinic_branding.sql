-- White-label: aparência da clínica (header, fundo, botão home)
ALTER TABLE clinics
  ADD COLUMN IF NOT EXISTS primary_color TEXT,
  ADD COLUMN IF NOT EXISTS background_color TEXT,
  ADD COLUMN IF NOT EXISTS home_label TEXT,
  ADD COLUMN IF NOT EXISTS home_image_url TEXT;

COMMENT ON COLUMN clinics.primary_color IS 'Cor principal (header, botões). Ex: #1a1a2e';
COMMENT ON COLUMN clinics.background_color IS 'Cor de fundo da área logada. Ex: #ffffff';
COMMENT ON COLUMN clinics.home_label IS 'Texto do botão/link Home. Ex: Minha Clínica';
COMMENT ON COLUMN clinics.home_image_url IS 'URL da imagem/logo do botão Home (opcional)';
