package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetOrCreateMedicalRecord(ctx context.Context, db *gorm.DB, patientID uuid.UUID) (uuid.UUID, error) {
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`SELECT id FROM medical_records WHERE patient_id = ?`, patientID).Scan(&res).Error
	if err == nil && res.ID != uuid.Nil {
		return res.ID, nil
	}
	err = db.WithContext(ctx).Raw(`INSERT INTO medical_records (patient_id) VALUES (?) ON CONFLICT (patient_id) DO UPDATE SET updated_at = now() RETURNING id`, patientID).Scan(&res).Error
	if err != nil {
		return res.ID, err
	}
	return res.ID, nil
}

type RecordEntry struct {
	ID                uuid.UUID
	MedicalRecordID   uuid.UUID
	ContentEncrypted  []byte
	ContentNonce      []byte
	ContentKeyVersion string
	EntryDate         time.Time
	AuthorID          uuid.UUID
	AuthorType        string
	CreatedAt         time.Time
}

func RecordEntriesByMedicalRecord(ctx context.Context, db *gorm.DB, medicalRecordID uuid.UUID) ([]RecordEntry, error) {
	list, _, err := RecordEntriesByMedicalRecordPaginated(ctx, db, medicalRecordID, 0, 0)
	return list, err
}

// RecordEntriesByMedicalRecordPaginated returns record entries with limit/offset. If limit is 0, no limit.
func RecordEntriesByMedicalRecordPaginated(ctx context.Context, db *gorm.DB, medicalRecordID uuid.UUID, limit, offset int) ([]RecordEntry, int, error) {
	var total int
	if err := db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM record_entries WHERE medical_record_id = ?`, medicalRecordID).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	q := `
		SELECT id, medical_record_id, content_encrypted, content_nonce, content_key_version, entry_date, author_id, author_type, created_at
		FROM record_entries WHERE medical_record_id = ? ORDER BY entry_date DESC, created_at DESC
	`
	args := []interface{}{medicalRecordID}
	if limit > 0 {
		q += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	var list []RecordEntry
	err := db.WithContext(ctx).Raw(q, args...).Scan(&list).Error
	return list, total, err
}

func CreateRecordEntry(ctx context.Context, db *gorm.DB, medicalRecordID uuid.UUID, contentEncrypted, contentNonce []byte, contentKeyVersion string, entryDate time.Time, authorID uuid.UUID, authorType string) (uuid.UUID, error) {
	var res struct{ ID uuid.UUID }
	err := db.WithContext(ctx).Raw(`
		INSERT INTO record_entries (medical_record_id, content_encrypted, content_nonce, content_key_version, entry_date, author_id, author_type)
		VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id
	`, medicalRecordID, contentEncrypted, contentNonce, contentKeyVersion, entryDate, authorID, authorType).Scan(&res).Error
	return res.ID, err
}

func RecordEntryByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*RecordEntry, error) {
	var e RecordEntry
	err := db.WithContext(ctx).Raw(`
		SELECT id, medical_record_id, content_encrypted, content_nonce, content_key_version, entry_date, author_id, author_type, created_at
		FROM record_entries WHERE id = ?
	`, id).Scan(&e).Error
	if err != nil {
		return nil, err
	}
	if e.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &e, nil
}

func PatientIDByMedicalRecordID(ctx context.Context, db *gorm.DB, medicalRecordID uuid.UUID) (uuid.UUID, error) {
	var res struct{ PatientID uuid.UUID }
	err := db.WithContext(ctx).Raw(`SELECT patient_id FROM medical_records WHERE id = ?`, medicalRecordID).Scan(&res).Error
	return res.PatientID, err
}
