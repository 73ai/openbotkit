CREATE TABLE IF NOT EXISTS whatsapp_messages (
  id INTEGER PRIMARY KEY,
  account TEXT NOT NULL DEFAULT 'default',
  message_id TEXT NOT NULL,
  chat_jid TEXT NOT NULL,
  sender_jid TEXT,
  sender_name TEXT,
  text TEXT,
  timestamp DATETIME,
  media_type TEXT,
  media_path TEXT,
  is_group INTEGER DEFAULT 0,
  is_from_me INTEGER DEFAULT 0,
  reply_to_id TEXT,
  synced_at DATETIME,
  UNIQUE(account, message_id, chat_jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_chats (
  account TEXT NOT NULL DEFAULT 'default',
  jid TEXT NOT NULL,
  name TEXT,
  is_group INTEGER DEFAULT 0,
  last_message_at DATETIME,
  PRIMARY KEY(account, jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_contacts (
  account TEXT NOT NULL DEFAULT 'default',
  jid TEXT NOT NULL,
  phone TEXT,
  first_name TEXT,
  full_name TEXT,
  push_name TEXT,
  business_name TEXT,
  updated_at DATETIME,
  PRIMARY KEY(account, jid)
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_account ON whatsapp_messages(account);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
