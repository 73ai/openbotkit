package history

import (
	"testing"

	"github.com/priyanshujain/openbotkit/store"
)

func testDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate(t *testing.T) {
	db := testDB(t)

	if err := Migrate(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM history_conversations").Scan(&count); err != nil {
		t.Fatalf("query conversations table: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM history_messages").Scan(&count); err != nil {
		t.Fatalf("query messages table: %v", err)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db := testDB(t)

	if err := Migrate(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate should be idempotent: %v", err)
	}
}
