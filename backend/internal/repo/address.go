package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Address struct {
	ID           uuid.UUID
	Street       *string
	Number       *string
	Complement   *string
	Neighborhood *string
	City         *string
	State        *string
	Country      *string
	Zip          *string
}

func CreateAddress(ctx context.Context, db *gorm.DB, a *Address) (uuid.UUID, error) {
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		INSERT INTO addresses (street, number, complement, neighborhood, city, state, country, zip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, a.Street, a.Number, a.Complement, a.Neighborhood, a.City, a.State, a.Country, a.Zip).Scan(&res).Error
	return res.ID, err
}

// CreateAddressTx insere um endereço dentro de uma transação e retorna o ID.
func CreateAddressTx(ctx context.Context, tx *gorm.DB, a *Address) (uuid.UUID, error) {
	var res struct{ ID uuid.UUID }
	err := tx.WithContext(ctx).Raw(`
		INSERT INTO addresses (street, number, complement, neighborhood, city, state, country, zip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, a.Street, a.Number, a.Complement, a.Neighborhood, a.City, a.State, a.Country, a.Zip).Scan(&res).Error
	return res.ID, err
}

func GetAddressByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*Address, error) {
	var a Address
	err := db.WithContext(ctx).Raw(`
		SELECT id, street, number, complement, neighborhood, city, state, country, zip
		FROM addresses WHERE id = ?
	`, id).Scan(&a).Error
	if err != nil {
		return nil, err
	}
	if a.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &a, nil
}

func UpdateAddress(ctx context.Context, db *gorm.DB, id uuid.UUID, a *Address) error {
	return db.WithContext(ctx).Exec(`
		UPDATE addresses
		SET street = ?, number = ?, complement = ?, neighborhood = ?, city = ?, state = ?, country = ?, zip = ?
		WHERE id = ?
	`, a.Street, a.Number, a.Complement, a.Neighborhood, a.City, a.State, a.Country, a.Zip, id).Error
}

// CleanupOrphanAddresses remove endereços que não são referenciados em legal_guardians, professionals ou patients.
// Retorna o número de linhas removidas.
func CleanupOrphanAddresses(ctx context.Context, db *gorm.DB) (int64, error) {
	result := db.WithContext(ctx).Exec(`
		DELETE FROM addresses
		WHERE id NOT IN (
			SELECT address_id FROM legal_guardians WHERE address_id IS NOT NULL
			UNION
			SELECT address_id FROM professionals WHERE address_id IS NOT NULL
			UNION
			SELECT address_id FROM patients WHERE address_id IS NOT NULL
		)
	`)
	return result.RowsAffected, result.Error
}
