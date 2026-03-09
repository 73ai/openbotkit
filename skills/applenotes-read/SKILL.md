---
name: applenotes-read
description: Search Apple Notes, find notes by title or content, browse notes by folder
allowed-tools: Bash(obk *)
---

## Schema

Full database schema: see schema.sql in this skill directory.

## Query patterns

```bash
# Recent notes
obk db applenotes "SELECT modified_at, folder, title FROM applenotes_notes ORDER BY modified_at DESC LIMIT 20;"

# Search by title
obk db applenotes "SELECT modified_at, folder, title FROM applenotes_notes WHERE LOWER(title) LIKE '%keyword%' ORDER BY modified_at DESC LIMIT 20;"

# Full text search across title and body
obk db applenotes "SELECT modified_at, folder, title, substr(body, 1, 200) FROM applenotes_notes WHERE LOWER(title) LIKE '%term%' OR LOWER(body) LIKE '%term%' ORDER BY modified_at DESC LIMIT 10;"

# Read full note
obk db applenotes "SELECT title, folder, account, created_at, modified_at, body FROM applenotes_notes WHERE id = <id>;"

# Notes in a specific folder
obk db applenotes "SELECT modified_at, title FROM applenotes_notes WHERE LOWER(folder) = 'notes' ORDER BY modified_at DESC LIMIT 20;"

# List all folders
obk db applenotes "SELECT name, account, (SELECT COUNT(*) FROM applenotes_notes WHERE folder_id = f.apple_id) as note_count FROM applenotes_folders f ORDER BY name;"

# Notes by account
obk db applenotes "SELECT account, COUNT(*) FROM applenotes_notes GROUP BY account;"

# Recently modified notes (last 7 days)
obk db applenotes "SELECT modified_at, folder, title FROM applenotes_notes WHERE modified_at >= datetime('now', '-7 days') ORDER BY modified_at DESC;"
```
