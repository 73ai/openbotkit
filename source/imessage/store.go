package imessage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/73ai/openbotkit/store"
)

func SaveHandle(db *store.DB, h *Handle) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO imessage_handles (handle_id, service)
			VALUES (?, ?)
			ON CONFLICT (handle_id) DO UPDATE SET service = ?`),
		h.ID, h.Service, h.Service,
	)
	if err != nil {
		return fmt.Errorf("save handle: %w", err)
	}
	return nil
}

func SaveChat(db *store.DB, c *Chat) error {
	participantsJSON, _ := json.Marshal(c.Participants)
	isGroup := 0
	if c.IsGroup {
		isGroup = 1
	}
	_, err := db.Exec(
		db.Rebind(`INSERT INTO imessage_chats
			(guid, display_name, service_name, participants_json, is_group, last_message_date)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT (guid) DO UPDATE SET
				display_name = ?, service_name = ?, participants_json = ?,
				is_group = ?, last_message_date = ?`),
		c.GUID, c.DisplayName, c.ServiceName, string(participantsJSON), isGroup, c.LastMessageDate,
		c.DisplayName, c.ServiceName, string(participantsJSON), isGroup, c.LastMessageDate,
	)
	if err != nil {
		return fmt.Errorf("save chat: %w", err)
	}
	return nil
}

func SaveMessage(db *store.DB, m *Message) error {
	isFromMe := 0
	if m.IsFromMe {
		isFromMe = 1
	}
	isRead := 0
	if m.IsRead {
		isRead = 1
	}
	_, err := db.Exec(
		db.Rebind(`INSERT INTO imessage_messages
			(apple_rowid, guid, text, chat_guid, sender_id, sender_service,
			 is_from_me, is_read, date_utc, date_read_utc, reply_to_guid,
			 associated_msg_guid, associated_msg_type, attachments_json,
			 chat_display_name, synced_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (guid) DO UPDATE SET
				text = ?, chat_guid = ?, sender_id = ?, sender_service = ?,
				is_from_me = ?, is_read = ?, date_utc = ?, date_read_utc = ?,
				reply_to_guid = ?, associated_msg_guid = ?, associated_msg_type = ?,
				attachments_json = ?, chat_display_name = ?, synced_at = CURRENT_TIMESTAMP`),
		m.AppleROWID, m.GUID, m.Text, m.ChatGUID, m.SenderID, m.SenderService,
		isFromMe, isRead, m.Date, m.DateRead, m.ReplyToGUID,
		m.AssociatedMessageGUID, m.AssociatedMessageType, m.AttachmentsJSON,
		m.ChatDisplayName,
		m.Text, m.ChatGUID, m.SenderID, m.SenderService,
		isFromMe, isRead, m.Date, m.DateRead, m.ReplyToGUID,
		m.AssociatedMessageGUID, m.AssociatedMessageType, m.AttachmentsJSON,
		m.ChatDisplayName,
	)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

func GetMessage(db *store.DB, guid string) (*Message, error) {
	var m Message
	var isFromMe, isRead int
	err := db.QueryRow(
		db.Rebind(`SELECT guid, apple_rowid, text, chat_guid, sender_id, sender_service,
			is_from_me, is_read, date_utc, date_read_utc, reply_to_guid,
			associated_msg_guid, associated_msg_type, attachments_json, chat_display_name
			FROM imessage_messages WHERE guid = ?`),
		guid,
	).Scan(&m.GUID, &m.AppleROWID, &m.Text, &m.ChatGUID, &m.SenderID, &m.SenderService,
		&isFromMe, &isRead, &m.Date, &m.DateRead, &m.ReplyToGUID,
		&m.AssociatedMessageGUID, &m.AssociatedMessageType, &m.AttachmentsJSON, &m.ChatDisplayName)
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}
	m.IsFromMe = isFromMe != 0
	m.IsRead = isRead != 0
	return &m, nil
}

func ListMessages(db *store.DB, opts ListOptions) ([]Message, error) {
	var conditions []string
	var args []any

	if opts.ChatGUID != "" {
		conditions = append(conditions, "chat_guid = ?")
		args = append(args, opts.ChatGUID)
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
		`SELECT guid, apple_rowid, text, chat_guid, sender_id, sender_service,
			is_from_me, is_read, date_utc, date_read_utc, reply_to_guid,
			associated_msg_guid, associated_msg_type, attachments_json, chat_display_name
		FROM imessage_messages %s ORDER BY date_utc DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, opts.Offset)

	rows, err := db.Query(db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

func ListMessagesModifiedSince(db *store.DB, since time.Time) ([]Message, error) {
	rows, err := db.Query(
		db.Rebind(`SELECT guid, apple_rowid, text, chat_guid, sender_id, sender_service,
			is_from_me, is_read, date_utc, date_read_utc, reply_to_guid,
			associated_msg_guid, associated_msg_type, attachments_json, chat_display_name
		FROM imessage_messages WHERE synced_at > ? ORDER BY date_utc DESC`),
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages modified since: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

func SearchMessages(db *store.DB, query string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.Query(
		db.Rebind(`SELECT guid, apple_rowid, text, chat_guid, sender_id, sender_service,
			is_from_me, is_read, date_utc, date_read_utc, reply_to_guid,
			associated_msg_guid, associated_msg_type, attachments_json, chat_display_name
		FROM imessage_messages
		WHERE LOWER(text) LIKE ?
		ORDER BY date_utc DESC LIMIT ?`),
		pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()

	return scanMessages(rows)
}

func ListChats(db *store.DB, opts ListOptions) ([]Chat, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(
		db.Rebind(`SELECT guid, display_name, service_name, participants_json, is_group, last_message_date
		FROM imessage_chats ORDER BY last_message_date DESC LIMIT ? OFFSET ?`),
		limit, opts.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	chats := []Chat{}
	for rows.Next() {
		var c Chat
		var participantsJSON sql.NullString
		var isGroup int
		if err := rows.Scan(&c.GUID, &c.DisplayName, &c.ServiceName,
			&participantsJSON, &isGroup, &c.LastMessageDate); err != nil {
			return nil, fmt.Errorf("scan chat: %w", err)
		}
		c.IsGroup = isGroup != 0
		if participantsJSON.Valid {
			json.Unmarshal([]byte(participantsJSON.String), &c.Participants)
		}
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func CountMessages(db *store.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM imessage_messages").Scan(&count)
	return count, err
}

func LastSyncTime(db *store.DB) (*time.Time, error) {
	var raw sql.NullString
	err := db.QueryRow("SELECT MAX(synced_at) FROM imessage_messages").Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	for _, f := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	} {
		if t, err := time.Parse(f, raw.String); err == nil {
			return &t, nil
		}
	}
	return nil, nil
}

func MaxAppleROWID(db *store.DB) (int64, error) {
	var raw sql.NullInt64
	err := db.QueryRow("SELECT MAX(apple_rowid) FROM imessage_messages").Scan(&raw)
	if err != nil {
		return 0, err
	}
	if !raw.Valid {
		return 0, nil
	}
	return raw.Int64, nil
}

func scanMessages(rows *sql.Rows) ([]Message, error) {
	msgs := []Message{}
	for rows.Next() {
		var m Message
		var isFromMe, isRead int
		if err := rows.Scan(&m.GUID, &m.AppleROWID, &m.Text, &m.ChatGUID, &m.SenderID,
			&m.SenderService, &isFromMe, &isRead, &m.Date, &m.DateRead, &m.ReplyToGUID,
			&m.AssociatedMessageGUID, &m.AssociatedMessageType, &m.AttachmentsJSON,
			&m.ChatDisplayName); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.IsFromMe = isFromMe != 0
		m.IsRead = isRead != 0
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
