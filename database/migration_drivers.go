//go:build !sqlite_only

package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4/database"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	pgmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	sqlitemigrate "github.com/golang-migrate/migrate/v4/database/sqlite3"
)

func newMigrateDriver(cfg *DBConfig, db *sql.DB) (database.Driver, string, error) {
	var (
		driver database.Driver
		name   string
		err    error
	)

	switch cfg.Type {
	case DBTypeSQLite:
		name = "sqlite3"
		driver, err = sqlitemigrate.WithInstance(db, &sqlitemigrate.Config{})
	case DBTypeMySQL:
		name = "mysql"
		driver, err = mysqlmigrate.WithInstance(db, &mysqlmigrate.Config{})
	case DBTypePostgres:
		name = "postgres"
		driver, err = pgmigrate.WithInstance(db, &pgmigrate.Config{})
	default:
		err = fmt.Errorf("unsupported database type for migration: %s", cfg.Type)
	}
	return driver, name, err
}
