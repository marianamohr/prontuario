-- Quantidade de agendamentos a criar ao assinar o contrato (null = sem limite, usa período do contrato)
ALTER TABLE contracts ADD COLUMN IF NOT EXISTS num_appointments INT;
