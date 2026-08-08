package database

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"gorm.io/gorm"
)

// Migrator runs schema migrations against a database.
type Migrator struct {
	cfg        *DBConfig
	migrations *embed.FS
}

// NewMigrator creates a Migrator that will apply the given embedded migrations.
func NewMigrator(cfg *DBConfig, migrations *embed.FS) *Migrator {
	return &Migrator{cfg: cfg, migrations: migrations}
}

func (m *Migrator) newMigrateInstance(db *gorm.DB) (*migrate.Migrate, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	driver, driverName, err := newMigrateDriver(m.cfg, sqlDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate driver: %w", err)
	}

	// Use the embedded FS as the migration source via httpfs.
	subFS, err := fs.Sub(m.migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to open migrations sub-directory: %w", err)
	}
	source, err := httpfs.New(http.FS(subFS), ".")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	mg, err := migrate.NewWithInstance("httpfs", source, driverName, driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return mg, nil
}

// MigrateUp runs all pending up migrations.
func (m *Migrator) MigrateUp(db *gorm.DB) error {
	mg, err := m.newMigrateInstance(db)
	if err != nil {
		return err
	}
	if err := mg.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up failed: %w", err)
	}
	return nil
}

// MigrateDown rolls back all applied migrations.
func (m *Migrator) MigrateDown(db *gorm.DB) error {
	mg, err := m.newMigrateInstance(db)
	if err != nil {
		return err
	}
	if err := mg.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

// MigrateTo migrates to the specified version.
func (m *Migrator) MigrateTo(db *gorm.DB, version uint) error {
	mg, err := m.newMigrateInstance(db)
	if err != nil {
		return err
	}
	if err := mg.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration to version %d failed: %w", version, err)
	}
	return nil
}
