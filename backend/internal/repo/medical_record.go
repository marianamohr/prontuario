package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetOrCreateMedicalRecord(ctx context.Context, pool *pgxpool.Pool, patientID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM medical_records WHERE patient_id = $1`, patientID).Scan(&id)
	if err == nil {
		return id, nil
	}
	err = pool.QueryRow(ctx, `INSERT INTO medical_records (patient_id) VALUES ($1) ON CONFLICT (patient_id) DO UPDATE SET updated_at = now() RETURNING id`, patientID).Scan(&id)
	if err != nil {
		return id, err
	}
	return id, nil
}

type RecordEntry struct {
	ID               uuid.UUID
	MedicalRecordID  uuid.UUID
	ContentEncrypted []byte
	ContentNonce     []byte
	ContentKeyVersion string
	EntryDate        time.Time
	AuthorID         uuid.UUID
	AuthorType       string
	CreatedAt        time.Time
}

func RecordEntriesByMedicalRecord(ctx context.Context, pool *pgxpool.Pool, medicalRecordID uuid.UUID) ([]RecordEntry, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, medical_record_id, content_encrypted, content_nonce, content_key_version, entry_date, author_id, author_type, created_at
		FROM record_entries WHERE medical_record_id = $1 ORDER BY entry_date DESC, created_at DESC
	`, medicalRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []RecordEntry
	for rows.Next() {
		var e RecordEntry
		if err := rows.Scan(&e.ID, &e.MedicalRecordID, &e.ContentEncrypted, &e.ContentNonce, &e.ContentKeyVersion, &e.EntryDate, &e.AuthorID, &e.AuthorType, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func CreateRecordEntry(ctx context.Context, pool *pgxpool.Pool, medicalRecordID uuid.UUID, contentEncrypted, contentNonce []byte, contentKeyVersion string, entryDate time.Time, authorID uuid.UUID, authorType string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO record_entries (medical_record_id, content_encrypted, content_nonce, content_key_version, entry_date, author_id, author_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, medicalRecordID, contentEncrypted, contentNonce, contentKeyVersion, entryDate, authorID, authorType).Scan(&id)
	return id, err
}

func RecordEntryByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*RecordEntry, error) {
	var e RecordEntry
	err := pool.QueryRow(ctx, `
		SELECT id, medical_record_id, content_encrypted, content_nonce, content_key_version, entry_date, author_id, author_type, created_at
		FROM record_entries WHERE id = $1
	`, id).Scan(&e.ID, &e.MedicalRecordID, &e.ContentEncrypted, &e.ContentNonce, &e.ContentKeyVersion, &e.EntryDate, &e.AuthorID, &e.AuthorType, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func PatientIDByMedicalRecordID(ctx context.Context, pool *pgxpool.Pool, medicalRecordID uuid.UUID) (uuid.UUID, error) {
	var patientID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT patient_id FROM medical_records WHERE id = $1`, medicalRecordID).Scan(&patientID)
	return patientID, err
}
