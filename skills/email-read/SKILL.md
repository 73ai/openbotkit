---
name: email-read
description: Search emails, check inbox, find messages, look up correspondence, check for replies
allowed-tools: Bash(sqlite3 *)
---

## Database

Path: `~/.obk/gmail/data.db`

## Schema

Full database schema: see schema.sql in this skill directory.

## Query patterns

```bash
# Recent emails
sqlite3 ~/.obk/gmail/data.db "SELECT date, from_addr, subject FROM gmail_emails ORDER BY date DESC LIMIT 20;"

# Search by subject
sqlite3 ~/.obk/gmail/data.db "SELECT date, from_addr, subject FROM gmail_emails WHERE LOWER(subject) LIKE '%keyword%' ORDER BY date DESC LIMIT 20;"

# Search by sender
sqlite3 ~/.obk/gmail/data.db "SELECT date, from_addr, subject FROM gmail_emails WHERE LOWER(from_addr) LIKE '%name%' ORDER BY date DESC LIMIT 20;"

# Full text search across subject and body
sqlite3 ~/.obk/gmail/data.db "SELECT date, from_addr, subject, substr(body, 1, 200) FROM gmail_emails WHERE LOWER(subject) LIKE '%term%' OR LOWER(body) LIKE '%term%' ORDER BY date DESC LIMIT 10;"

# Read full email
sqlite3 ~/.obk/gmail/data.db "SELECT from_addr, to_addr, subject, date, body FROM gmail_emails WHERE id = <id>;"

# Emails with attachments
sqlite3 ~/.obk/gmail/data.db "SELECT e.date, e.from_addr, e.subject, a.filename, a.mime_type FROM gmail_emails e JOIN gmail_attachments a ON a.email_id = e.id ORDER BY e.date DESC LIMIT 20;"

# Count by account
sqlite3 ~/.obk/gmail/data.db "SELECT account, COUNT(*) FROM gmail_emails GROUP BY account;"
```

Always use `-header -column` or `-json` mode for readable output.
