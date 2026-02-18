-- Local e data previstos para assinatura (preenchidos ao enviar; [LOCAL] e [DATA] no documento; na assinatura a data real é usada)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS sign_place TEXT;
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS sign_date DATE;
