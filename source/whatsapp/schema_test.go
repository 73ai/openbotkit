package whatsapp

import (
	"path/filepath"
	"testing"

	"github.com/73ai/openbotkit/store"
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

	// Verify tables exist by querying them.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM whatsapp_messages").Scan(&count); err != nil {
		t.Fatalf("query messages table: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM whatsapp_chats").Scan(&count); err != nil {
		t.Fatalf("query chats table: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM whatsapp_contacts").Scan(&count); err != nil {
		t.Fatalf("query contacts table: %v", err)
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

func TestMigrateUniqueConstraint(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Insert a message.
	_, err := db.Exec(`INSERT INTO whatsapp_messages (message_id, chat_jid, text) VALUES ('msg1', 'chat1', 'hello')`)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// Same message_id + chat_jid should conflict.
	_, err = db.Exec(`INSERT INTO whatsapp_messages (message_id, chat_jid, text) VALUES ('msg1', 'chat1', 'world')`)
	if err == nil {
		t.Fatal("expected unique constraint violation, got nil")
	}

	// Different chat_jid should succeed.
	_, err = db.Exec(`INSERT INTO whatsapp_messages (message_id, chat_jid, text) VALUES ('msg1', 'chat2', 'world')`)
	if err != nil {
		t.Fatalf("different chat_jid should succeed: %v", err)
	}
}

const legacySchemaSQLite = `
CREATE TABLE whatsapp_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id TEXT NOT NULL,
	chat_jid TEXT NOT NULL,
	sender_jid TEXT,
	sender_name TEXT,
	text TEXT,
	timestamp DATETIME,
	media_type TEXT,
	media_path TEXT,
	is_group INTEGER DEFAULT 0,
	is_from_me INTEGER DEFAULT 0,
	reply_to_id TEXT,
	synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(message_id, chat_jid)
);
CREATE TABLE whatsapp_chats (
	jid TEXT PRIMARY KEY,
	name TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_at DATETIME
);
CREATE TABLE whatsapp_contacts (
	jid TEXT PRIMARY KEY,
	phone TEXT,
	first_name TEXT,
	full_name TEXT,
	push_name TEXT,
	business_name TEXT,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
INSERT INTO whatsapp_messages (id, message_id, chat_jid, text) VALUES (7, 'm1', 'chat@g.us', 'old message');
INSERT INTO whatsapp_chats (jid, name, is_group) VALUES ('chat@g.us', 'Old Group', 1);
INSERT INTO whatsapp_contacts (jid, phone, first_name, full_name, push_name, business_name) VALUES ('alice@s.whatsapp.net', '111', 'Alice', 'Alice', '', '');
`

func TestMigrateLegacyTablesKeepsDataUnderDefaultAccount(t *testing.T) {
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "data.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(legacySchemaSQLite); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate legacy: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var id int
	var account, text string
	if err := db.QueryRow("SELECT id, account, text FROM whatsapp_messages WHERE message_id = 'm1'").Scan(&id, &account, &text); err != nil {
		t.Fatalf("read migrated message: %v", err)
	}
	if id != 7 || account != DefaultAccount || text != "old message" {
		t.Fatalf("migrated message = (%d, %q, %q)", id, account, text)
	}

	chats, err := ListChats(db, DefaultAccount)
	if err != nil || len(chats) != 1 || chats[0].Name != "Old Group" {
		t.Fatalf("migrated chats = %+v, err %v", chats, err)
	}
	contact, err := GetContact(db, DefaultAccount, "alice@s.whatsapp.net")
	if err != nil || contact == nil || contact.FullName != "Alice" {
		t.Fatalf("migrated contact = %+v, err %v", contact, err)
	}

	if err := SaveMessage(db, &Message{Account: "work", MessageID: "m2", ChatJID: "chat@g.us"}); err != nil {
		t.Fatalf("save after migrate: %v", err)
	}
	var nextID int
	if err := db.QueryRow("SELECT id FROM whatsapp_messages WHERE message_id = 'm2'").Scan(&nextID); err != nil {
		t.Fatalf("read new message: %v", err)
	}
	if nextID <= 7 {
		t.Fatalf("new message id %d reuses a migrated id", nextID)
	}

	var legacyTables int
	db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name LIKE '%_legacy'").Scan(&legacyTables)
	if legacyTables != 0 {
		t.Fatalf("legacy tables left behind: %d", legacyTables)
	}
}

func TestMigrateUniquePerAccount(t *testing.T) {
	db := testDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	insert := `INSERT INTO whatsapp_messages (account, message_id, chat_jid, text) VALUES (?, 'msg1', 'chat1', 'hi')`
	if _, err := db.Exec(insert, "default"); err != nil {
		t.Fatalf("insert default: %v", err)
	}
	if _, err := db.Exec(insert, "work"); err != nil {
		t.Fatalf("same message in another account should succeed: %v", err)
	}
	if _, err := db.Exec(insert, "work"); err == nil {
		t.Fatal("expected unique violation within one account")
	}
}
