package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Clinic struct {
	ID                  uuid.UUID
	Name                string
	PrimaryColor        *string
	BackgroundColor     *string
	HomeLabel           *string
	HomeImageURL        *string
	ActionButtonColor   *string
	NegationButtonColor *string
}

type ClinicBranding struct {
	PrimaryColor        *string
	BackgroundColor     *string
	HomeLabel           *string
	HomeImageURL        *string
	ActionButtonColor   *string
	NegationButtonColor *string
}

func ClinicByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Clinic, error) {
	var c Clinic
	err := pool.QueryRow(ctx, `
		SELECT id, name, primary_color, background_color, home_label, home_image_url, action_button_color, negation_button_color
		FROM clinics WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.PrimaryColor, &c.BackgroundColor, &c.HomeLabel, &c.HomeImageURL, &c.ActionButtonColor, &c.NegationButtonColor)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func GetClinicBranding(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID) (*ClinicBranding, error) {
	var b ClinicBranding
	err := pool.QueryRow(ctx, `
		SELECT primary_color, background_color, home_label, home_image_url, action_button_color, negation_button_color
		FROM clinics WHERE id = $1
	`, clinicID).Scan(&b.PrimaryColor, &b.BackgroundColor, &b.HomeLabel, &b.HomeImageURL, &b.ActionButtonColor, &b.NegationButtonColor)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func UpdateClinicBranding(ctx context.Context, pool *pgxpool.Pool, clinicID uuid.UUID, b *ClinicBranding) error {
	_, err := pool.Exec(ctx, `
		UPDATE clinics SET
			primary_color = $1, background_color = $2, home_label = $3, home_image_url = $4,
			action_button_color = $5, negation_button_color = $6,
			updated_at = now()
		WHERE id = $7
	`, b.PrimaryColor, b.BackgroundColor, b.HomeLabel, b.HomeImageURL, b.ActionButtonColor, b.NegationButtonColor, clinicID)
	return err
}
