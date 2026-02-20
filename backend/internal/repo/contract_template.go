package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContractTemplate struct {
	ID             uuid.UUID
	ClinicID       uuid.UUID
	ProfessionalID *uuid.UUID
	Name           string
	BodyHTML       string
	TipoServico    *string
	Periodicidade  *string
	Version        int
}

func ContractTemplatesByClinic(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) ([]ContractTemplate, error) {
	var list []ContractTemplate
	err := db.WithContext(ctx).Raw(`SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates WHERE clinic_id = ? ORDER BY name`, clinicID).Scan(&list).Error
	return list, err
}

// ContractTemplatesByClinicAndProfessional retorna modelos da clínica que são do profissional ou compartilhados (professional_id NULL).
func ContractTemplatesByClinicAndProfessional(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, professionalID *uuid.UUID) ([]ContractTemplate, error) {
	if professionalID == nil {
		return ContractTemplatesByClinic(ctx, db, clinicID)
	}
	var list []ContractTemplate
	err := db.WithContext(ctx).Raw(`
		SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates
		WHERE clinic_id = ? AND (professional_id = ? OR professional_id IS NULL) ORDER BY name
	`, clinicID, professionalID).Scan(&list).Error
	return list, err
}

func ContractTemplateByIDAndClinic(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) (*ContractTemplate, error) {
	var c ContractTemplate
	err := db.WithContext(ctx).Raw(`SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates WHERE id = ? AND clinic_id = ?`, id, clinicID).Scan(&c).Error
	if err != nil {
		return nil, err
	}
	if c.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &c, nil
}

func CreateContractTemplate(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, professionalID *uuid.UUID, name, bodyHTML, tipoServico, periodicidade string) (uuid.UUID, error) {
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`INSERT INTO contract_templates (clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version) VALUES (?, ?, ?, ?, ?, ?, 1) RETURNING id`, clinicID, professionalID, name, bodyHTML, nullIfEmpty(tipoServico), nullIfEmpty(periodicidade)).Scan(&res).Error
	return res.ID, err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func UpdateContractTemplate(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID, name, bodyHTML, tipoServico, periodicidade string, version int) error {
	return db.WithContext(ctx).Exec(`UPDATE contract_templates SET name = ?, body_html = ?, tipo_servico = ?, periodicidade = ?, version = ?, updated_at = now() WHERE id = ? AND clinic_id = ?`, name, bodyHTML, nullIfEmpty(tipoServico), nullIfEmpty(periodicidade), version, id, clinicID).Error
}

func DeleteContractTemplate(ctx context.Context, db *gorm.DB, id, clinicID uuid.UUID) error {
	return db.WithContext(ctx).Exec(`DELETE FROM contract_templates WHERE id = ? AND clinic_id = ?`, id, clinicID).Error
}

func ContractTemplateByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*ContractTemplate, error) {
	var c ContractTemplate
	err := db.WithContext(ctx).Raw(`SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates WHERE id = ?`, id).Scan(&c).Error
	if err != nil {
		return nil, err
	}
	if c.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &c, nil
}
