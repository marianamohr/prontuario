-- Adiciona 'ENDED' aos status permitidos em contracts (encerramento: serviço prestado até a data).
ALTER TABLE contracts DROP CONSTRAINT IF EXISTS contracts_status_check;
ALTER TABLE contracts ADD CONSTRAINT contracts_status_check CHECK (status IN ('PENDING', 'SIGNED', 'CANCELLED', 'ENDED'));
