package websearch

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS search_cache (
	cache_key TEXT UNIQUE NOT NULL,
	query TEXT NOT NULL,
	category TEXT NOT NULL DEFAULT 'web',
	results TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS fetch_cache (
	url TEXT UNIQUE NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL DEFAULT '',
	content_type TEXT NOT NULL DEFAULT '',
	format TEXT NOT NULL DEFAULT 'markdown',
	status_code INTEGER NOT NULL DEFAULT 0,
	fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS search_cache (
	cache_key TEXT UNIQUE NOT NULL,
	query TEXT NOT NULL,
	category TEXT NOT NULL DEFAULT 'web',
	results TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fetch_cache (
	url TEXT UNIQUE NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL DEFAULT '',
	content_type TEXT NOT NULL DEFAULT '',
	format TEXT NOT NULL DEFAULT 'markdown',
	status_code INTEGER NOT NULL DEFAULT 0,
	fetched_at TIMESTAMPTZ DEFAULT NOW()
);
`

var migrations = []store.Migration{
	{Version: 1, SQLite: schemaSQLite, Postgres: schemaPostgres},
}

func Migrate(db *store.DB) error {
	return db.Migrate("websearch", migrations)
}
