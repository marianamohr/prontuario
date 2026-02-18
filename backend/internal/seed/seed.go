package seed

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/backend/internal/auth"
)

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	var n int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM super_admins").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		// Admin e clínicas já existem; garantir que clínicas com 0 pacientes tenham o seed de pacientes
		log.Printf("seed: super admins existem, verificando pacientes por clínica")
		return seedPatientsForExistingClinics(ctx, pool)
	}

	adminHash, err := auth.HashPassword("Admin123!")
	if err != nil {
		return err
	}
	adminID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO super_admins (id, email, password_hash, full_name, status)
		VALUES ($1, $2, $3, $4, 'ACTIVE')
	`, adminID, "admin@prontuario.local", adminHash, "Super Admin")
	if err != nil {
		return err
	}

	var clinicCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM clinics").Scan(&clinicCount); err != nil {
		return err
	}
	if clinicCount > 0 {
		// Clínicas já existem (ex.: seed anterior); criar pacientes se alguma clínica não tiver nenhum
		return seedPatientsForExistingClinics(ctx, pool)
	}

	profHash, err := auth.HashPassword("ChangeMe123!")
	if err != nil {
		return err
	}
	c1, c2 := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO clinics (id, name) VALUES ($1, 'Clínica A'), ($2, 'Clínica B')`, c1, c2)
	if err != nil {
		return err
	}
	p1, p2 := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO professionals (id, clinic_id, email, password_hash, full_name, status)
		VALUES ($1, $2, 'profa@clinica-a.local', $3, 'Prof A', 'ACTIVE'),
		       ($4, $5, 'profb@clinica-b.local', $3, 'Prof B', 'ACTIVE')
	`, p1, c1, profHash, p2, c2)
	if err != nil {
		return err
	}

	guardianHash, err := auth.HashPassword("Guardian123!")
	if err != nil {
		return err
	}
	for _, clinic := range []struct {
		id   uuid.UUID
		name string
	}{{c1, "clinica-a"}, {c2, "clinica-b"}} {
		if err := seedPatientsForClinic(ctx, pool, clinic.id, clinic.name, guardianHash); err != nil {
			return err
		}
	}
	return nil
}

// EnsurePatientsForExistingClinics is exported so backoffice can trigger it manually.
// Cria 4 pacientes de exemplo em cada clínica que ainda não tem nenhum.
func EnsurePatientsForExistingClinics(ctx context.Context, pool *pgxpool.Pool) error {
	return seedPatientsForExistingClinics(ctx, pool)
}

// seedPatientsForExistingClinics seeds 4 patients only for the demo clinics "Clínica A" and "Clínica B".
// Clínicas criadas por convite (novos profissionais) ficam com 0 pacientes.
func seedPatientsForExistingClinics(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `SELECT id, name FROM clinics ORDER BY name`)
	if err != nil {
		return err
	}
	defer rows.Close()
	guardianHash, err := auth.HashPassword("Guardian123!")
	if err != nil {
		return err
	}
	// Só preencher pacientes nas clínicas de demonstração (profa@clinica-a.local e profb@clinica-b.local).
	// Novos profissionais (criados por convite) não recebem seed de pacientes.
	demoClinics := map[string]string{"Clínica A": "clinica-a", "Clínica B": "clinica-b"}
	seeded := 0
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		prefix, isDemo := demoClinics[name]
		if !isDemo {
			continue
		}
		var n int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM patients WHERE clinic_id = $1", id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		log.Printf("seed: clínica %q (%s) sem pacientes, inserindo 4", name, id.String())
		if err := seedPatientsForClinic(ctx, pool, id, prefix, guardianHash); err != nil {
			log.Printf("seed: erro ao inserir pacientes para %s: %v", name, err)
			return err
		}
		seeded++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if seeded > 0 {
		log.Printf("seed: pacientes criados para %d clínica(s)", seeded)
	}
	return nil
}

// seedPatientsForClinic creates 4 patients per clinic: 2 adults (patient = guardian) and 2 children (guardian = other adult).
func seedPatientsForClinic(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, clinicPrefix string, guardianHash string) error {
	// 2 adultos: paciente e responsável são a mesma pessoa
	adult1PatientID := uuid.New()
	adult1GuardianID := uuid.New()
	adult2PatientID := uuid.New()
	adult2GuardianID := uuid.New()
	// 2 crianças: paciente = criança, responsável = adulto
	child1PatientID := uuid.New()
	child1GuardianID := uuid.New()
	child2PatientID := uuid.New()
	child2GuardianID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO patients (id, clinic_id, full_name, birth_date, email) VALUES
		($1, $2, 'Maria Silva', '1985-03-15', $6),
		($3, $2, 'João Santos', '1990-07-22', $7),
		($4, $2, 'Ana Silva', '2018-01-10', $8),
		($5, $2, 'Pedro Santos', '2020-05-03', $9)
	`, adult1PatientID, clinicID, adult2PatientID, child1PatientID, child2PatientID,
		"maria.silva@"+clinicPrefix+".local",
		"joao.santos@"+clinicPrefix+".local",
		"ana.silva@"+clinicPrefix+".local",
		"pedro.santos@"+clinicPrefix+".local")
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO legal_guardians (id, email, full_name, password_hash, auth_provider, status) VALUES
		($1, $2, 'Maria Silva', $3, 'LOCAL', 'ACTIVE'),
		($4, $5, 'João Santos', $3, 'LOCAL', 'ACTIVE'),
		($6, $7, 'Carlos Silva', $3, 'LOCAL', 'ACTIVE'),
		($8, $9, 'Fernanda Santos', $3, 'LOCAL', 'ACTIVE')
	`,
		adult1GuardianID, "maria.silva@"+clinicPrefix+".local", guardianHash,
		adult2GuardianID, "joao.santos@"+clinicPrefix+".local",
		child1GuardianID, "carlos.silva@"+clinicPrefix+".local",
		child2GuardianID, "fernanda.santos@"+clinicPrefix+".local",
	)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO patient_guardians (patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts) VALUES
		($1, $2, 'Titular', true, true),
		($3, $4, 'Titular', true, true),
		($5, $6, 'Pai', true, true),
		($7, $8, 'Mãe', true, true)
	`,
		adult1PatientID, adult1GuardianID,
		adult2PatientID, adult2GuardianID,
		child1PatientID, child1GuardianID,
		child2PatientID, child2GuardianID,
	)
	return err
}
