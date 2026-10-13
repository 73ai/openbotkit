package telegram

import (
	"fmt"

	"github.com/73ai/openbotkit/store"
)

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS telegram_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id INTEGER NOT NULL,
	chat_id INTEGER NOT NULL,
	sender_id INTEGER,
	sender_name TEXT,
	text TEXT,
	timestamp DATETIME,
	media_type TEXT,
	reply_to_id INTEGER DEFAULT 0,
	is_outgoing INTEGER DEFAULT 0,
	edit_date DATETIME,
	synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(message_id, chat_id)
);

CREATE TABLE IF NOT EXISTS telegram_chats (
	chat_id INTEGER PRIMARY KEY,
	type TEXT,
	title TEXT,
	username TEXT,
	access_hash INTEGER DEFAULT 0,
	is_group INTEGER DEFAULT 0,
	is_channel INTEGER DEFAULT 0,
	is_current_dialog INTEGER NOT NULL DEFAULT 0,
	last_message_at DATETIME
);

CREATE TABLE IF NOT EXISTS telegram_users (
	user_id INTEGER PRIMARY KEY,
	username TEXT,
	first_name TEXT,
	last_name TEXT,
	phone TEXT,
	access_hash INTEGER DEFAULT 0,
	is_bot INTEGER DEFAULT 0,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS telegram_sync_state (
	chat_id INTEGER PRIMARY KEY,
	min_id INTEGER DEFAULT 0,
	max_id INTEGER DEFAULT 0,
	backfilled_until DATETIME,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS telegram_updates_state (
	key TEXT PRIMARY KEY,
	value TEXT
);

CREATE INDEX IF NOT EXISTS idx_telegram_messages_chat ON telegram_messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_telegram_messages_timestamp ON telegram_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_telegram_messages_sender ON telegram_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_telegram_users_username ON telegram_users(username);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS telegram_messages (
	id BIGSERIAL PRIMARY KEY,
	message_id BIGINT NOT NULL,
	chat_id BIGINT NOT NULL,
	sender_id BIGINT,
	sender_name TEXT,
	text TEXT,
	timestamp TIMESTAMPTZ,
	media_type TEXT,
	reply_to_id BIGINT DEFAULT 0,
	is_outgoing BOOLEAN DEFAULT FALSE,
	edit_date TIMESTAMPTZ,
	synced_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(message_id, chat_id)
);

CREATE TABLE IF NOT EXISTS telegram_chats (
	chat_id BIGINT PRIMARY KEY,
	type TEXT,
	title TEXT,
	username TEXT,
	access_hash BIGINT DEFAULT 0,
	is_group BOOLEAN DEFAULT FALSE,
	is_channel BOOLEAN DEFAULT FALSE,
	is_current_dialog BOOLEAN NOT NULL DEFAULT FALSE,
	last_message_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS telegram_users (
	user_id BIGINT PRIMARY KEY,
	username TEXT,
	first_name TEXT,
	last_name TEXT,
	phone TEXT,
	access_hash BIGINT DEFAULT 0,
	is_bot BOOLEAN DEFAULT FALSE,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS telegram_sync_state (
	chat_id BIGINT PRIMARY KEY,
	min_id BIGINT DEFAULT 0,
	max_id BIGINT DEFAULT 0,
	backfilled_until TIMESTAMPTZ,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS telegram_updates_state (
	key TEXT PRIMARY KEY,
	value TEXT
);

CREATE INDEX IF NOT EXISTS idx_telegram_messages_chat ON telegram_messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_telegram_messages_timestamp ON telegram_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_telegram_messages_sender ON telegram_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_telegram_users_username ON telegram_users(username);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	columnQuery := "SELECT COUNT(*) FROM pragma_table_info('telegram_chats') WHERE name = 'is_current_dialog'"
	if db.IsPostgres() {
		columnQuery = "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'telegram_chats' AND column_name = 'is_current_dialog'"
	}
	var columns int
	if err := db.QueryRow(columnQuery).Scan(&columns); err != nil || columns > 0 {
		return err
	}
	if _, err := db.Exec("ALTER TABLE telegram_chats ADD COLUMN is_current_dialog BOOLEAN NOT NULL DEFAULT FALSE"); err != nil {
		return fmt.Errorf("add dialog membership: %w", err)
	}
	_, err := db.Exec("UPDATE telegram_chats SET is_current_dialog = TRUE")
	return err
}
