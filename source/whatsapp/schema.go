package whatsapp

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/73ai/openbotkit/store"
)

const messagesTableSQLite = `
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
);`

const chatsTableSQLite = `
CREATE TABLE IF NOT EXISTS whatsapp_chats (
	account TEXT NOT NULL DEFAULT 'default',
	jid TEXT NOT NULL,
	name TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_at DATETIME,
	PRIMARY KEY(account, jid)
);`

const contactsTableSQLite = `
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
);`

const indexes = `
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_account ON whatsapp_messages(account);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
`

const schemaPostgres = `
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
`

type accountTable struct {
	name          string
	createSQLite  string
	columns       string
	oldConstraint string
	newConstraint string
}

var accountTables = []accountTable{
	{
		name:          "whatsapp_messages",
		createSQLite:  messagesTableSQLite,
		columns:       "id, message_id, chat_jid, sender_jid, sender_name, text, timestamp, media_type, media_path, is_group, is_from_me, reply_to_id, synced_at",
		oldConstraint: "whatsapp_messages_message_id_chat_jid_key",
		newConstraint: "UNIQUE(account, message_id, chat_jid)",
	},
	{
		name:          "whatsapp_chats",
		createSQLite:  chatsTableSQLite,
		columns:       "jid, name, is_group, last_message_at",
		oldConstraint: "whatsapp_chats_pkey",
		newConstraint: "PRIMARY KEY(account, jid)",
	},
	{
		name:          "whatsapp_contacts",
		createSQLite:  contactsTableSQLite,
		columns:       "jid, phone, first_name, full_name, push_name, business_name, updated_at",
		oldConstraint: "whatsapp_contacts_pkey",
		newConstraint: "PRIMARY KEY(account, jid)",
	},
}

func Migrate(db *store.DB) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	begin := []string{"PRAGMA busy_timeout=30000", "BEGIN IMMEDIATE"}
	schema := messagesTableSQLite + chatsTableSQLite + contactsTableSQLite + indexes
	if db.IsPostgres() {
		begin = []string{"BEGIN", "SELECT pg_advisory_xact_lock(hashtext('whatsapp_migrate'))"}
		schema = schemaPostgres + indexes
	}
	for _, stmt := range begin {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	defer conn.ExecContext(ctx, "ROLLBACK")

	for _, t := range accountTables {
		if err := addAccountColumn(ctx, conn, db, t); err != nil {
			return fmt.Errorf("add account to %s: %w", t.name, err)
		}
	}
	if _, err := conn.ExecContext(ctx, schema); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}

func addAccountColumn(ctx context.Context, conn *sql.Conn, db *store.DB, t accountTable) error {
	tableQuery := "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?"
	columnQuery := "SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'account'"
	if db.IsPostgres() {
		tableQuery = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1"
		columnQuery = "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = 'account'"
	}
	var tables, columns int
	if err := conn.QueryRowContext(ctx, tableQuery, t.name).Scan(&tables); err != nil || tables == 0 {
		return err
	}
	if err := conn.QueryRowContext(ctx, columnQuery, t.name).Scan(&columns); err != nil || columns > 0 {
		return err
	}

	stmts := []string{
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN account TEXT NOT NULL DEFAULT 'default'", t.name),
		fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", t.name, t.oldConstraint),
		fmt.Sprintf("ALTER TABLE %s ADD %s", t.name, t.newConstraint),
	}
	if db.IsSQLite() {
		legacy := t.name + "_legacy"
		stmts = []string{
			fmt.Sprintf("ALTER TABLE %s RENAME TO %s", t.name, legacy),
			t.createSQLite,
			fmt.Sprintf("INSERT INTO %s (account, %s) SELECT 'default', %s FROM %s", t.name, t.columns, t.columns, legacy),
			fmt.Sprintf("DROP TABLE %s", legacy),
		}
	}
	for _, stmt := range stmts {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
