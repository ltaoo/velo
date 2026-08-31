//go:build sqlite_only

package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4/database"
)

func newMigrateDriver(cfg *DBConfig, db *sql.DB) (database.Driver, string, error) {
	if cfg.Type != DBTypeSQLite {
		return nil, "", fmt.Errorf(
			"unsupported database type for migration in sqlite_only build: %s",
			cfg.Type,
		)
	}

	driver, err := new_sqlite_migration_driver(db)
	return driver, "sqlite", err
}
