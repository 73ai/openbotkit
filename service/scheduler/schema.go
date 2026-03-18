package scheduler

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS schedules (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	type          TEXT NOT NULL,
	cron_expr     TEXT,
	scheduled_at  TEXT,
	task          TEXT NOT NULL,
	channel       TEXT NOT NULL,
	channel_meta  TEXT,
	timezone      TEXT NOT NULL,
	description   TEXT,
	enabled       INTEGER NOT NULL DEFAULT 1,
	last_run_at   TEXT,
	last_error    TEXT,
	created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	completed_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_schedules_type ON schedules(type);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS schedules (
	id            BIGSERIAL PRIMARY KEY,
	type          TEXT NOT NULL,
	cron_expr     TEXT,
	scheduled_at  TIMESTAMPTZ,
	task          TEXT NOT NULL,
	channel       TEXT NOT NULL,
	channel_meta  TEXT,
	timezone      TEXT NOT NULL,
	description   TEXT,
	enabled       BOOLEAN NOT NULL DEFAULT TRUE,
	last_run_at   TIMESTAMPTZ,
	last_error    TEXT,
	created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	completed_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_schedules_type ON schedules(type);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
