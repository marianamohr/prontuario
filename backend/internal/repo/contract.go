package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func scanContractRow(c *Contract, profID *uuid.UUID, signedAt *time.Time, pdfURL, pdfSHA, verTok *string, audit []byte, startDate, endDate *time.Time, valor, periodicidade *string, signPlace *string, signDate *time.Time, numAppointments *int) {
	c.ProfessionalID = profID
	c.SignedAt = signedAt
	c.PDFURL = pdfURL
	c.PDFSHA256 = pdfSHA
	c.AuditJSON = audit
	c.VerificationToken = verTok
	c.StartDate = startDate
	c.EndDate = endDate
	c.Valor = valor
	c.Periodicidade = periodicidade
	c.SignPlace = signPlace
	c.SignDate = signDate
	c.NumAppointments = numAppointments
}

func ContractsByClinic(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID) ([]Contract, error) {
	list, _, err := ContractsByClinicPaginated(ctx, pool, clinicID, 0, 0)
	return list, err
}

func ContractsByClinicPaginated(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, limit, offset int) ([]Contract, int, error) {
	var total int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM contracts WHERE clinic_id = $1 AND deleted_at IS NULL`, clinicID).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE clinic_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC
	`
	args := []interface{}{clinicID}
	if limit > 0 {
		q += ` LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []Contract
	for rows.Next() {
		var c Contract
		var profID *uuid.UUID
		var signedAt *time.Time
		var pdfURL, pdfSHA, verTok *string
		var audit []byte
		var startDate, endDate *time.Time
		var valor, periodicidade, signPlace *string
		var signDate *time.Time
		var numAppointments *int
		if err := rows.Scan(&c.ID, &c.ClinicID, &c.PatientID, &c.LegalGuardianID, &profID, &c.TemplateID, &c.SignerRelation, &c.SignerIsPatient, &c.Status, &signedAt, &pdfURL, &pdfSHA, &audit, &c.TemplateVersion, &verTok, &startDate, &endDate, &valor, &periodicidade, &signPlace, &signDate, &numAppointments); err != nil {
			return nil, 0, err
		}
		scanContractRow(&c, profID, signedAt, pdfURL, pdfSHA, verTok, audit, startDate, endDate, valor, periodicidade, signPlace, signDate, numAppointments)
		list = append(list, c)
	}
	return list, total, rows.Err()
}

func ContractByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Contract, error) {
	var c Contract
	var profID *uuid.UUID
	var signedAt *time.Time
	var pdfURL, pdfSHA, verTok *string
	var audit []byte
	var startDate, endDate *time.Time
	var valor, periodicidade, signPlace *string
	var signDate *time.Time
	var numAppointments *int
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE id = $1
	`, id).Scan(&c.ID, &c.ClinicID, &c.PatientID, &c.LegalGuardianID, &profID, &c.TemplateID, &c.SignerRelation, &c.SignerIsPatient, &c.Status, &signedAt, &pdfURL, &pdfSHA, &audit, &c.TemplateVersion, &verTok, &startDate, &endDate, &valor, &periodicidade, &signPlace, &signDate, &numAppointments)
	if err != nil {
		return nil, err
	}
	scanContractRow(&c, profID, signedAt, pdfURL, pdfSHA, verTok, audit, startDate, endDate, valor, periodicidade, signPlace, signDate, numAppointments)
	return &c, nil
}

func ContractByIDAndClinic(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) (*Contract, error) {
	var c Contract
	var profID *uuid.UUID
	var signedAt *time.Time
	var pdfURL, pdfSHA, verTok *string
	var audit []byte
	var startDate, endDate *time.Time
	var valor, periodicidade, signPlace *string
	var signDate *time.Time
	var numAppointments *int
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE id = $1 AND clinic_id = $2 AND deleted_at IS NULL
	`, id, clinicID).Scan(&c.ID, &c.ClinicID, &c.PatientID, &c.LegalGuardianID, &profID, &c.TemplateID, &c.SignerRelation, &c.SignerIsPatient, &c.Status, &signedAt, &pdfURL, &pdfSHA, &audit, &c.TemplateVersion, &verTok, &startDate, &endDate, &valor, &periodicidade, &signPlace, &signDate, &numAppointments)
	if err != nil {
		return nil, err
	}
	scanContractRow(&c, profID, signedAt, pdfURL, pdfSHA, verTok, audit, startDate, endDate, valor, periodicidade, signPlace, signDate, numAppointments)
	return &c, nil
}

func ContractByAccessToken(ctx context.Context, pool *pgxpool.Pool, token string) (*Contract, *ContractTemplate, *Patient, *LegalGuardian, error) {
	var contractID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT contract_id FROM contract_access_tokens WHERE token = $1 AND expires_at > now() AND used_at IS NULL`, token).Scan(&contractID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var c Contract
	var profID *uuid.UUID
	var signedAt *time.Time
	var pdfURL, pdfSHA, verTok *string
	var audit []byte
	var startDate, endDate *time.Time
	var valor, periodicidade, signPlace *string
	var signDate *time.Time
	var numAppointments *int
	err = pool.QueryRow(ctx, `
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE id = $1 AND deleted_at IS NULL
	`, contractID).Scan(&c.ID, &c.ClinicID, &c.PatientID, &c.LegalGuardianID, &profID, &c.TemplateID, &c.SignerRelation, &c.SignerIsPatient, &c.Status, &signedAt, &pdfURL, &pdfSHA, &audit, &c.TemplateVersion, &verTok, &startDate, &endDate, &valor, &periodicidade, &signPlace, &signDate, &numAppointments)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	scanContractRow(&c, profID, signedAt, pdfURL, pdfSHA, verTok, audit, startDate, endDate, valor, periodicidade, signPlace, signDate, numAppointments)
	tpl, err := ContractTemplateByID(ctx, pool, c.TemplateID)
	if err != nil {
		return &c, nil, nil, nil, err
	}
	patient, err := PatientByID(ctx, pool, c.PatientID)
	if err != nil {
		return &c, tpl, nil, nil, err
	}
	guardian, err := LegalGuardianByID(ctx, pool, c.LegalGuardianID)
	if err != nil {
		return &c, tpl, patient, nil, err
	}
	return &c, tpl, patient, guardian, nil
}

