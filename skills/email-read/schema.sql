CREATE TABLE IF NOT EXISTS gmail_emails (
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

CREATE TABLE IF NOT EXISTS gmail_attachments (
  id INTEGER PRIMARY KEY,
  email_id INTEGER REFERENCES gmail_emails(id),
  filename TEXT,
  mime_type TEXT,
  saved_path TEXT
);

CREATE INDEX IF NOT EXISTS idx_gmail_emails_account ON gmail_emails(account);
CREATE INDEX IF NOT EXISTS idx_gmail_emails_date ON gmail_emails(date);
CREATE INDEX IF NOT EXISTS idx_gmail_emails_from_addr ON gmail_emails(from_addr);
