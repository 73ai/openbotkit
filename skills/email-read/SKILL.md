---
name: email-read
description: Search emails, check inbox, find messages, look up correspondence, check for replies
allowed-tools: Bash(obk *)
---

## Schema

Full database schema: see schema.sql in this skill directory.

## Query patterns

```bash
# Recent emails
obk db gmail "SELECT date, from_addr, subject FROM gmail_emails ORDER BY date DESC LIMIT 20;"

# Search by subject
obk db gmail "SELECT date, from_addr, subject FROM gmail_emails WHERE LOWER(subject) LIKE '%keyword%' ORDER BY date DESC LIMIT 20;"

# Search by sender
obk db gmail "SELECT date, from_addr, subject FROM gmail_emails WHERE LOWER(from_addr) LIKE '%name%' ORDER BY date DESC LIMIT 20;"

# Full text search across subject and body
obk db gmail "SELECT date, from_addr, subject, substr(body, 1, 200) FROM gmail_emails WHERE LOWER(subject) LIKE '%term%' OR LOWER(body) LIKE '%term%' ORDER BY date DESC LIMIT 10;"

# Read full email
obk db gmail "SELECT from_addr, to_addr, subject, date, body FROM gmail_emails WHERE id = <id>;"

# Emails with attachments
obk db gmail "SELECT e.date, e.from_addr, e.subject, a.filename, a.mime_type FROM gmail_emails e JOIN gmail_attachments a ON a.email_id = e.id ORDER BY e.date DESC LIMIT 20;"

# Count by account
obk db gmail "SELECT account, COUNT(*) FROM gmail_emails GROUP BY account;"
```
