package imessage

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS imessage_handles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	handle_id TEXT NOT NULL UNIQUE,
	service TEXT
);

CREATE TABLE IF NOT EXISTS imessage_chats (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	guid TEXT NOT NULL UNIQUE,
	display_name TEXT,
	service_name TEXT,
	participants_json TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_date DATETIME
);

CREATE TABLE IF NOT EXISTS imessage_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	apple_rowid INTEGER UNIQUE,
	guid TEXT NOT NULL UNIQUE,
	text TEXT,
	chat_guid TEXT,
	sender_id TEXT,
	sender_service TEXT,
	is_from_me INTEGER DEFAULT 0,
	is_read INTEGER DEFAULT 0,
	date_utc DATETIME,
	date_read_utc DATETIME,
	reply_to_guid TEXT,
	associated_msg_guid TEXT,
	associated_msg_type INTEGER DEFAULT 0,
	attachments_json TEXT,
	chat_display_name TEXT,
	synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_imessage_messages_guid ON imessage_messages(guid);
CREATE INDEX IF NOT EXISTS idx_imessage_messages_chat_guid ON imessage_messages(chat_guid);
CREATE INDEX IF NOT EXISTS idx_imessage_messages_date ON imessage_messages(date_utc);
CREATE INDEX IF NOT EXISTS idx_imessage_messages_apple_rowid ON imessage_messages(apple_rowid);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS imessage_handles (
	id BIGSERIAL PRIMARY KEY,
	handle_id TEXT NOT NULL UNIQUE,
	service TEXT
);

CREATE TABLE IF NOT EXISTS imessage_chats (
	id BIGSERIAL PRIMARY KEY,
	guid TEXT NOT NULL UNIQUE,
	display_name TEXT,
	service_name TEXT,
	participants_json TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_date TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS imessage_messages (
	id BIGSERIAL PRIMARY KEY,
	apple_rowid BIGINT UNIQUE,
	guid TEXT NOT NULL UNIQUE,
	text TEXT,
	chat_guid TEXT,
	sender_id TEXT,
	sender_service TEXT,
	is_from_me BOOLEAN DEFAULT FALSE,
	is_read BOOLEAN DEFAULT FALSE,
	date_utc TIMESTAMPTZ,
	date_read_utc TIMESTAMPTZ,
	reply_to_guid TEXT,
	associated_msg_guid TEXT,
	associated_msg_type INTEGER DEFAULT 0,
	attachments_json TEXT,
	chat_display_name TEXT,
	synced_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_imessage_messages_guid ON imessage_messages(guid);
CREATE INDEX IF NOT EXISTS idx_imessage_messages_chat_guid ON imessage_messages(chat_guid);
CREATE INDEX IF NOT EXISTS idx_imessage_messages_date ON imessage_messages(date_utc);
CREATE INDEX IF NOT EXISTS idx_imessage_messages_apple_rowid ON imessage_messages(apple_rowid);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
