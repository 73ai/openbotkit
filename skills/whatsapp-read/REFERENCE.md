## Schema

Full database schema: see schema.sql in this skill directory.

## Query patterns

```bash
# List chats
obk db whatsapp "SELECT jid, name, is_group, last_message_at FROM whatsapp_chats ORDER BY last_message_at DESC LIMIT 20;"

# Recent messages in a chat
obk db whatsapp "SELECT timestamp, sender_name, text FROM whatsapp_messages WHERE chat_jid = '<jid>' ORDER BY timestamp DESC LIMIT 30;"

# Search messages by text
obk db whatsapp "SELECT timestamp, sender_name, text FROM whatsapp_messages WHERE LOWER(text) LIKE '%keyword%' ORDER BY timestamp DESC LIMIT 20;"

# Messages from a specific person (always look up contact first, sender_name is often empty)
# Step 1: Find their JID
obk db whatsapp "SELECT jid, full_name, push_name FROM whatsapp_contacts WHERE LOWER(full_name) LIKE '%name%' OR LOWER(push_name) LIKE '%name%' OR LOWER(first_name) LIKE '%name%';"
# Step 2: Query messages by their JID (use chat_jid for 1:1 chats, sender_jid for groups)
obk db whatsapp "SELECT timestamp, text FROM whatsapp_messages WHERE chat_jid = '<jid>' ORDER BY timestamp DESC LIMIT 20;"

# Shortcut: join contacts and messages to find messages by person name
obk db whatsapp "SELECT m.timestamp, m.text FROM whatsapp_messages m JOIN whatsapp_contacts c ON m.chat_jid = c.jid WHERE LOWER(c.full_name) LIKE '%name%' OR LOWER(c.push_name) LIKE '%name%' ORDER BY m.timestamp DESC LIMIT 20;"

# My sent messages
obk db whatsapp "SELECT timestamp, chat_jid, text FROM whatsapp_messages WHERE is_from_me = 1 ORDER BY timestamp DESC LIMIT 20;"

# Group messages
obk db whatsapp "SELECT m.timestamp, m.sender_name, m.text FROM whatsapp_messages m JOIN whatsapp_chats c ON c.jid = m.chat_jid WHERE c.name LIKE '%group name%' ORDER BY m.timestamp DESC LIMIT 30;"

# Messages with media
obk db whatsapp "SELECT timestamp, sender_name, media_type, media_path FROM whatsapp_messages WHERE media_type IS NOT NULL AND media_type != '' ORDER BY timestamp DESC LIMIT 20;"

# Message count per chat
obk db whatsapp "SELECT c.name, COUNT(*) as cnt FROM whatsapp_messages m JOIN whatsapp_chats c ON c.jid = m.chat_jid GROUP BY m.chat_jid ORDER BY cnt DESC LIMIT 20;"

# Look up a contact by name
obk db whatsapp "SELECT jid, phone, full_name, push_name, business_name FROM whatsapp_contacts WHERE LOWER(full_name) LIKE '%name%' OR LOWER(push_name) LIKE '%name%' OR LOWER(first_name) LIKE '%name%';"

# Look up a contact by phone number
obk db whatsapp "SELECT jid, phone, full_name, push_name FROM whatsapp_contacts WHERE phone LIKE '%number%';"
```
