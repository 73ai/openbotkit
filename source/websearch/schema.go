package websearch

import "github.com/priyanshujain/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS search_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	query TEXT NOT NULL,
	result_count INTEGER NOT NULL DEFAULT 0,
	backends TEXT,
	search_ms INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_search_history_query ON search_history(query);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS search_history (
	id BIGSERIAL PRIMARY KEY,
	query TEXT NOT NULL,
	result_count INTEGER NOT NULL DEFAULT 0,
	backends TEXT,
	search_ms INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_history_query ON search_history(query);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
