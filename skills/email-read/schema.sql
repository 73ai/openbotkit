CREATE TABLE IF NOT EXISTS emails (
  id INTEGER PRIMARY KEY,
  message_id TEXT NOT NULL,
  account TEXT NOT NULL,
  from_addr TEXT,
  to_addr TEXT,
  subject TEXT,
  date DATETIME,
  body TEXT,
  html_body TEXT,
  fetched_at DATETIME,
  UNIQUE(message_id, account)
);

CREATE TABLE IF NOT EXISTS attachments (
  id INTEGER PRIMARY KEY,
  email_id INTEGER REFERENCES emails(id),
  filename TEXT,
  mime_type TEXT,
  saved_path TEXT
);

CREATE INDEX IF NOT EXISTS idx_emails_account ON emails(account);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date);
CREATE INDEX IF NOT EXISTS idx_emails_from_addr ON emails(from_addr);
