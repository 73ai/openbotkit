package servertest

import (
	"testing"

	"github.com/73ai/openbotkit/remote"
)

// Backend provides a configured client for running the server API contract
// tests. The suite verifies auth, CRUD, validation, and DB proxy behavior
// through the remote.Client — backend-agnostic so the same tests run against
// a local httptest server and a Docker container.
type Backend struct {
	Client       *remote.Client
	NoAuthClient *remote.Client
	// SeedDB seeds a source database with raw SQL (e.g. via sqlite3 exec).
	// Nil means direct-DB-seeding tests are skipped (e.g. Docker backend).
	SeedDB func(t *testing.T, source, sql string)
}

// Run executes the full server test suite against the given backend.
func Run(t *testing.T, b Backend) {
	t.Run("health", func(t *testing.T) { testHealth(t, b) })
	t.Run("memory_crud", func(t *testing.T) { testMemoryCRUD(t, b) })
	t.Run("auth", func(t *testing.T) { testAuth(t, b) })
	t.Run("applenotes_push_and_query", func(t *testing.T) { testAppleNotes(t, b) })
	t.Run("db_reject_non_select", func(t *testing.T) { testDBRejectNonSelect(t, b) })
	t.Run("db_reject_unknown_source", func(t *testing.T) { testDBRejectUnknownSource(t, b) })
	t.Run("gmail_send_validation", func(t *testing.T) { testGmailSendValidation(t, b) })
	t.Run("whatsapp_send_validation", func(t *testing.T) { testWhatsAppSendValidation(t, b) })
	if b.SeedDB != nil {
		t.Run("db_seeded_queries", func(t *testing.T) { testDBSeededQueries(t, b) })
	}
}

func testHealth(t *testing.T, b Backend) {
	health, err := b.Client.Health()
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health["status"] != "ok" {
		t.Fatalf("expected ok, got %q", health["status"])
	}
}

func testMemoryCRUD(t *testing.T, b Backend) {
	id1, err := b.Client.MemoryAdd("lives in San Francisco", "identity", "manual")
	if err != nil {
		t.Fatalf("add 1: %v", err)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero ID")
	}

	id2, err := b.Client.MemoryAdd("prefers dark mode", "preference", "manual")
	if err != nil {
		t.Fatalf("add 2: %v", err)
	}

	// List all
	items, err := b.Client.MemoryList("")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(items))
	}

	// List by category
	items, err = b.Client.MemoryList("identity")
	if err != nil {
		t.Fatalf("list by category: %v", err)
	}
	if len(items) != 1 || items[0].Content != "lives in San Francisco" {
		t.Fatalf("unexpected filtered list: %v", items)
	}

	// Delete one
	if err := b.Client.MemoryDelete(id2); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify only one left
	items, err = b.Client.MemoryList("")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 memory after delete, got %d", len(items))
	}

	// Cleanup
	b.Client.MemoryDelete(id1)
}

func testAuth(t *testing.T, b Backend) {
	if b.NoAuthClient == nil {
		t.Skip("no unauthenticated client provided")
	}

	// Health works without auth
	_, err := b.NoAuthClient.Health()
	if err != nil {
		t.Fatalf("health without auth should work: %v", err)
	}

	// Memory list fails without auth
	_, err = b.NoAuthClient.MemoryList("")
	if err == nil {
		t.Fatal("expected auth error for memory list")
	}

	// Memory add fails without auth
	_, err = b.NoAuthClient.MemoryAdd("test", "preference", "manual")
	if err == nil {
		t.Fatal("expected auth error for memory add")
	}
}

