package history

import "github.com/priyanshujain/openbotkit/store"

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS history_conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL UNIQUE,
	cwd TEXT,
	started_at DATETIME,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS history_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER REFERENCES history_conversations(id),
	role TEXT NOT NULL,
	content TEXT,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_history_conv_session ON history_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_history_msgs_conv ON history_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_history_msgs_role ON history_messages(role);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS history_conversations (
	id BIGSERIAL PRIMARY KEY,
	session_id TEXT NOT NULL UNIQUE,
	cwd TEXT,
	started_at TIMESTAMPTZ,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS history_messages (
	id BIGSERIAL PRIMARY KEY,
	conversation_id BIGINT REFERENCES history_conversations(id),
	role TEXT NOT NULL,
	content TEXT,
	timestamp TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_history_conv_session ON history_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_history_msgs_conv ON history_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_history_msgs_role ON history_messages(role);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}
