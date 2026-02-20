package seed

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/prontuario/backend/internal/auth"
	"gorm.io/gorm"
)

func Run(ctx context.Context, db *gorm.DB) error {
	var n int
	if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM super_admins").Scan(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		log.Printf("seed: super admins existem, verificando pacientes por clínica")
		return seedPatientsForExistingClinics(ctx, db)
	}

	adminHash, err := auth.HashPassword("Admin123!")
	if err != nil {
		return err
	}
	adminID := uuid.New()
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO super_admins (id, email, password_hash, full_name, status)
		VALUES (?, ?, ?, ?, 'ACTIVE')
	`, adminID, "admin@prontuario.local", adminHash, "Super Admin").Error; err != nil {
		return err
	}

	var clinicCount int
	if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM clinics").Scan(&clinicCount).Error; err != nil {
		return err
	}
	if clinicCount > 0 {
		return seedPatientsForExistingClinics(ctx, db)
	}

	profHash, err := auth.HashPassword("ChangeMe123!")
	if err != nil {
		return err
	}
	c1, c2 := uuid.New(), uuid.New()
	if err := db.WithContext(ctx).Exec(`INSERT INTO clinics (id, name) VALUES (?, 'Clínica A'), (?, 'Clínica B')`, c1, c2).Error; err != nil {
		return err
	}
	p1, p2 := uuid.New(), uuid.New()
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO professionals (id, clinic_id, email, password_hash, full_name, status)
		VALUES (?, ?, 'profa@clinica-a.local', ?, 'Prof A', 'ACTIVE'),
		       (?, ?, 'profb@clinica-b.local', ?, 'Prof B', 'ACTIVE')
	`, p1, c1, profHash, p2, c2, profHash).Error; err != nil {
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
		if err := seedPatientsForClinic(ctx, db, clinic.id, clinic.name, guardianHash); err != nil {
			return err
		}
	}
	return nil
}

// EnsurePatientsForExistingClinics is exported so backoffice can trigger it manually.
func EnsurePatientsForExistingClinics(ctx context.Context, db *gorm.DB) error {
	return seedPatientsForExistingClinics(ctx, db)
}

func seedPatientsForExistingClinics(ctx context.Context, db *gorm.DB) error {
	var rows []struct {
		ID   uuid.UUID
		Name string
	}
	if err := db.WithContext(ctx).Raw(`SELECT id, name FROM clinics ORDER BY name`).Scan(&rows).Error; err != nil {
		return err
	}
	guardianHash, err := auth.HashPassword("Guardian123!")
	if err != nil {
		return err
	}
	demoClinics := map[string]string{"Clínica A": "clinica-a", "Clínica B": "clinica-b"}
	seeded := 0
	for _, row := range rows {
		prefix, isDemo := demoClinics[row.Name]
		if !isDemo {
			continue
		}
		var n int
		if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM patients WHERE clinic_id = ?", row.ID).Scan(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		log.Printf("seed: clínica %q (%s) sem pacientes, inserindo 4", row.Name, row.ID.String())
		if err := seedPatientsForClinic(ctx, db, row.ID, prefix, guardianHash); err != nil {
			log.Printf("seed: erro ao inserir pacientes para %s: %v", row.Name, err)
			return err
		}
		seeded++
	}
	if seeded > 0 {
		log.Printf("seed: pacientes criados para %d clínica(s)", seeded)
	}
	return nil
}

func seedPatientsForClinic(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, clinicPrefix string, guardianHash string) error {
	adult1PatientID := uuid.New()
	adult1GuardianID := uuid.New()
	adult2PatientID := uuid.New()
	adult2GuardianID := uuid.New()
	child1PatientID := uuid.New()
	child1GuardianID := uuid.New()
	child2PatientID := uuid.New()
	child2GuardianID := uuid.New()

	if err := db.WithContext(ctx).Exec(`
		INSERT INTO patients (id, clinic_id, full_name, birth_date, email) VALUES
		(?, ?, 'Maria Silva', '1985-03-15', ?),
		(?, ?, 'João Santos', '1990-07-22', ?),
		(?, ?, 'Ana Silva', '2018-01-10', ?),
		(?, ?, 'Pedro Santos', '2020-05-03', ?)
	`, adult1PatientID, clinicID, "maria.silva@"+clinicPrefix+".local",
		adult2PatientID, clinicID, "joao.santos@"+clinicPrefix+".local",
		child1PatientID, clinicID, "ana.silva@"+clinicPrefix+".local",
		child2PatientID, clinicID, "pedro.santos@"+clinicPrefix+".local").Error; err != nil {
		return err
	}

	if err := db.WithContext(ctx).Exec(`
		INSERT INTO legal_guardians (id, email, full_name, password_hash, auth_provider, status) VALUES
		(?, ?, 'Maria Silva', ?, 'LOCAL', 'ACTIVE'),
		(?, ?, 'João Santos', ?, 'LOCAL', 'ACTIVE'),
		(?, ?, 'Carlos Silva', ?, 'LOCAL', 'ACTIVE'),
		(?, ?, 'Fernanda Santos', ?, 'LOCAL', 'ACTIVE')
	`,
		adult1GuardianID, "maria.silva@"+clinicPrefix+".local", guardianHash,
		adult2GuardianID, "joao.santos@"+clinicPrefix+".local", guardianHash,
		child1GuardianID, "carlos.silva@"+clinicPrefix+".local", guardianHash,
		child2GuardianID, "fernanda.santos@"+clinicPrefix+".local", guardianHash,
	).Error; err != nil {
		return err
	}

	return db.WithContext(ctx).Exec(`
		INSERT INTO patient_guardians (patient_id, legal_guardian_id, relation, can_view_medical_record, can_view_contracts) VALUES
		(?, ?, 'Titular', true, true),
		(?, ?, 'Titular', true, true),
		(?, ?, 'Pai', true, true),
		(?, ?, 'Mãe', true, true)
	`,
		adult1PatientID, adult1GuardianID,
		adult2PatientID, adult2GuardianID,
		child1PatientID, child1GuardianID,
		child2PatientID, child2GuardianID,
	).Error
}
