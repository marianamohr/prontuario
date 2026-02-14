package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

func ContractTemplatesByClinic(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID) ([]ContractTemplate, error) {
	rows, err := pool.Query(ctx, `SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates WHERE clinic_id = $1 ORDER BY name`, clinicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ContractTemplate
	for rows.Next() {
		var c ContractTemplate
		var profID *uuid.UUID
		var tipoServico, periodicidade *string
		if err := rows.Scan(&c.ID, &c.ClinicID, &profID, &c.Name, &c.BodyHTML, &tipoServico, &periodicidade, &c.Version); err != nil {
			return nil, err
		}
		c.ProfessionalID = profID
		c.TipoServico = tipoServico
		c.Periodicidade = periodicidade
		list = append(list, c)
	}
	return list, rows.Err()
}

// ContractTemplatesByClinicAndProfessional retorna modelos da clínica que são do profissional ou compartilhados (professional_id NULL).
func ContractTemplatesByClinicAndProfessional(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, professionalID *uuid.UUID) ([]ContractTemplate, error) {
	if professionalID == nil {
		return ContractTemplatesByClinic(ctx, pool, clinicID)
	}
	rows, err := pool.Query(ctx, `
		SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates
		WHERE clinic_id = $1 AND (professional_id = $2 OR professional_id IS NULL) ORDER BY name
	`, clinicID, professionalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ContractTemplate
	for rows.Next() {
		var c ContractTemplate
		var profID *uuid.UUID
		var tipoServico, periodicidade *string
		if err := rows.Scan(&c.ID, &c.ClinicID, &profID, &c.Name, &c.BodyHTML, &tipoServico, &periodicidade, &c.Version); err != nil {
			return nil, err
		}
		c.ProfessionalID = profID
		c.TipoServico = tipoServico
		c.Periodicidade = periodicidade
		list = append(list, c)
	}
	return list, rows.Err()
}

func ContractTemplateByIDAndClinic(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) (*ContractTemplate, error) {
	var c ContractTemplate
	var profID *uuid.UUID
	var tipoServico, periodicidade *string
	err := pool.QueryRow(ctx, `SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates WHERE id = $1 AND clinic_id = $2`, id, clinicID).Scan(&c.ID, &c.ClinicID, &profID, &c.Name, &c.BodyHTML, &tipoServico, &periodicidade, &c.Version)
	c.ProfessionalID = profID
	c.TipoServico = tipoServico
	c.Periodicidade = periodicidade
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func CreateContractTemplate(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, professionalID *uuid.UUID, name, bodyHTML, tipoServico, periodicidade string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `INSERT INTO contract_templates (clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version) VALUES ($1, $2, $3, $4, $5, $6, 1) RETURNING id`, clinicID, professionalID, name, bodyHTML, nullIfEmpty(tipoServico), nullIfEmpty(periodicidade)).Scan(&id)
	return id, err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func UpdateContractTemplate(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID, name, bodyHTML, tipoServico, periodicidade string, version int) error {
	_, err := pool.Exec(ctx, `UPDATE contract_templates SET name = $1, body_html = $2, tipo_servico = $3, periodicidade = $4, version = $5, updated_at = now() WHERE id = $6 AND clinic_id = $7`, name, bodyHTML, nullIfEmpty(tipoServico), nullIfEmpty(periodicidade), version, id, clinicID)
	return err
}

func DeleteContractTemplate(ctx context.Context, pool *pgxpool.Pool, id, clinicID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM contract_templates WHERE id = $1 AND clinic_id = $2`, id, clinicID)
	return err
}

func ContractTemplateByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*ContractTemplate, error) {
	var c ContractTemplate
	var profID *uuid.UUID
	var tipoServico, periodicidade *string
	err := pool.QueryRow(ctx, `SELECT id, clinic_id, professional_id, name, body_html, tipo_servico, periodicidade, version FROM contract_templates WHERE id = $1`, id).Scan(&c.ID, &c.ClinicID, &profID, &c.Name, &c.BodyHTML, &tipoServico, &periodicidade, &c.Version)
	c.ProfessionalID = profID
	c.TipoServico = tipoServico
	c.Periodicidade = periodicidade
	if err != nil {
		return nil, err
	}
	return &c, nil
}
