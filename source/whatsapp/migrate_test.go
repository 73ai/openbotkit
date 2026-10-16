package whatsapp

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/73ai/openbotkit/store"
	"github.com/73ai/openbotkit/store/migratetest"
)

const legacySchemaSQLite = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
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

CREATE TABLE IF NOT EXISTS whatsapp_chats (
	jid TEXT PRIMARY KEY,
	name TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_at DATETIME
);

CREATE TABLE IF NOT EXISTS whatsapp_contacts (
	jid TEXT PRIMARY KEY,
	phone TEXT,
	first_name TEXT,
	full_name TEXT,
	push_name TEXT,
	business_name TEXT,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
`

const legacySchemaPostgres = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
	id BIGSERIAL PRIMARY KEY,
	message_id TEXT NOT NULL,
	chat_jid TEXT NOT NULL,
	sender_jid TEXT,
	sender_name TEXT,
	text TEXT,
	timestamp TIMESTAMPTZ,
	media_type TEXT,
	media_path TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	is_from_me BOOLEAN DEFAULT FALSE,
	reply_to_id TEXT,
	synced_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(message_id, chat_jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_chats (
	jid TEXT PRIMARY KEY,
	name TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	last_message_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS whatsapp_contacts (
	jid TEXT PRIMARY KEY,
	phone TEXT,
	first_name TEXT,
	full_name TEXT,
	push_name TEXT,
	business_name TEXT,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
`

const accountSchemaSQLite = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account TEXT NOT NULL DEFAULT 'default',
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
	UNIQUE(account, message_id, chat_jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_chats (
	account TEXT NOT NULL DEFAULT 'default',
	jid TEXT NOT NULL,
	name TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_at DATETIME,
	PRIMARY KEY(account, jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_contacts (
	account TEXT NOT NULL DEFAULT 'default',
	jid TEXT NOT NULL,
	phone TEXT,
	first_name TEXT,
	full_name TEXT,
	push_name TEXT,
	business_name TEXT,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(account, jid)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_account ON whatsapp_messages(account);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
`

const accountSchemaPostgres = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
	id BIGSERIAL PRIMARY KEY,
	account TEXT NOT NULL DEFAULT 'default',
	message_id TEXT NOT NULL,
	chat_jid TEXT NOT NULL,
	sender_jid TEXT,
	sender_name TEXT,
	text TEXT,
	timestamp TIMESTAMPTZ,
	media_type TEXT,
	media_path TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	is_from_me BOOLEAN DEFAULT FALSE,
	reply_to_id TEXT,
	synced_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(account, message_id, chat_jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_chats (
	account TEXT NOT NULL DEFAULT 'default',
	jid TEXT NOT NULL,
	name TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	last_message_at TIMESTAMPTZ,
	PRIMARY KEY(account, jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_contacts (
	account TEXT NOT NULL DEFAULT 'default',
	jid TEXT NOT NULL,
	phone TEXT,
	first_name TEXT,
	full_name TEXT,
	push_name TEXT,
	business_name TEXT,
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY(account, jid)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_account ON whatsapp_messages(account);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
`

const partialSchemaSQLite = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account TEXT NOT NULL DEFAULT 'default',
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
	UNIQUE(account, message_id, chat_jid)
);
CREATE TABLE IF NOT EXISTS whatsapp_chats (
	jid TEXT PRIMARY KEY,
	name TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_at DATETIME
);
`

const partialSchemaPostgres = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
	id BIGSERIAL PRIMARY KEY,
	account TEXT NOT NULL DEFAULT 'default',
	message_id TEXT NOT NULL,
	chat_jid TEXT NOT NULL,
	sender_jid TEXT,
	sender_name TEXT,
	text TEXT,
	timestamp TIMESTAMPTZ,
	media_type TEXT,
	media_path TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	is_from_me BOOLEAN DEFAULT FALSE,
	reply_to_id TEXT,
	synced_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(account, message_id, chat_jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_chats (
	jid TEXT PRIMARY KEY,
	name TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	last_message_at TIMESTAMPTZ
);
`

const partialData = `
INSERT INTO whatsapp_messages (id, message_id, chat_jid, sender_jid, sender_name, text, timestamp, is_group, is_from_me) VALUES
	(1041, '3EB0A1F27C9D4B6E81A0', '120363041234567890@g.us', '919812345678@s.whatsapp.net', 'Ravi', 'standup moved to 11', '2026-03-02 09:14:00', TRUE, FALSE),
	(1042, '3EB0A2D83F15C6B9E042', '120363041234567890@g.us', '919876543210@s.whatsapp.net', '', 'ok', '2026-03-02 09:15:00', TRUE, TRUE),
	(1187, 'BAE5F3C0D19A7E24', '919812345678@s.whatsapp.net', '919812345678@s.whatsapp.net', 'Ravi', 'call me when free', '2026-03-03 18:40:00', FALSE, FALSE);
INSERT INTO whatsapp_messages (id, message_id, chat_jid, sender_jid, text, timestamp, media_type, media_path, reply_to_id) VALUES
	(1203, 'BAE5F4A8E63B0D71', '919812345678@s.whatsapp.net', '919876543210@s.whatsapp.net', '', '2026-03-03 18:42:00', 'image', '/Users/ravi/.obk/whatsapp/media/BAE5F4A8E63B0D71.jpg', 'BAE5F3C0D19A7E24');
INSERT INTO whatsapp_chats (jid, name, is_group, last_message_at) VALUES
	('120363041234567890@g.us', 'Platform Team', TRUE, '2026-03-02 09:15:00'),
	('919812345678@s.whatsapp.net', 'Ravi', FALSE, '2026-03-03 18:42:00');
`

const legacyData = partialData + `
INSERT INTO whatsapp_contacts (jid, phone, first_name, full_name, push_name, business_name) VALUES
	('919812345678@s.whatsapp.net', '919812345678', 'Ravi', 'Ravi Kumar', 'ravi', ''),
	('14155550100@s.whatsapp.net', '14155550100', '', '', 'Acme Support', 'Acme Inc');
`

const officeData = `
INSERT INTO whatsapp_messages (id, account, message_id, chat_jid, sender_jid, sender_name, text, timestamp, is_group, is_from_me) VALUES
	(1204, 'office', '3EB0A1F27C9D4B6E81A0', '120363041234567890@g.us', '919812345678@s.whatsapp.net', 'Ravi', 'standup moved to 11', '2026-03-02 09:14:00', TRUE, FALSE),
	(1206, 'office', '3EB0D44C2E7A19F0B355', '918860012345@s.whatsapp.net', '918860012345@s.whatsapp.net', 'Meera', 'invoice attached', '2026-03-05 11:20:00', FALSE, FALSE);
INSERT INTO whatsapp_chats (account, jid, name, is_group, last_message_at) VALUES
	('office', '120363041234567890@g.us', 'Platform Team', TRUE, '2026-03-02 09:14:00'),
	('office', '918860012345@s.whatsapp.net', 'Meera', FALSE, '2026-03-05 11:20:00');
INSERT INTO whatsapp_contacts (account, jid, phone, full_name, push_name) VALUES
	('office', '919812345678@s.whatsapp.net', '919812345678', 'Ravi K (Office)', 'ravi'),
	('office', '918860012345@s.whatsapp.net', '918860012345', 'Meera Iyer', 'meera');
`

const officeSequence = `
SELECT setval('whatsapp_messages_id_seq', 1206);
`

const postgresSequence = `
SELECT setval('whatsapp_messages_id_seq', 1203);
`

const runnerBaselineSQLite = `
CREATE TABLE schema_migrations (
	source TEXT NOT NULL,
	version INTEGER NOT NULL,
	applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(source, version)
);
INSERT INTO schema_migrations (source, version, applied_at) VALUES ('whatsapp', 1, '2026-10-14 21:07:33');
`

var (
	legacyTableQueries = []string{
		"SELECT id, message_id, chat_jid, sender_jid, sender_name, text, timestamp, media_type, media_path, is_group, is_from_me, reply_to_id, synced_at FROM whatsapp_messages ORDER BY id",
		"SELECT jid, name, is_group, last_message_at FROM whatsapp_chats ORDER BY jid",
		"SELECT jid, phone, first_name, full_name, push_name, business_name, updated_at FROM whatsapp_contacts ORDER BY jid",
	}
	accountTableQueries = []string{
		"SELECT id, account, message_id, chat_jid, sender_jid, sender_name, text, timestamp, media_type, media_path, is_group, is_from_me, reply_to_id, synced_at FROM whatsapp_messages ORDER BY id",
		"SELECT account, jid, name, is_group, last_message_at FROM whatsapp_chats ORDER BY account, jid",
		"SELECT account, jid, phone, first_name, full_name, push_name, business_name, updated_at FROM whatsapp_contacts ORDER BY account, jid",
	}
	migrationAccounts = []string{DefaultAccount, "work", "family", "assistant"}
)

const (
	liveMessageID = "3EB0C7F2A19D4E6B8C21"
	groupJID      = "120363041234567890@g.us"
	raviJID       = "919812345678@s.whatsapp.net"
)

func TestMigrations(t *testing.T) {
	migratetest.Run(t, migratetest.Suite{
		Source:  "whatsapp",
		Migrate: Migrate,
		Steps:   migrations,
		Fixtures: []migratetest.Fixture{
			{Name: "master", SQLite: legacySchemaSQLite + legacyData, Postgres: legacySchemaPostgres + legacyData + postgresSequence, Preserve: legacyTableQueries},
			{Name: "runner_baseline", SQLite: legacySchemaSQLite + legacyData + runnerBaselineSQLite, Postgres: legacySchemaPostgres + legacyData + postgresSequence + runnerBaselineSQLite, Preserve: legacyTableQueries},
			{Name: "account_column", SQLite: accountSchemaSQLite + legacyData + officeData, Postgres: accountSchemaPostgres + legacyData + officeData + officeSequence, Preserve: accountTableQueries},
			{Name: "partial", SQLite: partialSchemaSQLite + partialData, Postgres: partialSchemaPostgres + partialData + postgresSequence, Preserve: legacyTableQueries[:2]},
		},
		Use:   useAccount,
		Check: checkAccounts,
	})
}

func useAccount(db *store.DB, worker int) error {
	account := migrationAccounts[worker]
	msg := &Message{Account: account, MessageID: liveMessageID, ChatJID: groupJID, SenderJID: raviJID, SenderName: "Ravi", Text: account, Timestamp: time.Date(2026, 3, 4, 10, 2, 0, 0, time.UTC), IsGroup: true}
	if err := SaveMessage(db, msg); err != nil {
		return err
	}
	if account == DefaultAccount {
		return nil
	}
	if err := UpsertChat(db, account, groupJID, "Platform Team ("+account+")", true); err != nil {
		return err
	}
	return SaveContact(db, &Contact{Account: account, JID: raviJID, Phone: "919812345678", FullName: "Ravi (" + account + ")"})
}

func checkAccounts(t *testing.T, db *store.DB, workers int) {
	want := map[string][]string{}
	for _, account := range migrationAccounts[:workers] {
		want["message"] = append(want["message"], account+" "+account)
		if account != DefaultAccount {
			want["chat"] = append(want["chat"], account+" Platform Team ("+account+")")
			want["contact"] = append(want["contact"], account+" Ravi ("+account+")")
		}
	}
	got := map[string][]string{
		"message": migratetest.Dump(t, db, "SELECT account, text FROM whatsapp_messages WHERE message_id = '"+liveMessageID+"'"),
		"chat":    migratetest.Dump(t, db, "SELECT account, name FROM whatsapp_chats WHERE account IN ('work', 'family', 'assistant')"),
		"contact": migratetest.Dump(t, db, "SELECT account, full_name FROM whatsapp_contacts WHERE account IN ('work', 'family', 'assistant')"),
	}
	for kind := range want {
		sort.Strings(want[kind])
		sort.Strings(got[kind])
		if !reflect.DeepEqual(got[kind], want[kind]) {
			t.Fatalf("%s rows by account = %v, want %v", kind, got[kind], want[kind])
		}
	}
	if got := migratetest.Dump(t, db, "SELECT DISTINCT account FROM whatsapp_messages WHERE message_id <> '"+liveMessageID+"' AND account NOT IN ('default', 'office')"); len(got) > 0 {
		t.Fatalf("migrated messages relabelled to %v", got)
	}

	var newID, maxMigratedID int64
	if err := db.QueryRow(db.Rebind("SELECT MIN(id) FROM whatsapp_messages WHERE message_id = ?"), liveMessageID).Scan(&newID); err != nil {
		t.Fatalf("new message id: %v", err)
	}
	if err := db.QueryRow(db.Rebind("SELECT COALESCE(MAX(id), 0) FROM whatsapp_messages WHERE message_id <> ?"), liveMessageID).Scan(&maxMigratedID); err != nil {
		t.Fatalf("max migrated id: %v", err)
	}
	if newID <= maxMigratedID {
		t.Fatalf("new message id %d collides with migrated ids up to %d", newID, maxMigratedID)
	}

	wantIndexes := []string{
		"idx_whatsapp_contacts_phone",
		"idx_whatsapp_messages_account",
		"idx_whatsapp_messages_chat",
		"idx_whatsapp_messages_sender",
		"idx_whatsapp_messages_timestamp",
	}
	query := "SELECT name FROM sqlite_master WHERE type = 'index' AND name LIKE 'idx_whatsapp_%' ORDER BY name"
	if db.IsPostgres() {
		query = "SELECT indexname FROM pg_indexes WHERE indexname LIKE 'idx_whatsapp_%' ORDER BY indexname"
	}
	if got := migratetest.Dump(t, db, query); !reflect.DeepEqual(got, wantIndexes) {
		t.Fatalf("indexes = %v, want %v", got, wantIndexes)
	}
	if db.IsSQLite() {
		if got := migratetest.Dump(t, db, "SELECT name FROM sqlite_master WHERE name LIKE '%_legacy'"); len(got) != 0 {
			t.Fatalf("legacy tables left behind: %v", got)
		}
	}
}
