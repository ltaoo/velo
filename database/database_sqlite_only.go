//go:build sqlite_only

package database

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// NewDatabase opens a pure-Go SQLite connection. Other database types are not
// linked into sqlite_only builds.
func NewDatabase(cfg *DBConfig) (*gorm.DB, error) {
	if cfg.Type != DBTypeSQLite {
		return nil, fmt.Errorf("unsupported database type in sqlite_only build: %s", cfg.Type)
	}

	return openDatabase(sqlite.Open(cfg.Path), cfg)
}