func testAppleNotes(t *testing.T, b Backend) {
	notes := []map[string]any{
		{
			"apple_id":           "suite-note-1",
			"title":              "Shopping List",
			"body":               "Milk, bread, eggs",
			"folder":             "Notes",
			"folder_id":          "folder-1",
			"account":            "iCloud",
			"password_protected": false,
			"created_at":         "2024-01-15T10:00:00Z",
			"modified_at":        "2024-01-15T10:00:00Z",
		},
		{
			"apple_id":           "suite-note-2",
			"title":              "Meeting Notes",
			"body":               "Discussed project timeline",
			"folder":             "Work",
			"folder_id":          "folder-2",
			"account":            "iCloud",
			"password_protected": false,
			"created_at":         "2024-01-16T14:00:00Z",
			"modified_at":        "2024-01-16T14:00:00Z",
		},
	}

	if err := b.Client.AppleNotesPush(notes); err != nil {
		t.Fatalf("push: %v", err)
	}

	// Verify via DB proxy
	resp, err := b.Client.DB("applenotes", "SELECT title FROM applenotes_notes ORDER BY title")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(resp.Rows) < 2 {
		t.Fatalf("expected at least 2 notes, got %d", len(resp.Rows))
	}

	// Verify upsert
	notes[0]["body"] = "Milk, bread, eggs, cheese"
	if err := b.Client.AppleNotesPush(notes); err != nil {
		t.Fatalf("re-push: %v", err)
	}
	resp, err = b.Client.DB("applenotes", "SELECT body FROM applenotes_notes WHERE apple_id = 'suite-note-1'")
	if err != nil {
		t.Fatalf("verify upsert: %v", err)
	}
	if len(resp.Rows) != 1 || resp.Rows[0][0] != "Milk, bread, eggs, cheese" {
		t.Fatalf("expected updated body, got %v", resp.Rows)
	}
}

func testDBRejectNonSelect(t *testing.T, b Backend) {
	_, err := b.Client.DB("applenotes", "DROP TABLE applenotes_notes")
	if err == nil {
		t.Fatal("expected error for DROP query")
	}
}

func testDBRejectUnknownSource(t *testing.T, b Backend) {
	_, err := b.Client.DB("nonexistent", "SELECT 1")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func testGmailSendValidation(t *testing.T, b Backend) {
	// Empty recipients should fail
	_, err := b.Client.GmailSend(nil, nil, nil, "test", "body", "")
	if err == nil {
		t.Fatal("expected error for empty recipients")
	}

	// No auth should fail
	if b.NoAuthClient != nil {
		_, err = b.NoAuthClient.GmailSend([]string{"a@b.com"}, nil, nil, "test", "body", "")
		if err == nil {
			t.Fatal("expected auth error for gmail send")
		}
	}
}

func testWhatsAppSendValidation(t *testing.T, b Backend) {
	// Empty to should fail
	_, err := b.Client.WhatsAppSend("", "hello")
	if err == nil {
		t.Fatal("expected error for empty to")
	}

	// Empty text should fail
	_, err = b.Client.WhatsAppSend("123@s.whatsapp.net", "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}

	// No auth should fail
	if b.NoAuthClient != nil {
		_, err = b.NoAuthClient.WhatsAppSend("123@s.whatsapp.net", "hello")
		if err == nil {
			t.Fatal("expected auth error for whatsapp send")
		}
	}
}

func testDBSeededQueries(t *testing.T, b Backend) {
	b.SeedDB(t, "gmail", `
		CREATE TABLE IF NOT EXISTS emails (
			id INTEGER PRIMARY KEY,
			account TEXT,
			from_addr TEXT,
			subject TEXT,
			body TEXT
		);
		INSERT INTO emails VALUES (1, 'user@gmail.com', 'alice@example.com', 'Meeting Tomorrow', 'Lets meet at 2pm.');
		INSERT INTO emails VALUES (2, 'user@gmail.com', 'bob@example.com', 'Project Update', 'Here is the latest.');
		INSERT INTO emails VALUES (3, 'work@gmail.com', 'charlie@example.com', 'Invoice #123', 'Please find attached.');
	`)

	// Query with columns
	resp, err := b.Client.DB("gmail", "SELECT id, from_addr, subject FROM emails ORDER BY id")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(resp.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d: %v", len(resp.Columns), resp.Columns)
	}
	if len(resp.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(resp.Rows))
	}
	if resp.Rows[0][2] != "Meeting Tomorrow" {
		t.Fatalf("expected 'Meeting Tomorrow', got %q", resp.Rows[0][2])
	}

	// Filtered query
	resp, err = b.Client.DB("gmail", "SELECT subject FROM emails WHERE account = 'work@gmail.com'")
	if err != nil {
		t.Fatalf("filtered query: %v", err)
	}
	if len(resp.Rows) != 1 || resp.Rows[0][0] != "Invoice #123" {
		t.Fatalf("unexpected filtered result: %v", resp.Rows)
	}

	// Count
	resp, err = b.Client.DB("gmail", "SELECT COUNT(*) as total FROM emails")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if resp.Rows[0][0] != "3" {
		t.Fatalf("expected count 3, got %q", resp.Rows[0][0])
	}
}
