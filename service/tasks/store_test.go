package tasks

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
	if err := Migrate(db); err != nil {
		db.Close()
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInsertAndGet(t *testing.T) {
	db := openTestDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	r := &TaskRecord{
		ID: "t1", Task: "do stuff", Agent: "claude",
		Status: "running", StartedAt: now,
	}
	if err := Insert(db, r); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := Get(db, "t1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.ID != "t1" || got.Task != "do stuff" || got.Agent != "claude" || got.Status != "running" {
		t.Errorf("unexpected record: %+v", got)
	}
	if got.StartedAt.Truncate(time.Second) != now {
		t.Errorf("StartedAt = %v, want %v", got.StartedAt, now)
	}
}

func TestSetCompleted(t *testing.T) {
	db := openTestDB(t)
	Insert(db, &TaskRecord{ID: "t1", Task: "t", Agent: "claude", Status: "running", StartedAt: time.Now()})
	if err := SetCompleted(db, "t1", "result text"); err != nil {
		t.Fatalf("SetCompleted: %v", err)
	}
	got, _ := Get(db, "t1")
	if got.Status != "completed" {
		t.Errorf("Status = %q", got.Status)
	}
	if got.Output != "result text" {
		t.Errorf("Output = %q", got.Output)
	}
	if got.DoneAt == nil {
		t.Error("DoneAt should be set")
	}
}

func TestSetFailed(t *testing.T) {
	db := openTestDB(t)
	Insert(db, &TaskRecord{ID: "t1", Task: "t", Agent: "claude", Status: "running", StartedAt: time.Now()})
	if err := SetFailed(db, "t1", "timeout"); err != nil {
		t.Fatalf("SetFailed: %v", err)
	}
	got, _ := Get(db, "t1")
	if got.Status != "failed" {
		t.Errorf("Status = %q", got.Status)
	}
	if got.Error != "timeout" {
		t.Errorf("Error = %q", got.Error)
	}
}

func TestGetNotFound(t *testing.T) {
	db := openTestDB(t)
	got, err := Get(db, "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestListOrdering(t *testing.T) {
	db := openTestDB(t)
	t1 := time.Now().UTC().Add(-2 * time.Hour)
	t2 := time.Now().UTC().Add(-1 * time.Hour)
	t3 := time.Now().UTC()
	Insert(db, &TaskRecord{ID: "old", Task: "t", Agent: "a", Status: "completed", StartedAt: t1})
	Insert(db, &TaskRecord{ID: "mid", Task: "t", Agent: "a", Status: "running", StartedAt: t2})
	Insert(db, &TaskRecord{ID: "new", Task: "t", Agent: "a", Status: "running", StartedAt: t3})

	list, err := List(db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("got %d tasks", len(list))
	}
	if list[0].ID != "new" || list[1].ID != "mid" || list[2].ID != "old" {
		t.Errorf("order: %s, %s, %s", list[0].ID, list[1].ID, list[2].ID)
	}
}

func TestCountRunning(t *testing.T) {
	db := openTestDB(t)
	Insert(db, &TaskRecord{ID: "t1", Task: "t", Agent: "a", Status: "running", StartedAt: time.Now()})
	Insert(db, &TaskRecord{ID: "t2", Task: "t", Agent: "a", Status: "completed", StartedAt: time.Now()})
	Insert(db, &TaskRecord{ID: "t3", Task: "t", Agent: "a", Status: "running", StartedAt: time.Now()})

	count, err := CountRunning(db)
	if err != nil {
		t.Fatalf("CountRunning: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestDeleteOlderThan(t *testing.T) {
	db := openTestDB(t)
	old := time.Now().UTC().Add(-10 * 24 * time.Hour)
	recent := time.Now().UTC()
	Insert(db, &TaskRecord{ID: "old", Task: "t", Agent: "a", Status: "completed", StartedAt: old})
	SetCompleted(db, "old", "done")
	// Manually set done_at to old time
	db.Exec(db.Rebind(`UPDATE tasks SET done_at = ? WHERE id = ?`), old.Format(timeFormat), "old")
	Insert(db, &TaskRecord{ID: "new", Task: "t", Agent: "a", Status: "completed", StartedAt: recent})
	SetCompleted(db, "new", "done")

	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)
	n, err := DeleteOlderThan(db, cutoff)
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted %d, want 1", n)
	}
	got, _ := Get(db, "old")
	if got != nil {
		t.Error("old task should be deleted")
	}
	got, _ = Get(db, "new")
	if got == nil {
		t.Error("new task should still exist")
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}
