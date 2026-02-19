-- Place and date expected for signature (filled when sending; [LOCAL] and [DATA] in document; at sign time actual date is used)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS sign_place TEXT;
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS sign_date DATE;
