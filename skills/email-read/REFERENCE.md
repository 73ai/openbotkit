## Schema

Full database schema: see schema.sql in this skill directory.

## Query patterns

```bash
# Recent emails
obk db gmail "SELECT date, from_addr, subject FROM emails ORDER BY date DESC LIMIT 20;"

# Search by subject
obk db gmail "SELECT date, from_addr, subject FROM emails WHERE LOWER(subject) LIKE '%keyword%' ORDER BY date DESC LIMIT 20;"

# Search by sender
obk db gmail "SELECT date, from_addr, subject FROM emails WHERE LOWER(from_addr) LIKE '%name%' ORDER BY date DESC LIMIT 20;"

# Full text search across subject and body
obk db gmail "SELECT date, from_addr, subject, substr(body, 1, 200) FROM emails WHERE LOWER(subject) LIKE '%term%' OR LOWER(body) LIKE '%term%' ORDER BY date DESC LIMIT 10;"

# Read full email
obk db gmail "SELECT from_addr, to_addr, subject, date, body FROM emails WHERE id = <id>;"

# Emails with attachments
obk db gmail "SELECT e.date, e.from_addr, e.subject, a.filename, a.mime_type FROM emails e JOIN attachments a ON a.email_id = e.id ORDER BY e.date DESC LIMIT 20;"

# Count by account
obk db gmail "SELECT account, COUNT(*) FROM emails GROUP BY account;"
```
