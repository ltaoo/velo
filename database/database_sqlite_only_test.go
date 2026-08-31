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
	sql_db, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying database: %v", err)
	}
	t.Cleanup(func() { _ = sql_db.Close() })

	_, driver_name, err := newMigrateDriver(cfg, sql_db)
	if err != nil {
		t.Fatalf("create SQLite migration driver: %v", err)
	}
	if driver_name != "sqlite" {
		t.Fatalf("unexpected SQLite migration driver name: %s", driver_name)
	}

	migration_runner := NewMigrator(cfg, &sqliteOnlyMigrations)
	if err := migration_runner.MigrateUp(db); err != nil {
		t.Fatalf("run SQLite migration: %v", err)
	}

	var table_name string
	if err := sql_db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
		"velo_migration_test",
	).Scan(&table_name); err != nil {
		t.Fatalf("find migrated table: %v", err)
	}

	if err := migration_runner.MigrateDown(db); err != nil {
		t.Fatalf("roll back SQLite migration: %v", err)
	}
	var table_count int
	if err := sql_db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		"velo_migration_test",
	).Scan(&table_count); err != nil {
		t.Fatalf("check rolled back table: %v", err)
	}
	if table_count != 0 {
		t.Fatalf("expected rolled back table to be removed, got count %d", table_count)
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
