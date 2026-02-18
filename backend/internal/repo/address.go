package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Address struct {
	ID          uuid.UUID
	Street      *string
	Number      *string
	Complement  *string
	Neighborhood *string
	City        *string
	State       *string
	Country     *string
	Zip         *string
}

func CreateAddress(ctx context.Context, pool *pgxpool.Pool, a *Address) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO addresses (street, number, complement, neighborhood, city, state, country, zip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, a.Street, a.Number, a.Complement, a.Neighborhood, a.City, a.State, a.Country, a.Zip).Scan(&id)
	return id, err
}

// CreateAddressTx insere um endereço dentro de uma transação e retorna o ID.
func CreateAddressTx(ctx context.Context, tx pgx.Tx, a *Address) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO addresses (street, number, complement, neighborhood, city, state, country, zip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, a.Street, a.Number, a.Complement, a.Neighborhood, a.City, a.State, a.Country, a.Zip).Scan(&id)
	return id, err
}

func GetAddressByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Address, error) {
	var a Address
	err := pool.QueryRow(ctx, `
		SELECT id, street, number, complement, neighborhood, city, state, country, zip
		FROM addresses WHERE id = $1
	`, id).Scan(&a.ID, &a.Street, &a.Number, &a.Complement, &a.Neighborhood, &a.City, &a.State, &a.Country, &a.Zip)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func UpdateAddress(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, a *Address) error {
	_, err := pool.Exec(ctx, `
		UPDATE addresses
		SET street = $1, number = $2, complement = $3, neighborhood = $4, city = $5, state = $6, country = $7, zip = $8
		WHERE id = $9
	`, a.Street, a.Number, a.Complement, a.Neighborhood, a.City, a.State, a.Country, a.Zip, id)
	return err
}

// CleanupOrphanAddresses remove endereços que não são referenciados em legal_guardians, professionals ou patients.
// Retorna o número de linhas removidas.
func CleanupOrphanAddresses(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	result, err := pool.Exec(ctx, `
		DELETE FROM addresses
		WHERE id NOT IN (
			SELECT address_id FROM legal_guardians WHERE address_id IS NOT NULL
			UNION
			SELECT address_id FROM professionals WHERE address_id IS NOT NULL
			UNION
			SELECT address_id FROM patients WHERE address_id IS NOT NULL
		)
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
