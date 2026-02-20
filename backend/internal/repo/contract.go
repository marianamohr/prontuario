package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Contract struct {
	ID                uuid.UUID
	ClinicID          uuid.UUID
	PatientID         uuid.UUID
	LegalGuardianID   uuid.UUID
	ProfessionalID    *uuid.UUID
	TemplateID        uuid.UUID
	SignerRelation    string
	SignerIsPatient   bool
	Status            string
	SignedAt          *time.Time
	PDFURL            *string
	PDFSHA256         *string
	AuditJSON         []byte
	TemplateVersion   int
	VerificationToken *string
	StartDate         *time.Time // data de início do contrato (para placeholder [DATA_INICIO])
	EndDate           *time.Time // data de término (para placeholder [DATA_FIM])
	Valor             *string    // valor do serviço (placeholder [VALOR], informado ao disparar)
	Periodicidade     *string    // periodicidade (placeholder [PERIODICIDADE], informada ao disparar)
	SignPlace         *string    // local de assinatura (placeholder [LOCAL])
	SignDate          *time.Time // data prevista para assinatura (exibida ao responsável; na assinatura usa-se a data real)
	NumAppointments   *int       // quantidade de agendamentos a criar ao assinar (nil = sem limite)
}

func ContractsByClinic(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) ([]Contract, error) {
	list, _, err := ContractsByClinicPaginated(ctx, db, clinicID, 0, 0)
	return list, err
}

func ContractsByClinicPaginated(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, limit, offset int) ([]Contract, int, error) {
	var total int
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM contracts WHERE clinic_id = ? AND deleted_at IS NULL`, clinicID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	q := `
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE clinic_id = ? AND deleted_at IS NULL ORDER BY created_at DESC
	`
	args := []interface{}{clinicID}
	if limit > 0 {
		q += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	var list []Contract
	err := db.WithContext(ctx).Raw(q, args...).Scan(&list).Error
	return list, total, err
}

func ContractByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*Contract, error) {
	var c Contract
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE id = ?
	`, id).Scan(&c).Error
	if err != nil {
		return nil, err
	}
	if c.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &c, nil
}

func ContractByIDAndClinic(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) (*Contract, error) {
	var c Contract
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE id = ? AND clinic_id = ? AND deleted_at IS NULL
	`, id, clinicID).Scan(&c).Error
	if err != nil {
		return nil, err
	}
	if c.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &c, nil
}

func ContractByAccessToken(ctx context.Context, db *gorm.DB, token string) (*Contract, *ContractTemplate, *Patient, *LegalGuardian, error) {
	var res struct{ ContractID uuid.UUID }
	err := db.WithContext(ctx).Raw(`SELECT contract_id FROM contract_access_tokens WHERE token = ? AND expires_at > now() AND used_at IS NULL`, token).Scan(&res).Error
	if err != nil || res.ContractID == uuid.Nil {
		return nil, nil, nil, nil, err
	}
	c, err := ContractByID(ctx, db, res.ContractID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	tpl, err := ContractTemplateByID(ctx, db, c.TemplateID)
	if err != nil {
		return c, nil, nil, nil, err
	}
	patient, err := PatientByID(ctx, db, c.PatientID)
	if err != nil {
		return c, tpl, nil, nil, err
	}
	guardian, err := LegalGuardianByID(ctx, db, c.LegalGuardianID)
	if err != nil {
		return c, tpl, patient, nil, err
	}
	return c, tpl, patient, guardian, nil
}

func CreateContract(ctx context.Context, db *gorm.DB, clinicID, patientID, legalGuardianID uuid.UUID, professionalID *uuid.UUID, templateID uuid.UUID, signerRelation string, signerIsPatient bool, templateVersion int, startDate, endDate *time.Time, valor, periodicidade *string, signPlace *string, signDate *time.Time, numAppointments *int) (uuid.UUID, error) {
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		INSERT INTO contracts (clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, template_version, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id
	`, clinicID, patientID, legalGuardianID, professionalID, templateID, signerRelation, signerIsPatient, templateVersion, startDate, endDate, valor, periodicidade, signPlace, signDate, numAppointments).Scan(&res).Error
	return res.ID, err
}

