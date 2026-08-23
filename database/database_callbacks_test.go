package database

import "testing"

func TestTimestampCallbacksCanBeDisabled(t *testing.T) {
	tests := []struct {
		name     string
		disabled bool
		want     bool
	}{
		{name: "enabled by default", want: true},
		{name: "disabled explicitly", disabled: true, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := NewDatabase(&DBConfig{
				Type:                      DBTypeSQLite,
				Path:                      ":memory:",
				DisableTimestampCallbacks: test.disabled,
			})
			if err != nil {
				t.Fatalf("open SQLite database: %v", err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatalf("get underlying database: %v", err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })

			if got := db.Callback().Create().Get("set_created_at") != nil; got != test.want {
				t.Fatalf("create timestamp callback registered = %v, want %v", got, test.want)
			}
			if got := db.Callback().Update().Get("set_updated_at") != nil; got != test.want {
				t.Fatalf("update timestamp callback registered = %v, want %v", got, test.want)
			}
		})
	}
}
