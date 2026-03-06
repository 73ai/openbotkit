package applenotes

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate(t *testing.T) {
	db := openTestDB(t)

	// Running migrate again should be idempotent
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestSaveAndGetNote(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().Truncate(time.Second)
	note := &Note{
		AppleID:           "x-coredata://ABC/ICNote/p1",
		Title:             "Test Note",
		Body:              "Hello world",
		Folder:            "Notes",
		FolderID:          "x-coredata://ABC/ICFolder/p1",
		Account:           "iCloud",
		PasswordProtected: false,
		CreatedAt:         now.Add(-time.Hour),
		ModifiedAt:        now,
	}

	if err := SaveNote(db, note); err != nil {
		t.Fatalf("save note: %v", err)
	}

	got, err := GetNote(db, note.AppleID)
	if err != nil {
		t.Fatalf("get note: %v", err)
	}

	if got.Title != "Test Note" {
		t.Errorf("title = %q, want %q", got.Title, "Test Note")
	}
	if got.Body != "Hello world" {
		t.Errorf("body = %q, want %q", got.Body, "Hello world")
	}
	if got.Folder != "Notes" {
		t.Errorf("folder = %q, want %q", got.Folder, "Notes")
	}
	if got.Account != "iCloud" {
		t.Errorf("account = %q, want %q", got.Account, "iCloud")
	}
}

func TestSaveNoteUpsert(t *testing.T) {
	db := openTestDB(t)

	note := &Note{
		AppleID: "x-coredata://ABC/ICNote/p2",
		Title:   "Original",
		Body:    "v1",
	}
	if err := SaveNote(db, note); err != nil {
		t.Fatalf("save: %v", err)
	}

	note.Title = "Updated"
	note.Body = "v2"
	if err := SaveNote(db, note); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := GetNote(db, note.AppleID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Updated" {
		t.Errorf("title = %q, want %q", got.Title, "Updated")
	}
	if got.Body != "v2" {
		t.Errorf("body = %q, want %q", got.Body, "v2")
	}
}

func TestListNotes(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().Truncate(time.Second)
	for i, title := range []string{"Note A", "Note B", "Note C"} {
		n := &Note{
			AppleID:    "x-coredata://ABC/ICNote/p" + title,
			Title:      title,
			Body:       "body " + title,
			Folder:     "Notes",
			ModifiedAt: now.Add(time.Duration(i) * time.Minute),
		}
		if err := SaveNote(db, n); err != nil {
			t.Fatalf("save %q: %v", title, err)
		}
	}

	notes, err := ListNotes(db, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("count = %d, want 3", len(notes))
	}
	// Should be ordered by modified_at DESC
	if notes[0].Title != "Note C" {
		t.Errorf("first = %q, want %q", notes[0].Title, "Note C")
	}
}

func TestListNotesFilterByFolder(t *testing.T) {
	db := openTestDB(t)

	for _, n := range []Note{
		{AppleID: "id1", Title: "In Work", Folder: "Work"},
		{AppleID: "id2", Title: "In Personal", Folder: "Personal"},
		{AppleID: "id3", Title: "Also Work", Folder: "Work"},
	} {
		if err := SaveNote(db, &n); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	notes, err := ListNotes(db, ListOptions{Folder: "Work", Limit: 50})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("count = %d, want 2", len(notes))
	}
}

func TestSearchNotes(t *testing.T) {
	db := openTestDB(t)

	for _, n := range []Note{
		{AppleID: "s1", Title: "Meeting notes", Body: "discuss project"},
		{AppleID: "s2", Title: "Shopping list", Body: "buy groceries"},
		{AppleID: "s3", Title: "Project plan", Body: "timeline and milestones"},
	} {
		if err := SaveNote(db, &n); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	notes, err := SearchNotes(db, "project", 50)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("count = %d, want 2", len(notes))
	}
}

func TestCountNotes(t *testing.T) {
	db := openTestDB(t)

	count, err := CountNotes(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	SaveNote(db, &Note{AppleID: "c1", Title: "One"})
	SaveNote(db, &Note{AppleID: "c2", Title: "Two"})

	count, err = CountNotes(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestLastSyncTime(t *testing.T) {
	db := openTestDB(t)

	last, err := LastSyncTime(db)
	if err != nil {
		t.Fatalf("last sync: %v", err)
	}
	if last != nil {
		t.Errorf("expected nil, got %v", last)
	}

	SaveNote(db, &Note{AppleID: "lt1", Title: "First"})
	last, err = LastSyncTime(db)
	if err != nil {
		t.Fatalf("last sync: %v", err)
	}
	if last == nil {
		t.Fatal("expected non-nil time")
	}
}

func TestSaveAndGetFolder(t *testing.T) {
	db := openTestDB(t)

	folder := &Folder{
		AppleID:       "x-coredata://ABC/ICFolder/p1",
		Name:          "Work",
		ParentAppleID: "",
		Account:       "iCloud",
	}

	if err := SaveFolder(db, folder); err != nil {
		t.Fatalf("save folder: %v", err)
	}

	// Upsert with different name
	folder.Name = "Work Projects"
	if err := SaveFolder(db, folder); err != nil {
		t.Fatalf("upsert folder: %v", err)
	}

	// Verify via raw query
	var name string
	err := db.QueryRow(
		db.Rebind("SELECT name FROM applenotes_folders WHERE apple_id = ?"),
		folder.AppleID,
	).Scan(&name)
	if err != nil {
		t.Fatalf("query folder: %v", err)
	}
	if name != "Work Projects" {
		t.Errorf("name = %q, want %q", name, "Work Projects")
	}
}
