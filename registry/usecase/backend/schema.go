package main

import "github.com/73ai/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS users (
	id         TEXT PRIMARY KEY,
	email      TEXT NOT NULL UNIQUE,
	name       TEXT NOT NULL,
	avatar_url TEXT,
	org_name   TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS use_cases (
	id                   TEXT PRIMARY KEY,
	title                TEXT NOT NULL,
	slug                 TEXT NOT NULL UNIQUE,
	description          TEXT NOT NULL,
	domain               TEXT NOT NULL,
	industry_tags        TEXT,
	risk_level           TEXT NOT NULL DEFAULT 'medium',
	roi_potential        TEXT NOT NULL DEFAULT 'medium',
	status               TEXT NOT NULL DEFAULT 'draft',
	impl_status          TEXT NOT NULL DEFAULT 'evaluating',
	visibility           TEXT NOT NULL DEFAULT 'public',
	safety_pii           INTEGER NOT NULL DEFAULT 0,
	safety_autonomous    INTEGER NOT NULL DEFAULT 0,
	safety_blast_radius  TEXT,
	safety_oversight     TEXT,
	forked_from          TEXT REFERENCES use_cases(id),
	fork_count           INTEGER NOT NULL DEFAULT 0,
	author_id            TEXT NOT NULL REFERENCES users(id),
	created_at           DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at           DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_use_cases_domain ON use_cases(domain);
CREATE INDEX IF NOT EXISTS idx_use_cases_visibility ON use_cases(visibility);
CREATE INDEX IF NOT EXISTS idx_use_cases_author ON use_cases(author_id);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS users (
	id         TEXT PRIMARY KEY,
	email      TEXT NOT NULL UNIQUE,
	name       TEXT NOT NULL,
	avatar_url TEXT,
	org_name   TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS use_cases (
	id                   TEXT PRIMARY KEY,
	title                TEXT NOT NULL,
	slug                 TEXT NOT NULL UNIQUE,
	description          TEXT NOT NULL,
	domain               TEXT NOT NULL,
	industry_tags        TEXT,
	risk_level           TEXT NOT NULL DEFAULT 'medium',
	roi_potential        TEXT NOT NULL DEFAULT 'medium',
	status               TEXT NOT NULL DEFAULT 'draft',
	impl_status          TEXT NOT NULL DEFAULT 'evaluating',
	visibility           TEXT NOT NULL DEFAULT 'public',
	safety_pii           INTEGER NOT NULL DEFAULT 0,
	safety_autonomous    INTEGER NOT NULL DEFAULT 0,
	safety_blast_radius  TEXT,
	safety_oversight     TEXT,
	forked_from          TEXT REFERENCES use_cases(id),
	fork_count           INTEGER NOT NULL DEFAULT 0,
	author_id            TEXT NOT NULL REFERENCES users(id),
	created_at           TIMESTAMPTZ DEFAULT NOW(),
	updated_at           TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_use_cases_domain ON use_cases(domain);
CREATE INDEX IF NOT EXISTS idx_use_cases_visibility ON use_cases(visibility);
CREATE INDEX IF NOT EXISTS idx_use_cases_author ON use_cases(author_id);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
