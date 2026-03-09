CREATE TABLE IF NOT EXISTS history_conversations (
  id INTEGER PRIMARY KEY,
  session_id TEXT NOT NULL UNIQUE,
  cwd TEXT,
  started_at DATETIME,
  updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS history_messages (
  id INTEGER PRIMARY KEY,
  conversation_id INTEGER REFERENCES history_conversations(id),
  role TEXT NOT NULL,
  content TEXT,
  timestamp DATETIME
);

CREATE INDEX IF NOT EXISTS idx_history_conversations_session_id ON history_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_history_messages_conversation_id ON history_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_history_messages_role ON history_messages(role);
