package contacts

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS contacts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	display_name TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS contact_identities (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	contact_id INTEGER NOT NULL REFERENCES contacts(id),
	source TEXT NOT NULL,
	identity_type TEXT NOT NULL,
	identity_value TEXT NOT NULL,
	display_name TEXT,
	raw_value TEXT,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(source, identity_type, identity_value)
);

CREATE TABLE IF NOT EXISTS contact_aliases (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	contact_id INTEGER NOT NULL REFERENCES contacts(id),
	alias TEXT NOT NULL,
	alias_lower TEXT NOT NULL,
	source TEXT,
	UNIQUE(contact_id, alias_lower)
);

CREATE TABLE IF NOT EXISTS contact_interactions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	contact_id INTEGER NOT NULL REFERENCES contacts(id),
	channel TEXT NOT NULL,
	message_count INTEGER DEFAULT 0,
	last_interaction_at DATETIME,
	UNIQUE(contact_id, channel)
);

CREATE TABLE IF NOT EXISTS contact_sync_state (
	source TEXT PRIMARY KEY,
	last_synced_at DATETIME,
	last_cursor TEXT
);

CREATE INDEX IF NOT EXISTS idx_contact_identities_value ON contact_identities(identity_value);
CREATE INDEX IF NOT EXISTS idx_contact_identities_contact ON contact_identities(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_aliases_lower ON contact_aliases(alias_lower);
CREATE INDEX IF NOT EXISTS idx_contact_aliases_contact ON contact_aliases(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_interactions_contact ON contact_interactions(contact_id);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS contacts (
	id BIGSERIAL PRIMARY KEY,
	display_name TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contact_identities (
	id BIGSERIAL PRIMARY KEY,
	contact_id BIGINT NOT NULL REFERENCES contacts(id),
	source TEXT NOT NULL,
	identity_type TEXT NOT NULL,
	identity_value TEXT NOT NULL,
	display_name TEXT,
	raw_value TEXT,
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(source, identity_type, identity_value)
);

CREATE TABLE IF NOT EXISTS contact_aliases (
	id BIGSERIAL PRIMARY KEY,
	contact_id BIGINT NOT NULL REFERENCES contacts(id),
	alias TEXT NOT NULL,
	alias_lower TEXT NOT NULL,
	source TEXT,
	UNIQUE(contact_id, alias_lower)
);

CREATE TABLE IF NOT EXISTS contact_interactions (
	id BIGSERIAL PRIMARY KEY,
	contact_id BIGINT NOT NULL REFERENCES contacts(id),
	channel TEXT NOT NULL,
	message_count INTEGER DEFAULT 0,
	last_interaction_at TIMESTAMPTZ,
	UNIQUE(contact_id, channel)
);

CREATE TABLE IF NOT EXISTS contact_sync_state (
	source TEXT PRIMARY KEY,
	last_synced_at TIMESTAMPTZ,
	last_cursor TEXT
);

CREATE INDEX IF NOT EXISTS idx_contact_identities_value ON contact_identities(identity_value);
CREATE INDEX IF NOT EXISTS idx_contact_identities_contact ON contact_identities(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_aliases_lower ON contact_aliases(alias_lower);
CREATE INDEX IF NOT EXISTS idx_contact_aliases_contact ON contact_aliases(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_interactions_contact ON contact_interactions(contact_id);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
