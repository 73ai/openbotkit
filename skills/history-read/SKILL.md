---
name: history-read
description: Recall past conversations, what was discussed before, previous questions asked, conversation history
allowed-tools: Bash(sqlite3 *)
---

## Database

Path: `~/.obk/history/data.db`

## Schema

Full database schema: see schema.sql in this skill directory.

## Query patterns

```bash
# Recent conversations
sqlite3 ~/.obk/history/data.db "SELECT id, session_id, cwd, started_at, updated_at FROM history_conversations ORDER BY updated_at DESC LIMIT 10;"

# Messages in a conversation
sqlite3 ~/.obk/history/data.db "SELECT role, substr(content, 1, 200) FROM history_messages WHERE conversation_id = <id> ORDER BY timestamp;"

# Search across all conversations
sqlite3 ~/.obk/history/data.db "SELECT c.session_id, c.cwd, m.role, substr(m.content, 1, 200) FROM history_messages m JOIN history_conversations c ON c.id = m.conversation_id WHERE LOWER(m.content) LIKE '%keyword%' ORDER BY m.timestamp DESC LIMIT 20;"

# What did I ask about a topic
sqlite3 ~/.obk/history/data.db "SELECT c.cwd, m.content FROM history_messages m JOIN history_conversations c ON c.id = m.conversation_id WHERE m.role = 'user' AND LOWER(m.content) LIKE '%topic%' ORDER BY m.timestamp DESC LIMIT 10;"

# What was the assistant's response about a topic
sqlite3 ~/.obk/history/data.db "SELECT substr(m.content, 1, 500) FROM history_messages m JOIN history_conversations c ON c.id = m.conversation_id WHERE m.role = 'assistant' AND LOWER(m.content) LIKE '%topic%' ORDER BY m.timestamp DESC LIMIT 10;"

# Conversations by project directory
sqlite3 ~/.obk/history/data.db "SELECT id, session_id, started_at FROM history_conversations WHERE cwd LIKE '%project-name%' ORDER BY updated_at DESC;"

# Stats
sqlite3 ~/.obk/history/data.db "SELECT COUNT(*) as conversations FROM history_conversations; SELECT COUNT(*) as messages FROM history_messages;"
```

Always use `-header -column` or `-json` mode for readable output.
