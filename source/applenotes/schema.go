package applenotes

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS applenotes_folders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	apple_id TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	parent_apple_id TEXT,
	account TEXT
);

CREATE TABLE IF NOT EXISTS applenotes_notes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	apple_id TEXT NOT NULL UNIQUE,
	title TEXT,
	body TEXT,
	folder TEXT,
	folder_id TEXT,
	account TEXT,
	password_protected INTEGER DEFAULT 0,
	created_at DATETIME,
	modified_at DATETIME,
	synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_applenotes_notes_apple_id ON applenotes_notes(apple_id);
CREATE INDEX IF NOT EXISTS idx_applenotes_notes_folder ON applenotes_notes(folder);
CREATE INDEX IF NOT EXISTS idx_applenotes_notes_modified ON applenotes_notes(modified_at);
CREATE INDEX IF NOT EXISTS idx_applenotes_folders_apple_id ON applenotes_folders(apple_id);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS applenotes_folders (
	id BIGSERIAL PRIMARY KEY,
	apple_id TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	parent_apple_id TEXT,
	account TEXT
);

CREATE TABLE IF NOT EXISTS applenotes_notes (
	id BIGSERIAL PRIMARY KEY,
	apple_id TEXT NOT NULL UNIQUE,
	title TEXT,
	body TEXT,
	folder TEXT,
	folder_id TEXT,
	account TEXT,
	password_protected BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMPTZ,
	modified_at TIMESTAMPTZ,
	synced_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_applenotes_notes_apple_id ON applenotes_notes(apple_id);
CREATE INDEX IF NOT EXISTS idx_applenotes_notes_folder ON applenotes_notes(folder);
CREATE INDEX IF NOT EXISTS idx_applenotes_notes_modified ON applenotes_notes(modified_at);
CREATE INDEX IF NOT EXISTS idx_applenotes_folders_apple_id ON applenotes_folders(apple_id);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
