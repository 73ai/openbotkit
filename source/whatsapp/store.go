package whatsapp

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func SaveMessage(db *store.DB, msg *Message) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO whatsapp_messages
			(message_id, chat_jid, sender_jid, sender_name, text, timestamp, media_type, media_path, is_group, is_from_me, reply_to_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(message_id, chat_jid) DO UPDATE SET
				text = excluded.text,
				sender_name = excluded.sender_name,
				media_type = excluded.media_type,
				media_path = excluded.media_path`),
		msg.MessageID, msg.ChatJID, msg.SenderJID, msg.SenderName,
		msg.Text, msg.Timestamp, msg.MediaType, msg.MediaPath,
		msg.IsGroup, msg.IsFromMe, msg.ReplyToID,
	)
	if err != nil {
		return fmt.Errorf("upsert message: %w", err)
	}
	return nil
}

func UpsertChat(db *store.DB, jid, name string, isGroup bool) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO whatsapp_chats (jid, name, is_group, last_message_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(jid) DO UPDATE SET
				name = CASE WHEN excluded.name != '' THEN excluded.name ELSE whatsapp_chats.name END,
				last_message_at = excluded.last_message_at`),
		jid, name, isGroup, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("upsert chat: %w", err)
	}
	return nil
}

func MessageExists(db *store.DB, messageID, chatJID string) (bool, error) {
	var count int
	err := db.QueryRow(
		db.Rebind("SELECT COUNT(*) FROM whatsapp_messages WHERE message_id = ? AND chat_jid = ?"),
		messageID, chatJID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check message exists: %w", err)
	}
	return count > 0, nil
}

func ListMessages(db *store.DB, opts ListOptions) ([]Message, error) {
	var conditions []string
	var args []any

	if opts.ChatJID != "" {
		conditions = append(conditions, "chat_jid = ?")
		args = append(args, opts.ChatJID)
	}
	if opts.After != "" {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, opts.After)
	}
	if opts.Before != "" {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, opts.Before)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(
		`SELECT message_id, chat_jid, sender_jid, sender_name, text, timestamp,
			media_type, media_path, is_group, is_from_me, reply_to_id
		 FROM whatsapp_messages %s ORDER BY timestamp DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, opts.Offset)

	rows, err := db.Query(db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.MessageID, &m.ChatJID, &m.SenderJID, &m.SenderName,
			&m.Text, &m.Timestamp, &m.MediaType, &m.MediaPath,
			&m.IsGroup, &m.IsFromMe, &m.ReplyToID)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func SearchMessages(db *store.DB, query string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.Query(
		db.Rebind(`SELECT message_id, chat_jid, sender_jid, sender_name, text, timestamp,
			media_type, media_path, is_group, is_from_me, reply_to_id
		 FROM whatsapp_messages
		 WHERE LOWER(text) LIKE ?
		 ORDER BY timestamp DESC LIMIT ?`),
		pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.MessageID, &m.ChatJID, &m.SenderJID, &m.SenderName,
			&m.Text, &m.Timestamp, &m.MediaType, &m.MediaPath,
			&m.IsGroup, &m.IsFromMe, &m.ReplyToID); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func CountMessages(db *store.DB, chatJID string) (int64, error) {
	query := "SELECT COUNT(*) FROM whatsapp_messages"
	var args []any
	if chatJID != "" {
		query += " WHERE chat_jid = ?"
		args = append(args, chatJID)
	}

	var count int64
	err := db.QueryRow(db.Rebind(query), args...).Scan(&count)
	return count, err
}

func LastSyncTime(db *store.DB) (*time.Time, error) {
	var raw sql.NullString
	err := db.QueryRow("SELECT MAX(synced_at) FROM whatsapp_messages").Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	for _, fmt := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	} {
		if t, err := time.Parse(fmt, raw.String); err == nil {
			return &t, nil
		}
	}
	return nil, nil
}

func ListChats(db *store.DB) ([]Chat, error) {
	rows, err := db.Query(`SELECT c.jid, c.name, c.is_group, c.last_message_at,
		(SELECT COUNT(*) FROM whatsapp_messages m WHERE m.chat_jid = c.jid) as msg_count
		FROM whatsapp_chats c ORDER BY c.last_message_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	chats := []Chat{}
	for rows.Next() {
		var c Chat
		var lastMsg sql.NullTime
		var msgCount int64
		if err := rows.Scan(&c.JID, &c.Name, &c.IsGroup, &lastMsg, &msgCount); err != nil {
			return nil, fmt.Errorf("scan chat: %w", err)
		}
		if lastMsg.Valid {
			c.LastMessageAt = &lastMsg.Time
		}
		chats = append(chats, c)
	}
	return chats, rows.Err()
}
