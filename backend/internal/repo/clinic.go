package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func ClinicByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*Clinic, error) {
	var c Clinic
	err := db.WithContext(ctx).Raw(`
		SELECT id, name, primary_color, background_color, home_label, home_image_url, action_button_color, negation_button_color
		FROM clinics WHERE id = ?
	`, id).Scan(&c).Error
	if err != nil {
		return nil, err
	}
	if c.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &c, nil
}

func GetClinicBranding(ctx context.Context, db *gorm.DB, clinicID uuid.UUID) (*ClinicBranding, error) {
	var b ClinicBranding
	err := db.WithContext(ctx).Raw(`
		SELECT primary_color, background_color, home_label, home_image_url, action_button_color, negation_button_color
		FROM clinics WHERE id = ?
	`, clinicID).Scan(&b).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func UpdateClinicBranding(ctx context.Context, db *gorm.DB, clinicID uuid.UUID, b *ClinicBranding) error {
	return db.WithContext(ctx).Exec(`
		UPDATE clinics SET
			primary_color = ?, background_color = ?, home_label = ?, home_image_url = ?,
			action_button_color = ?, negation_button_color = ?,
			updated_at = now()
		WHERE id = ?
	`, b.PrimaryColor, b.BackgroundColor, b.HomeLabel, b.HomeImageURL, b.ActionButtonColor, b.NegationButtonColor, clinicID).Error
}