func CreateContractAccessToken(ctx context.Context, db *gorm.DB, contractID uuid.UUID, exp time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	return token, db.WithContext(ctx).Exec(`INSERT INTO contract_access_tokens (contract_id, token, expires_at) VALUES (?, ?, ?)`, contractID, token, time.Now().Add(exp)).Error
}

func MarkContractAccessTokenUsed(ctx context.Context, db *gorm.DB, token string) error {
	return db.WithContext(ctx).Exec(`UPDATE contract_access_tokens SET used_at = now() WHERE token = ?`, token).Error
}

func SignContract(ctx context.Context, db *gorm.DB, contractID uuid.UUID, pdfSHA256, verificationToken string, auditJSON []byte) error {
	return db.WithContext(ctx).Exec(`
		UPDATE contracts SET status = 'SIGNED', signed_at = now(), pdf_sha256 = ?, verification_token = ?, audit_json = ?, updated_at = now()
		WHERE id = ? AND deleted_at IS NULL
	`, pdfSHA256, verificationToken, auditJSON, contractID).Error
}

func ContractByVerificationToken(ctx context.Context, db *gorm.DB, verificationToken string) (*Contract, error) {
	var c Contract
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE verification_token = ? AND deleted_at IS NULL
	`, verificationToken).Scan(&c).Error
	if err != nil {
		return nil, err
	}
	if c.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &c, nil
}

// ContractForAgenda é um contrato assinado com dados para o dropdown da Agenda (criar agendamentos).
type ContractForAgenda struct {
	ID             uuid.UUID
	PatientID      uuid.UUID
	PatientName    string
	TemplateName   string
	ProfessionalID *uuid.UUID
}

// SignedContractsByClinicWithDetails retorna contratos SIGNED da clínica com nome do paciente e do modelo.
func SignedContractsByClinicWithDetails(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) ([]ContractForAgenda, error) {
	var list []ContractForAgenda
	err := db.WithContext(ctx).Raw(`
		SELECT c.id, c.patient_id, p.full_name AS patient_name, t.name AS template_name, c.professional_id
		FROM contracts c
		JOIN patients p ON p.id = c.patient_id
		JOIN contract_templates t ON t.id = c.template_id
		WHERE c.clinic_id = ? AND c.status = 'SIGNED' AND c.deleted_at IS NULL AND p.deleted_at IS NULL
		ORDER BY c.signed_at DESC
	`, clinicID).Scan(&list).Error
	return list, err
}

// PatientContractItem representa um contrato do paciente com dados para exibição na lista.
type PatientContractItem struct {
	ID                uuid.UUID
	LegalGuardianID   uuid.UUID
	Status            string
	SignedAt          *time.Time
	VerificationToken *string
	PDFURL            *string
	TemplateName      string
	GuardianName      string
	GuardianEmail     string
}

// PendingContractItem é um contrato pendente de assinatura com dados para exibição na home.
type PendingContractItem struct {
	ID           uuid.UUID
	PatientID    uuid.UUID
	PatientName  string
	TemplateName string
	GuardianName string
}

// PendingContractsByClinic retorna os contratos PENDING da clínica (para a home: "contratos que faltam assinar").
func PendingContractsByClinic(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) ([]PendingContractItem, error) {
	var list []PendingContractItem
	err := db.WithContext(ctx).Raw(`
		SELECT c.id, c.patient_id, p.full_name AS patient_name, t.name AS template_name, g.full_name AS guardian_name
		FROM contracts c
		JOIN contract_templates t ON t.id = c.template_id
		JOIN legal_guardians g ON g.id = c.legal_guardian_id
		JOIN patients p ON p.id = c.patient_id
		WHERE c.clinic_id = ? AND c.status = 'PENDING' AND c.deleted_at IS NULL AND p.deleted_at IS NULL AND g.deleted_at IS NULL
		ORDER BY c.created_at DESC
	`, clinicID).Scan(&list).Error
	return list, err
}

// ContractsByPatientAndClinic retorna os contratos do paciente na clínica, com nome do modelo e do responsável.
func ContractsByPatientAndClinic(ctx context.Context, db *gorm.DB, patientID, clinicID uuid.UUID) ([]PatientContractItem, error) {
	list, _, err := ContractsByPatientAndClinicPaginated(ctx, db, patientID, clinicID, 0, 0)
	return list, err
}

// ContractsByPatientAndClinicPaginated retorna os contratos do paciente com limit/offset.
func ContractsByPatientAndClinicPaginated(ctx context.Context, db *gorm.DB, patientID, clinicID uuid.UUID, limit, offset int) ([]PatientContractItem, int, error) {
	var total int
	if err := db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM contracts c
		JOIN legal_guardians g ON g.id = c.legal_guardian_id
		WHERE c.patient_id = ? AND c.clinic_id = ? AND c.deleted_at IS NULL AND g.deleted_at IS NULL
	`, patientID, clinicID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	q := `
		SELECT c.id, c.legal_guardian_id, c.status, c.signed_at, c.verification_token, c.pdf_url,
		       t.name AS template_name, g.full_name AS guardian_name, g.email AS guardian_email
		FROM contracts c
		JOIN contract_templates t ON t.id = c.template_id
		JOIN legal_guardians g ON g.id = c.legal_guardian_id
		WHERE c.patient_id = ? AND c.clinic_id = ? AND c.deleted_at IS NULL AND g.deleted_at IS NULL
		ORDER BY c.created_at DESC
	`
	args := []interface{}{patientID, clinicID}
	if limit > 0 {
		q += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	var list []PatientContractItem
	err := db.WithContext(ctx).Raw(q, args...).Scan(&list).Error
	return list, total, err
}

// CancelOtherPendingContractsForPatientAndGuardian marca como CANCELLED os demais contratos PENDING do mesmo paciente e responsável (exceto o id indicado).
func CancelOtherPendingContractsForPatientAndGuardian(ctx context.Context, db *gorm.DB, contractID, patientID, legalGuardianID uuid.UUID) error {
	return db.WithContext(ctx).Exec(`
		UPDATE contracts SET status = 'CANCELLED', updated_at = now()
		WHERE patient_id = ? AND legal_guardian_id = ? AND status = 'PENDING' AND id != ?
	`, patientID, legalGuardianID, contractID).Error
}

// CancelContract marca o contrato como CANCELLED (inativo). Permite cancelar contratos PENDING ou SIGNED.
func CancelContract(ctx context.Context, db *gorm.DB, contractID, clinicID uuid.UUID) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE contracts SET status = 'CANCELLED', updated_at = now() WHERE id = ? AND clinic_id = ? AND status IN ('PENDING', 'SIGNED') AND deleted_at IS NULL
	`, contractID, clinicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SetContractEndDate define a data de término do contrato e marca como ENDED (encerramento: serviço prestado até essa data).
func SetContractEndDate(ctx context.Context, db *gorm.DB, contractID, clinicID uuid.UUID, endDate time.Time) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE contracts SET end_date = ?, status = 'ENDED', updated_at = now() WHERE id = ? AND clinic_id = ? AND status = 'SIGNED' AND deleted_at IS NULL
	`, endDate, contractID, clinicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func SoftDeleteContract(ctx context.Context, db *gorm.DB, contractID, clinicID uuid.UUID) error {
	result := db.WithContext(ctx).Exec(`
		UPDATE contracts SET deleted_at = now(), updated_at = now() WHERE id = ? AND clinic_id = ? AND deleted_at IS NULL
	`, contractID, clinicID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
