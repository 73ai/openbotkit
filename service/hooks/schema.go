package hooks

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS event_hooks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	event_type TEXT NOT NULL,
	prompt     TEXT NOT NULL,
	channel    TEXT NOT NULL,
	model_tier TEXT NOT NULL DEFAULT 'nano',
	enabled    INTEGER NOT NULL DEFAULT 1,
	last_run_at TEXT,
	last_error  TEXT,
	created_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_event_hooks_type ON event_hooks(event_type);
CREATE INDEX IF NOT EXISTS idx_event_hooks_enabled ON event_hooks(enabled);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS event_hooks (
	id         BIGSERIAL PRIMARY KEY,
	event_type TEXT NOT NULL,
	prompt     TEXT NOT NULL,
	channel    TEXT NOT NULL,
	model_tier TEXT NOT NULL DEFAULT 'nano',
	enabled    BOOLEAN NOT NULL DEFAULT TRUE,
	last_run_at TIMESTAMPTZ,
	last_error  TEXT,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_event_hooks_type ON event_hooks(event_type);
CREATE INDEX IF NOT EXISTS idx_event_hooks_enabled ON event_hooks(enabled);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
