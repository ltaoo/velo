package velo

import (
	"embed"

	"github.com/ltaoo/velo/database"
	"gorm.io/gorm"
)

// DBType represents a database driver type.
//
// It aliases database.DBType so applications can configure a database by
// importing only the root velo package.
type DBType = database.DBType

// DBConfig holds the configuration for a database connection.
type DBConfig = database.DBConfig

// Migrator runs embedded schema migrations against a database.
type Migrator = database.Migrator

// VeloDatabaseOpt configures a database initialized through Box.Migrate.
type VeloDatabaseOpt struct {
	DBType     DBType
	DBPath     string
	Migrations *embed.FS
}

const (
	DBTypeSQLite   = database.DBTypeSQLite
	DBTypeMySQL    = database.DBTypeMySQL
	DBTypePostgres = database.DBTypePostgres
)

// DefaultSQLiteConfig returns a config for a SQLite database stored beside the executable.
func DefaultSQLiteConfig() *DBConfig {
	return database.DefaultSQLiteConfig()
}

// NewDatabase opens a database connection based on the given config.
func NewDatabase(cfg *DBConfig) (*gorm.DB, error) {
	return database.NewDatabase(cfg)
}

// NewMigrator creates a Migrator that will apply the given embedded migrations.
func NewMigrator(cfg *DBConfig, migrations *embed.FS) *Migrator {
	return database.NewMigrator(cfg, migrations)
}
