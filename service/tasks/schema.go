package tasks

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS tasks (
	id         TEXT PRIMARY KEY,
	task       TEXT NOT NULL,
	agent      TEXT NOT NULL,
	status     TEXT NOT NULL,
	started_at TEXT NOT NULL,
	done_at    TEXT,
	output     TEXT,
	error      TEXT
);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_started_at ON tasks(started_at);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS tasks (
	id         TEXT PRIMARY KEY,
	task       TEXT NOT NULL,
	agent      TEXT NOT NULL,
	status     TEXT NOT NULL,
	started_at TIMESTAMPTZ NOT NULL,
	done_at    TIMESTAMPTZ,
	output     TEXT,
	error      TEXT
);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_started_at ON tasks(started_at);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
