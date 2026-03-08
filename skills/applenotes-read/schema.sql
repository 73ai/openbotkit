CREATE TABLE IF NOT EXISTS applenotes_notes (
  id INTEGER PRIMARY KEY,
  apple_id TEXT NOT NULL UNIQUE,
  title TEXT,
  body TEXT,
  folder TEXT,
  folder_id TEXT,
  account TEXT,
  password_protected INTEGER DEFAULT 0,
  created_at DATETIME,
  modified_at DATETIME,
  synced_at DATETIME
);

CREATE TABLE IF NOT EXISTS applenotes_folders (
  id INTEGER PRIMARY KEY,
  apple_id TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  parent_apple_id TEXT,
  account TEXT
);

CREATE INDEX IF NOT EXISTS idx_applenotes_notes_apple_id ON applenotes_notes(apple_id);
CREATE INDEX IF NOT EXISTS idx_applenotes_notes_folder ON applenotes_notes(folder);
CREATE INDEX IF NOT EXISTS idx_applenotes_notes_modified_at ON applenotes_notes(modified_at);
