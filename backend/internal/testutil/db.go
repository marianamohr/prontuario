package testutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/prontuario/backend/internal/migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// OpenDB abre conexão GORM a partir de DATABASE_URL. Se não houver, retorna nil.
func OpenDB(ctx context.Context) (*gorm.DB, string) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, ""
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, url
	}
	if _, err := db.DB(); err != nil {
		return nil, url
	}
	return db, url
}

func MustMigrate(ctx context.Context, db *gorm.DB) error {
	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return err
	}
	return migrate.Run(ctx, db, migrationsDir)
}

func findMigrationsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(cur, "migrations")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return "", errors.New("migrations dir not found from working directory")
}
