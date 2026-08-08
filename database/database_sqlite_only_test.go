//go:build sqlite_only

package database

import (
	"embed"
	"testing"
)

//go:embed migrations/*.sql
var sqliteOnlyMigrations embed.FS

func TestSQLiteOnlyDatabase(t *testing.T) {
	db, err := NewDatabase(&DBConfig{Type: DBTypeSQLite, Path: ":memory:"})
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if _, err := sqlDB.Exec("CREATE TABLE velo_sqlite_only_test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("execute SQLite statement: %v", err)
	}
}

func TestSQLiteOnlyMigration(t *testing.T) {
	cfg := &DBConfig{Type: DBTypeSQLite, Path: ":memory:"}
	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("open SQLite database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := NewMigrator(cfg, &sqliteOnlyMigrations).MigrateUp(db); err != nil {
		t.Fatalf("run SQLite migration: %v", err)
	}

	var tableName string
	if err := sqlDB.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
		"velo_migration_test",
	).Scan(&tableName); err != nil {
		t.Fatalf("find migrated table: %v", err)
	}
}

func TestSQLiteOnlyRejectsOtherDatabaseTypes(t *testing.T) {
	for _, dbType := range []DBType{DBTypeMySQL, DBTypePostgres} {
		t.Run(string(dbType), func(t *testing.T) {
			cfg := &DBConfig{Type: dbType}
			if _, err := NewDatabase(cfg); err == nil {
				t.Fatalf("expected %s database to be unavailable", dbType)
			}

			db, err := NewDatabase(&DBConfig{Type: DBTypeSQLite, Path: ":memory:"})
			if err != nil {
				t.Fatalf("open SQLite database: %v", err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatalf("get underlying database: %v", err)
			}
			defer sqlDB.Close()
			if _, _, err := newMigrateDriver(cfg, sqlDB); err == nil {
				t.Fatalf("expected %s migration driver to be unavailable", dbType)
			}
		})
	}
}
