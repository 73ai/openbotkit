CREATE TABLE IF NOT EXISTS memory_conversations (
  id INTEGER PRIMARY KEY,
  session_id TEXT NOT NULL UNIQUE,
  cwd TEXT,
  started_at DATETIME,
  updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS memory_messages (
  id INTEGER PRIMARY KEY,
  conversation_id INTEGER REFERENCES memory_conversations(id),
  role TEXT NOT NULL,
  content TEXT,
  timestamp DATETIME
);

CREATE INDEX IF NOT EXISTS idx_memory_conversations_session_id ON memory_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_memory_messages_conversation_id ON memory_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_memory_messages_role ON memory_messages(role);
