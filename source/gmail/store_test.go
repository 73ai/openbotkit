package gmail

import (
	"path/filepath"
	"testing"

	"github.com/73ai/openbotkit/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestGetSyncState_NoRow(t *testing.T) {
	db := openTestDB(t)

	state, err := GetSyncState(db, "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil state, got %+v", state)
	}
}

func TestSaveSyncState_Insert(t *testing.T) {
	db := openTestDB(t)

	if err := SaveSyncState(db, "alice@example.com", 12345); err != nil {
		t.Fatalf("save: %v", err)
	}

	state, err := GetSyncState(db, "alice@example.com")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if state == nil {
		t.Fatal("expected state, got nil")
	}
	if state.Account != "alice@example.com" {
		t.Errorf("account = %q, want %q", state.Account, "alice@example.com")
	}
	if state.HistoryID != 12345 {
		t.Errorf("historyID = %d, want %d", state.HistoryID, 12345)
	}
	if state.UpdatedAt.IsZero() {
		t.Error("updatedAt should not be zero")
	}
}

func TestSaveSyncState_Upsert(t *testing.T) {
	db := openTestDB(t)

	if err := SaveSyncState(db, "alice@example.com", 100); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := SaveSyncState(db, "alice@example.com", 200); err != nil {
		t.Fatalf("second save: %v", err)
	}

	state, err := GetSyncState(db, "alice@example.com")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if state.HistoryID != 200 {
		t.Errorf("historyID = %d, want %d after upsert", state.HistoryID, 200)
	}
}

func TestSaveSyncState_MultipleAccounts(t *testing.T) {
	db := openTestDB(t)

	if err := SaveSyncState(db, "alice@example.com", 100); err != nil {
		t.Fatalf("save alice: %v", err)
	}
	if err := SaveSyncState(db, "bob@example.com", 200); err != nil {
		t.Fatalf("save bob: %v", err)
	}

	alice, err := GetSyncState(db, "alice@example.com")
	if err != nil {
		t.Fatalf("get alice: %v", err)
	}
	bob, err := GetSyncState(db, "bob@example.com")
	if err != nil {
		t.Fatalf("get bob: %v", err)
	}

	if alice.HistoryID != 100 {
		t.Errorf("alice historyID = %d, want 100", alice.HistoryID)
	}
	if bob.HistoryID != 200 {
		t.Errorf("bob historyID = %d, want 200", bob.HistoryID)
	}
}
