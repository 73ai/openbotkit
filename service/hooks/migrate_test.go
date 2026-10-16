package hooks

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/73ai/openbotkit/service/scheduler"
	"github.com/73ai/openbotkit/store"
	"github.com/73ai/openbotkit/store/migratetest"
)

func TestMigrations(t *testing.T) {
	migratetest.Run(t, migratetest.Suite{
		Source:   "hooks",
		Migrate:  Migrate,
		Steps:    migrations,
		Fixtures: []migratetest.Fixture{{Name: "pre_runner", SQLite: schemaSQLite, Postgres: schemaPostgres}},
	})
}

func TestMigrationsShareSchedulerDatabase(t *testing.T) {
	db, err := store.Open(store.SQLiteConfig(filepath.Join(t.TempDir(), "data.db")))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := scheduler.Migrate(db); err != nil {
		t.Fatalf("scheduler migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("hooks migrate: %v", err)
	}
	got := migratetest.Dump(t, db, "SELECT source, version FROM schema_migrations ORDER BY source, version")
	if want := []string{"hooks 1", "scheduler 1", "scheduler 2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("schema_migrations = %v, want %v", got, want)
	}
}
