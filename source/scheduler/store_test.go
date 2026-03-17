package scheduler

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(store.Config{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCRUDRoundTrip(t *testing.T) {
	db := openTestDB(t)

	s := &Schedule{
		Type:        Recurring,
		CronExpr:    "0 9 * * *",
		Task:        "check weather",
		Channel:     "telegram",
		ChannelMeta: ChannelMeta{BotToken: "tok", OwnerID: 42},
		Timezone:    "UTC",
		Description: "daily weather",
	}

	id, err := Create(db, s)
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
	if got.Type != Recurring {
		t.Errorf("type: got %q, want %q", got.Type, Recurring)
	}
	if got.Task != "check weather" {
		t.Errorf("task: got %q, want %q", got.Task, "check weather")
	}
	if got.ChannelMeta.OwnerID != 42 {
		t.Errorf("owner_id: got %d, want 42", got.ChannelMeta.OwnerID)
	}
	if !got.Enabled {
		t.Error("expected enabled")
	}

	all, err := List(db)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("list: got %d, want 1", len(all))
	}

	enabled, err := ListEnabled(db)
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("list enabled: got %d, want 1", len(enabled))
	}

	now := time.Now().UTC()
	if err := UpdateLastRun(db, id, now, ""); err != nil {
		t.Fatalf("update last run: %v", err)
	}
	got, _ = Get(db, id)
	if got.LastRunAt == nil {
		t.Fatal("expected last_run_at set")
	}

	if err := Delete(db, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	all, _ = List(db)
	if len(all) != 0 {
		t.Fatalf("after delete: got %d, want 0", len(all))
	}
}

func TestListDueOneShot(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	pastSchedule := &Schedule{
		Type:        OneShot,
		ScheduledAt: &past,
		Task:        "past task",
		Channel:     "telegram",
		ChannelMeta: ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	}
	futureSchedule := &Schedule{
		Type:        OneShot,
		ScheduledAt: &future,
		Task:        "future task",
		Channel:     "telegram",
		ChannelMeta: ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	}

	if _, err := Create(db, pastSchedule); err != nil {
		t.Fatalf("create past: %v", err)
	}
	if _, err := Create(db, futureSchedule); err != nil {
		t.Fatalf("create future: %v", err)
	}

	due, err := ListDueOneShot(db, now)
	if err != nil {
		t.Fatalf("list due: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due: got %d, want 1", len(due))
	}
	if due[0].Task != "past task" {
		t.Errorf("task: got %q, want %q", due[0].Task, "past task")
	}
}

func TestMarkCompleted(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	s := &Schedule{
		Type:        OneShot,
		ScheduledAt: &past,
		Task:        "do something",
		Channel:     "telegram",
		ChannelMeta: ChannelMeta{BotToken: "tok", OwnerID: 1},
		Timezone:    "UTC",
	}
	id, err := Create(db, s)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := MarkCompleted(db, id, now); err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	got, _ := Get(db, id)
	if got.CompletedAt == nil {
		t.Fatal("expected completed_at set")
	}
	if got.Enabled {
		t.Error("expected disabled after completion")
	}

	due, _ := ListDueOneShot(db, now)
	if len(due) != 0 {
		t.Fatalf("should not be due after completion, got %d", len(due))
	}
}
