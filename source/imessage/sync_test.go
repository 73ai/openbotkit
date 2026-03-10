package imessage

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/priyanshujain/openbotkit/store"
)

// createMockChatDB creates a file-backed SQLite that mimics Apple's chat.db schema
// and returns both a persistent connection (for inserting more data in tests) and
// the path (so OpenChatDB can open fresh connections each time Sync is called).
func createMockChatDB(t *testing.T) (db *sql.DB, dbPath string) {
	t.Helper()

	dbPath = filepath.Join(t.TempDir(), "mock_chat.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open mock chat.db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
		CREATE TABLE handle (
			ROWID INTEGER PRIMARY KEY,
			id TEXT,
			service TEXT
		);
		CREATE TABLE chat (
			ROWID INTEGER PRIMARY KEY,
			guid TEXT,
			display_name TEXT,
			service_name TEXT
		);
		CREATE TABLE chat_handle_join (
			chat_id INTEGER,
			handle_id INTEGER
		);
		CREATE TABLE message (
			ROWID INTEGER PRIMARY KEY,
			guid TEXT,
			text TEXT,
			handle_id INTEGER,
			is_from_me INTEGER DEFAULT 0,
			is_read INTEGER DEFAULT 0,
			date INTEGER DEFAULT 0,
			date_read INTEGER DEFAULT 0,
			cache_has_attachments INTEGER DEFAULT 0,
			thread_originator_guid TEXT,
			associated_message_guid TEXT,
			associated_message_type INTEGER DEFAULT 0
		);
		CREATE TABLE chat_message_join (
			chat_id INTEGER,
			message_id INTEGER
		);
		CREATE TABLE attachment (
			ROWID INTEGER PRIMARY KEY,
			filename TEXT,
			mime_type TEXT,
			total_bytes INTEGER
		);
		CREATE TABLE message_attachment_join (
			message_id INTEGER,
			attachment_id INTEGER
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create mock schema: %v", err)
	}

	// Insert test handles
	db.Exec(`INSERT INTO handle (ROWID, id, service) VALUES (1, '+1234567890', 'iMessage')`)
	db.Exec(`INSERT INTO handle (ROWID, id, service) VALUES (2, '+0987654321', 'SMS')`)

	// Insert test chat
	db.Exec(`INSERT INTO chat (ROWID, guid, display_name, service_name)
		VALUES (1, 'iMessage;-;+1234567890', 'John', 'iMessage')`)
	db.Exec(`INSERT INTO chat_handle_join (chat_id, handle_id) VALUES (1, 1)`)

	// Insert test messages (date in Apple nanoseconds)
	// 2024-01-15 12:00:00 UTC -> Apple nanos: 727012800_000_000_000
	db.Exec(`INSERT INTO message (ROWID, guid, text, handle_id, is_from_me, is_read, date)
		VALUES (1, 'msg-guid-001', 'Hello from John', 1, 0, 1, 727012800000000000)`)
	db.Exec(`INSERT INTO chat_message_join (chat_id, message_id) VALUES (1, 1)`)

	db.Exec(`INSERT INTO message (ROWID, guid, text, handle_id, is_from_me, is_read, date)
		VALUES (2, 'msg-guid-002', 'Hi John!', 0, 1, 1, 727012860000000000)`)
	db.Exec(`INSERT INTO chat_message_join (chat_id, message_id) VALUES (1, 2)`)

	// A message with NULL text (should be skipped)
	db.Exec(`INSERT INTO message (ROWID, guid, text, handle_id, is_from_me, date)
		VALUES (3, 'msg-guid-003', NULL, 1, 0, 727012920000000000)`)
	db.Exec(`INSERT INTO chat_message_join (chat_id, message_id) VALUES (1, 3)`)

	return db, dbPath
}

func TestSync(t *testing.T) {
	_, mockPath := createMockChatDB(t)

	original := OpenChatDB
	OpenChatDB = func() (*sql.DB, error) { return sql.Open("sqlite3", mockPath+"?mode=ro") }
	t.Cleanup(func() { OpenChatDB = original })

	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open app db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	result, err := Sync(db, SyncOptions{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if result.Synced != 2 {
		t.Errorf("synced = %d, want 2", result.Synced)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}

	count, err := CountMessages(db)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	msg, err := GetMessage(db, "msg-guid-001")
	if err != nil {
		t.Fatalf("get message: %v", err)
	}
	if msg.Text != "Hello from John" {
		t.Errorf("text = %q, want %q", msg.Text, "Hello from John")
	}
	if msg.SenderID != "+1234567890" {
		t.Errorf("sender = %q, want %q", msg.SenderID, "+1234567890")
	}
}

func TestSyncIncremental(t *testing.T) {
	mockDB, mockPath := createMockChatDB(t)

	original := OpenChatDB
	OpenChatDB = func() (*sql.DB, error) { return sql.Open("sqlite3", mockPath+"?mode=ro") }
	t.Cleanup(func() { OpenChatDB = original })

	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open app db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// First sync
	result1, err := Sync(db, SyncOptions{})
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if result1.Synced != 2 {
		t.Errorf("first sync: synced = %d, want 2", result1.Synced)
	}

	// Add a new message to mock chat.db via the persistent connection
	mockDB.Exec(`INSERT INTO message (ROWID, guid, text, handle_id, is_from_me, date)
		VALUES (4, 'msg-guid-004', 'New message', 1, 0, 727013000000000000)`)
	mockDB.Exec(`INSERT INTO chat_message_join (chat_id, message_id) VALUES (1, 4)`)

	// Second sync (incremental) — should only get new message
	result2, err := Sync(db, SyncOptions{})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if result2.Synced != 1 {
		t.Errorf("second sync: synced = %d, want 1", result2.Synced)
	}

	count, _ := CountMessages(db)
	if count != 3 {
		t.Errorf("total count = %d, want 3", count)
	}
}

func TestSyncFull(t *testing.T) {
	_, mockPath := createMockChatDB(t)

	original := OpenChatDB
	OpenChatDB = func() (*sql.DB, error) { return sql.Open("sqlite3", mockPath+"?mode=ro") }
	t.Cleanup(func() { OpenChatDB = original })

	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("open app db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// First sync
	Sync(db, SyncOptions{})

	// Full sync should re-fetch everything
	result, err := Sync(db, SyncOptions{Full: true})
	if err != nil {
		t.Fatalf("full sync: %v", err)
	}
	if result.Synced != 2 {
		t.Errorf("full sync: synced = %d, want 2", result.Synced)
	}
}
