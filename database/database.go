package database

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/ltaoo/velo/dir"
	"gorm.io/gorm"
)

// DBType represents the database driver type.
type DBType string

const (
	DBTypeSQLite   DBType = "sqlite"
	DBTypeMySQL    DBType = "mysql"
	DBTypePostgres DBType = "postgres"
)

// DBConfig holds the configuration for a database connection.
type DBConfig struct {
	Type     DBType
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	// Path is the file path for SQLite databases.
	Path string
}

// DefaultSQLiteConfig returns a config for a SQLite database stored beside the executable.
func DefaultSQLiteConfig() *DBConfig {
	return &DBConfig{
		Type: DBTypeSQLite,
		Path: filepath.Join(dir.ExeDir(), "app.db"),
	}
}

func openDatabase(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	registerTimestampCallbacks(db)

	return db, nil
}

// registerTimestampCallbacks adds GORM callbacks that set created_at and updated_at
// as formatted time strings on create and update operations.
func registerTimestampCallbacks(db *gorm.DB) {
	now := func() string {
		return time.Now().Format("2006-01-02 15:04:05")
	}

	db.Callback().Create().Before("gorm:create").Register("set_created_at", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil {
			if field := tx.Statement.Schema.LookUpField("CreatedAt"); field != nil {
				if _, isZero := field.ValueOf(tx.Statement.Context, tx.Statement.ReflectValue); isZero {
					field.Set(tx.Statement.Context, tx.Statement.ReflectValue, now())
				}
			}
			if field := tx.Statement.Schema.LookUpField("UpdatedAt"); field != nil {
				field.Set(tx.Statement.Context, tx.Statement.ReflectValue, now())
			}
		}
	})

	db.Callback().Update().Before("gorm:update").Register("set_updated_at", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil {
			if field := tx.Statement.Schema.LookUpField("UpdatedAt"); field != nil {
				field.Set(tx.Statement.Context, tx.Statement.ReflectValue, now())
			}
		}
	})
}
