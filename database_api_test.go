package velo_test

import (
	"testing"

	"github.com/ltaoo/velo"
)

func TestDatabaseAPIAvailableFromRootPackage(t *testing.T) {
	cfg := &velo.DBConfig{
		Type: velo.DBTypeSQLite,
		Path: ":memory:",
	}

	var dbType velo.DBType = cfg.Type
	if dbType != velo.DBTypeSQLite {
		t.Fatalf("unexpected database type: %q", dbType)
	}

	db, err := velo.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("open database through root package: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying database: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if migrator := velo.NewMigrator(cfg, nil); migrator == nil {
		t.Fatal("expected root package to create a migrator")
	}

	defaultCfg := velo.DefaultSQLiteConfig()
	if defaultCfg.Type != velo.DBTypeSQLite || defaultCfg.Path == "" {
		t.Fatalf("unexpected default SQLite config: %#v", defaultCfg)
	}

	_ = []velo.DBType{velo.DBTypeMySQL, velo.DBTypePostgres}
}

func TestBoxMigrateUsesConfiguredSQLiteDatabase(t *testing.T) {
	app := velo.NewApp(&velo.VeloAppOpt{Mode: velo.ModeHttp})

	if err := app.Migrate(nil); err == nil {
		t.Fatal("expected nil database options to return an error")
	}

	if err := app.Migrate(&velo.VeloDatabaseOpt{
		DBType:                    velo.DBTypeSQLite,
		DBPath:                    ":memory:",
		DisableTimestampCallbacks: true,
	}); err != nil {
		t.Fatalf("migrate through Box: %v", err)
	}
	if app.DB == nil {
		t.Fatal("expected Migrate to store the database on Box.DB")
	}
	if app.DB.Callback().Create().Get("set_created_at") != nil {
		t.Fatal("expected Migrate to disable timestamp callbacks")
	}

	sqlDB, err := app.DB.DB()
	if err != nil {
		t.Fatalf("get underlying database: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
}
