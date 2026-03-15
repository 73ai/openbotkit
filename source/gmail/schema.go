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

const migrateRenames = `
ALTER TABLE IF EXISTS gmail_emails RENAME TO emails;
ALTER TABLE IF EXISTS gmail_attachments RENAME TO attachments;
ALTER TABLE IF EXISTS gmail_sync_state RENAME TO sync_state;
`

func Migrate(db *store.DB) error {
	// Rename legacy tables if they exist.
	if db.IsPostgres() {
		db.Exec(migrateRenames)
	} else {
		// SQLite doesn't support ALTER TABLE IF EXISTS, so check first.
		for _, pair := range [][2]string{
			{"gmail_emails", "emails"},
			{"gmail_attachments", "attachments"},
			{"gmail_sync_state", "sync_state"},
		} {
			var n int
			err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", pair[0]).Scan(&n)
			if err == nil && n > 0 {
				db.Exec("ALTER TABLE " + pair[0] + " RENAME TO " + pair[1])
			}
		}
	}

	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
