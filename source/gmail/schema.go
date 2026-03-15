package gmail

import "github.com/priyanshujain/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS emails (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id TEXT NOT NULL,
	account TEXT NOT NULL,
	from_addr TEXT,
	to_addr TEXT,
	subject TEXT,
	date DATETIME,
	body TEXT,
	html_body TEXT,
	fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(message_id, account)
);

CREATE TABLE IF NOT EXISTS attachments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email_id INTEGER REFERENCES emails(id),
	filename TEXT,
	mime_type TEXT,
	saved_path TEXT
);

CREATE INDEX IF NOT EXISTS idx_emails_account ON emails(account);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date);
CREATE INDEX IF NOT EXISTS idx_emails_from ON emails(from_addr);

CREATE TABLE IF NOT EXISTS sync_state (
	account TEXT PRIMARY KEY,
	history_id INTEGER NOT NULL,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS emails (
	id BIGSERIAL PRIMARY KEY,
	message_id TEXT NOT NULL,
	account TEXT NOT NULL,
	from_addr TEXT,
	to_addr TEXT,
	subject TEXT,
	date TIMESTAMPTZ,
	body TEXT,
	html_body TEXT,
	fetched_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(message_id, account)
);

CREATE TABLE IF NOT EXISTS attachments (
	id BIGSERIAL PRIMARY KEY,
	email_id BIGINT REFERENCES emails(id),
	filename TEXT,
	mime_type TEXT,
	saved_path TEXT
);

CREATE INDEX IF NOT EXISTS idx_emails_account ON emails(account);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date);
CREATE INDEX IF NOT EXISTS idx_emails_from ON emails(from_addr);

CREATE TABLE IF NOT EXISTS sync_state (
	account TEXT PRIMARY KEY,
	history_id BIGINT NOT NULL,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
