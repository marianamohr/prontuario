package testutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prontuario/backend/internal/migrate"
)

// OpenPool abre pool a partir de DATABASE_URL. Se não houver, retorna nil.
func OpenPool(ctx context.Context) (*pgxpool.Pool, string) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, ""
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, url
	}
	return pool, url
}

func MustMigrate(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return err
	}
	return migrate.Run(ctx, pool, migrationsDir)
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
