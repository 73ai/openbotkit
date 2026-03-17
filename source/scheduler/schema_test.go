package scheduler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/73ai/openbotkit/store"
)

func TestMigrateSQLite(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(store.Config{
		Driver: "sqlite",
		DSN:    filepath.Join(dir, "test.db"),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schedules'").Scan(&name)
	if err != nil {
		t.Fatalf("table not found: %v", err)
	}
	if name != "schedules" {
		t.Fatalf("expected schedules, got %q", name)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
