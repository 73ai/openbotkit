package hooks

import (
	"testing"
	"time"

	"github.com/73ai/openbotkit/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(store.SQLiteConfig(":memory:"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCreateAndGet(t *testing.T) {
	db := openTestDB(t)
	id, err := Create(db, &EventHook{
		EventType: "gmail_sync",
		Prompt:    "test prompt",
		Channel:   "telegram",
		ModelTier: "nano",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := Get(db, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.EventType != "gmail_sync" || got.Channel != "telegram" {
		t.Errorf("unexpected hook: %+v", got)
	}
	if !got.Enabled {
		t.Error("expected enabled")
	}
}

func TestListEnabled(t *testing.T) {
	db := openTestDB(t)
	Create(db, &EventHook{EventType: "gmail_sync", Prompt: "p1", Channel: "telegram", ModelTier: "nano"})
	Create(db, &EventHook{EventType: "gmail_sync", Prompt: "p2", Channel: "telegram", ModelTier: "nano"})
	Create(db, &EventHook{EventType: "backup_complete", Prompt: "p3", Channel: "telegram", ModelTier: "nano"})

	hooks, err := ListEnabled(db, "gmail_sync")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(hooks))
	}
}

func TestGetNotFound(t *testing.T) {
	db := openTestDB(t)
	_, err := Get(db, 999)
	if err == nil {
		t.Error("expected error for non-existent hook")
	}
}

func TestListEnabledEmpty(t *testing.T) {
	db := openTestDB(t)
	hooks, err := ListEnabled(db, "nonexistent")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks, got %d", len(hooks))
	}
}

func TestUpdateLastRun(t *testing.T) {
	db := openTestDB(t)
	id, _ := Create(db, &EventHook{EventType: "gmail_sync", Prompt: "p", Channel: "telegram", ModelTier: "nano"})

	now := time.Now().UTC()
	if err := UpdateLastRun(db, id, now, "some error"); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := Get(db, id)
	if got.LastError != "some error" {
		t.Errorf("expected error string, got %q", got.LastError)
	}
	if got.LastRunAt == nil {
		t.Error("expected last_run_at to be set")
	}
}
