//go:build sqlite_only

package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4/database"
	sqlitemigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
)

func newMigrateDriver(cfg *DBConfig, db *sql.DB) (database.Driver, string, error) {
	if cfg.Type != DBTypeSQLite {
		return nil, "", fmt.Errorf(
			"unsupported database type for migration in sqlite_only build: %s",
			cfg.Type,
		)
	}

	driver, err := sqlitemigrate.WithInstance(db, &sqlitemigrate.Config{})
	return driver, "sqlite3", err
}
