//go:build sqlite_only

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	migratedatabase "github.com/golang-migrate/migrate/v4/database"
)

const sqlite_migrations_table = "schema_migrations"

// sqlite_migration_driver adapts an existing pure-Go SQLite connection to
// golang-migrate without blank-importing another package that also registers
// the global "sqlite" database/sql driver name.
type sqlite_migration_driver struct {
	db        *sql.DB
	is_locked atomic.Bool
}

func new_sqlite_migration_driver(db *sql.DB) (*sqlite_migration_driver, error) {
	if db == nil {
		return nil, fmt.Errorf("SQLite migration database is nil")
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping SQLite migration database: %w", err)
	}

	driver := &sqlite_migration_driver{db: db}
	if err := driver.ensure_version_table(); err != nil {
		return nil, err
	}
	return driver, nil
}

func (d *sqlite_migration_driver) Open(string) (migratedatabase.Driver, error) {
	return nil, fmt.Errorf("SQLite migration driver only supports an existing database connection")
}

func (d *sqlite_migration_driver) Close() error {
	return d.db.Close()
}

func (d *sqlite_migration_driver) Lock() error {
	if !d.is_locked.CompareAndSwap(false, true) {
		return migratedatabase.ErrLocked
	}
	return nil
}

func (d *sqlite_migration_driver) Unlock() error {
	if !d.is_locked.CompareAndSwap(true, false) {
		return migratedatabase.ErrNotLocked
	}
	return nil
}

func (d *sqlite_migration_driver) Run(migration io.Reader) error {
	query, err := io.ReadAll(migration)
	if err != nil {
		return fmt.Errorf("read SQLite migration: %w", err)
	}
	return d.execute_query(string(query))
}

func (d *sqlite_migration_driver) SetVersion(version int, dirty bool) error {
	tx, err := d.db.Begin()
	if err != nil {
		return &migratedatabase.Error{OrigErr: err, Err: "transaction start failed"}
	}

	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	delete_query := "DELETE FROM " + sqlite_migrations_table
	if _, err := tx.Exec(delete_query); err != nil {
		return &migratedatabase.Error{OrigErr: err, Query: []byte(delete_query)}
	}

	if version >= 0 || (version == migratedatabase.NilVersion && dirty) {
		insert_query := fmt.Sprintf(
			"INSERT INTO %s (version, dirty) VALUES (?, ?)",
			sqlite_migrations_table,
		)
		if _, err := tx.Exec(insert_query, version, dirty); err != nil {
			return &migratedatabase.Error{OrigErr: err, Query: []byte(insert_query)}
		}
	}

	if err := tx.Commit(); err != nil {
		return &migratedatabase.Error{OrigErr: err, Err: "transaction commit failed"}
	}
	rollback = false
	return nil
}

func (d *sqlite_migration_driver) Version() (int, bool, error) {
	query := "SELECT version, dirty FROM " + sqlite_migrations_table + " LIMIT 1"
	var version int
	var dirty bool
	if err := d.db.QueryRow(query).Scan(&version, &dirty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return migratedatabase.NilVersion, false, nil
		}
		return migratedatabase.NilVersion, false, err
	}
	return version, dirty, nil
}

func (d *sqlite_migration_driver) Drop() error {
	rows, err := d.db.Query(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'",
	)
	if err != nil {
		return &migratedatabase.Error{OrigErr: err, Err: "list SQLite tables failed"}
	}

	var table_names []string
	for rows.Next() {
		var table_name string
		if err := rows.Scan(&table_name); err != nil {
			_ = rows.Close()
			return err
		}
		table_names = append(table_names, table_name)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, table_name := range table_names {
		quoted_name := `"` + strings.ReplaceAll(table_name, `"`, `""`) + `"`
		if err := d.execute_query("DROP TABLE " + quoted_name); err != nil {
			return err
		}
	}
	if _, err := d.db.Exec("VACUUM"); err != nil {
		return &migratedatabase.Error{OrigErr: err, Query: []byte("VACUUM")}
	}
	return nil
}

func (d *sqlite_migration_driver) ensure_version_table() (err error) {
	if err := d.Lock(); err != nil {
		return err
	}
	defer func() {
		if unlock_err := d.Unlock(); err == nil {
			err = unlock_err
		}
	}()

	query := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (version uint64, dirty bool); "+
			"CREATE UNIQUE INDEX IF NOT EXISTS version_unique ON %s (version);",
		sqlite_migrations_table,
		sqlite_migrations_table,
	)
	if _, err := d.db.Exec(query); err != nil {
		return &migratedatabase.Error{OrigErr: err, Query: []byte(query)}
	}
	return nil
}

func (d *sqlite_migration_driver) execute_query(query string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return &migratedatabase.Error{OrigErr: err, Err: "transaction start failed"}
	}
	if _, err := tx.Exec(query); err != nil {
		_ = tx.Rollback()
		return &migratedatabase.Error{OrigErr: err, Query: []byte(query)}
	}
	if err := tx.Commit(); err != nil {
		return &migratedatabase.Error{OrigErr: err, Err: "transaction commit failed"}
	}
	return nil
}
