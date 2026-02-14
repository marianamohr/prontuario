-- Força relação 1:1 entre clinic e professional (conceito: clinic interna do profissional).
-- Permite reuso do clinic_id apenas se o profissional anterior estiver CANCELLED.
--
-- IMPORTANTE: se já existirem 2 profissionais não-cancelados na mesma clinic_id, esta migration vai falhar.
CREATE UNIQUE INDEX IF NOT EXISTS ux_professionals_one_active_per_clinic
ON professionals(clinic_id)
WHERE status != 'CANCELLED';

