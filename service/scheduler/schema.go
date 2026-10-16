package scheduler

import (
	"context"
	"database/sql"

	"github.com/73ai/openbotkit/store"
)

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

var migrations = []store.Migration{
	{Version: 1, SQLite: schemaSQLite, Postgres: schemaPostgres},
	{Version: 2, Func: addTriggerColumns},
}

func Migrate(db *store.DB) error {
	return db.Migrate("scheduler", migrations)
}

func addTriggerColumns(ctx context.Context, conn *sql.Conn, db *store.DB) error {
	for _, c := range []struct{ name, def string }{
		{"model_tier", "TEXT DEFAULT 'fast'"},
		{"max_budget_usd", "REAL DEFAULT 0"},
		{"trigger_source", "TEXT"},
		{"trigger_query", "TEXT"},
		{"last_trigger_id", "INTEGER DEFAULT 0"},
	} {
		ok, err := store.HasColumn(ctx, conn, db, "schedules", c.name)
		if err != nil {
			return err
		}
		if !ok {
			if _, err := conn.ExecContext(ctx, "ALTER TABLE schedules ADD COLUMN "+c.name+" "+c.def); err != nil {
				return err
			}
		}
	}
	return nil
}
