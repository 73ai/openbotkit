package store_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

func TestMigrateRefusesDatabaseFromNewerObk(t *testing.T) {
	db, err := store.Open(store.SQLiteConfig(filepath.Join(t.TempDir(), "whatsapp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := whatsapp.Migrate(db); err != nil {
		t.Fatal(err)
	}
	var known int
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations WHERE source = 'whatsapp'").Scan(&known); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO schema_migrations (source, version) VALUES ('whatsapp', ?)", known+1); err != nil {
		t.Fatal(err)
	}

	err = whatsapp.Migrate(db)
	if !errors.Is(err, store.ErrSchemaTooNew) {
		t.Fatalf("Migrate error = %v, want ErrSchemaTooNew", err)
	}
	want := fmt.Sprintf("database whatsapp is at schema version %d but this obk only knows up to %d; update obk", known+1, known)
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}
