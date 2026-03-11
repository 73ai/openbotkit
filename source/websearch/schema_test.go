package websearch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/priyanshujain/openbotkit/store"
)

func TestMigrateSQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Verify tables exist by querying them.
	tables := []string{"search_history", "search_cache", "fetch_cache"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err != nil {
			t.Errorf("table %s not created: %v", table, err)
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	if err := Migrate(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate should not error: %v", err)
	}
}