func CreateContract(ctx context.Context, pool *pgxpool.Pool, clinicID, patientID, legalGuardianID uuid.UUID, professionalID *uuid.UUID, templateID uuid.UUID, signerRelation string, signerIsPatient bool, templateVersion int, startDate, endDate *time.Time, valor, periodicidade *string, signPlace *string, signDate *time.Time, numAppointments *int) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO contracts (clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, template_version, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id
	`, clinicID, patientID, legalGuardianID, professionalID, templateID, signerRelation, signerIsPatient, templateVersion, startDate, endDate, valor, periodicidade, signPlace, signDate, numAppointments).Scan(&id)
	return id, err
}

func CreateContractAccessToken(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID, exp time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	_, err = pool.Exec(ctx, `INSERT INTO contract_access_tokens (contract_id, token, expires_at) VALUES ($1, $2, $3)`, contractID, token, time.Now().Add(exp))
	return token, err
}

func MarkContractAccessTokenUsed(ctx context.Context, pool *pgxpool.Pool, token string) error {
	_, err := pool.Exec(ctx, `UPDATE contract_access_tokens SET used_at = now() WHERE token = $1`, token)
	return err
}

func SignContract(ctx context.Context, pool *pgxpool.Pool, contractID uuid.UUID, pdfSHA256, verificationToken string, auditJSON []byte) error {
	_, err := pool.Exec(ctx, `
		UPDATE contracts SET status = 'SIGNED', signed_at = now(), pdf_sha256 = $1, verification_token = $2, audit_json = $3, updated_at = now()
		WHERE id = $4 AND deleted_at IS NULL
	`, pdfSHA256, verificationToken, auditJSON, contractID)
	return err
}

func ContractByVerificationToken(ctx context.Context, pool *pgxpool.Pool, verificationToken string) (*Contract, error) {
	var c Contract
	var profID *uuid.UUID
	var signedAt *time.Time
	var pdfURL, pdfSHA, verTok *string
	var audit []byte
	var startDate, endDate *time.Time
	var valor, periodicidade, signPlace *string
	var signDate *time.Time
	var numAppointments *int
	err := pool.QueryRow(ctx, `
		SELECT id, clinic_id, patient_id, legal_guardian_id, professional_id, template_id, signer_relation, signer_is_patient, status, signed_at, pdf_url, pdf_sha256, audit_json, template_version, verification_token, start_date, end_date, valor, periodicidade, sign_place, sign_date, num_appointments
		FROM contracts WHERE verification_token = $1 AND deleted_at IS NULL
	`, verificationToken).Scan(&c.ID, &c.ClinicID, &c.PatientID, &c.LegalGuardianID, &profID, &c.TemplateID, &c.SignerRelation, &c.SignerIsPatient, &c.Status, &signedAt, &pdfURL, &pdfSHA, &audit, &c.TemplateVersion, &verTok, &startDate, &endDate, &valor, &periodicidade, &signPlace, &signDate, &numAppointments)
	if err != nil {
		return nil, err
	}
	scanContractRow(&c, profID, signedAt, pdfURL, pdfSHA, verTok, audit, startDate, endDate, valor, periodicidade, signPlace, signDate, numAppointments)
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
func SignedContractsByClinicWithDetails(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID) ([]ContractForAgenda, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.patient_id, p.full_name AS patient_name, t.name AS template_name, c.professional_id
		FROM contracts c
		JOIN patients p ON p.id = c.patient_id
		JOIN contract_templates t ON t.id = c.template_id
		WHERE c.clinic_id = $1 AND c.status = 'SIGNED' AND c.deleted_at IS NULL AND p.deleted_at IS NULL
		ORDER BY c.signed_at DESC
	`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ContractForAgenda
	for rows.Next() {
		var item ContractForAgenda
		var profID *uuid.UUID
		if err := rows.Scan(&item.ID, &item.PatientID, &item.PatientName, &item.TemplateName, &profID); err != nil {
			return nil, err
		}
		item.ProfessionalID = profID
		list = append(list, item)
	}
	return list, rows.Err()
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
func PendingContractsByClinic(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID) ([]PendingContractItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.patient_id, p.full_name AS patient_name, t.name AS template_name, g.full_name AS guardian_name
		FROM contracts c
		JOIN contract_templates t ON t.id = c.template_id
		JOIN legal_guardians g ON g.id = c.legal_guardian_id
		JOIN patients p ON p.id = c.patient_id
		WHERE c.clinic_id = $1 AND c.status = 'PENDING' AND c.deleted_at IS NULL AND p.deleted_at IS NULL AND g.deleted_at IS NULL
		ORDER BY c.created_at DESC
	`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PendingContractItem
	for rows.Next() {
		var item PendingContractItem
		if err := rows.Scan(&item.ID, &item.PatientID, &item.PatientName, &item.TemplateName, &item.GuardianName); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ContractsByPatientAndClinic retorna os contratos do paciente na clínica, com nome do modelo e do responsável.
func ContractsByPatientAndClinic(ctx context.Context, pool *pgxpool.Pool, patientID, clinicID uuid.UUID) ([]PatientContractItem, error) {
	list, _, err := ContractsByPatientAndClinicPaginated(ctx, pool, patientID, clinicID, 0, 0)
	return list, err
}

// ContractsByPatientAndClinicPaginated retorna os contratos do paciente com limit/offset.
func ContractsByPatientAndClinicPaginated(ctx context.Context, pool *pgxpool.Pool, patientID, clinicID uuid.UUID, limit, offset int) ([]PatientContractItem, int, error) {
	var total int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM contracts c
		JOIN legal_guardians g ON g.id = c.legal_guardian_id
		WHERE c.patient_id = $1 AND c.clinic_id = $2 AND c.deleted_at IS NULL AND g.deleted_at IS NULL
	`, patientID, clinicID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	q := `
		SELECT c.id, c.legal_guardian_id, c.status, c.signed_at, c.verification_token, c.pdf_url,
		       t.name AS template_name, g.full_name AS guardian_name, g.email AS guardian_email
		FROM contracts c
		JOIN contract_templates t ON t.id = c.template_id
		JOIN legal_guardians g ON g.id = c.legal_guardian_id
		WHERE c.patient_id = $1 AND c.clinic_id = $2 AND c.deleted_at IS NULL AND g.deleted_at IS NULL
		ORDER BY c.created_at DESC
	`
	args := []interface{}{patientID, clinicID}
	if limit > 0 {
		q += ` LIMIT $3 OFFSET $4`
		args = append(args, limit, offset)
	}
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []PatientContractItem
	for rows.Next() {
		var item PatientContractItem
		var signedAt *time.Time
		var verTok, pdfURL *string
		if err := rows.Scan(&item.ID, &item.LegalGuardianID, &item.Status, &signedAt, &verTok, &pdfURL, &item.TemplateName, &item.GuardianName, &item.GuardianEmail); err != nil {
			return nil, 0, err
		}
		item.SignedAt = signedAt
		item.VerificationToken = verTok
		item.PDFURL = pdfURL
		list = append(list, item)
	}
	return list, total, rows.Err()
}

// CancelOtherPendingContractsForPatientAndGuardian marca como CANCELLED os demais contratos PENDING do mesmo paciente e responsável (exceto o id indicado).
func CancelOtherPendingContractsForPatientAndGuardian(ctx context.Context, pool *pgxpool.Pool, contractID, patientID, legalGuardianID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE contracts SET status = 'CANCELLED', updated_at = now()
		WHERE patient_id = $1 AND legal_guardian_id = $2 AND status = 'PENDING' AND id != $3
	`, patientID, legalGuardianID, contractID)
	return err
}

// CancelContract marca o contrato como CANCELLED (inativo). Permite cancelar contratos PENDING ou SIGNED.
func CancelContract(ctx context.Context, pool *pgxpool.Pool, contractID, clinicID uuid.UUID) error {
	result, err := pool.Exec(ctx, `
		UPDATE contracts SET status = 'CANCELLED', updated_at = now() WHERE id = $1 AND clinic_id = $2 AND status IN ('PENDING', 'SIGNED') AND deleted_at IS NULL
	`, contractID, clinicID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SetContractEndDate define a data de término do contrato e marca como ENDED (encerramento: serviço prestado até essa data).
func SetContractEndDate(ctx context.Context, pool *pgxpool.Pool, contractID, clinicID uuid.UUID, endDate time.Time) error {
	result, err := pool.Exec(ctx, `
		UPDATE contracts SET end_date = $1, status = 'ENDED', updated_at = now() WHERE id = $2 AND clinic_id = $3 AND status = 'SIGNED' AND deleted_at IS NULL
	`, endDate, contractID, clinicID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func SoftDeleteContract(ctx context.Context, pool *pgxpool.Pool, contractID, clinicID uuid.UUID) error {
	result, err := pool.Exec(ctx, `
		UPDATE contracts SET deleted_at = now(), updated_at = now() WHERE id = $1 AND clinic_id = $2 AND deleted_at IS NULL
	`, contractID, clinicID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
