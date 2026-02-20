package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// Run applies all pending migrations in migrationsDir (e.g. "migrations").
func Run(ctx context.Context, db *gorm.DB, migrationsDir string) error {
	ensureSchemaMigrations(ctx, db)
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := db.WithContext(ctx).Exec(string(raw)).Error; err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if err := db.WithContext(ctx).Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}

func ensureSchemaMigrations(ctx context.Context, db *gorm.DB) {
	_ = db.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`).Error
}

func appliedVersions(ctx context.Context, db *gorm.DB) (map[string]bool, error) {
	var rows []struct {
		Version string `gorm:"column:version"`
	}
	if err := db.WithContext(ctx).Raw("SELECT version FROM schema_migrations").Scan(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]bool)
	for _, r := range rows {
		m[r.Version] = true
	}
	return m, nil
}
