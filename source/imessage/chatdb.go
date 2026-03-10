package imessage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultChatDBPath = "~/Library/Messages/chat.db"

// OpenChatDB opens the Apple Messages chat.db in read-only mode.
// It is a package-level variable so tests can swap it with a mock.
var OpenChatDB = defaultOpenChatDB

func defaultOpenChatDB() (*sql.DB, error) {
	path := defaultChatDBPath
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home dir: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	db, err := sql.Open("sqlite3", path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open chat.db: %w (does your terminal have Full Disk Access?)", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("access chat.db: %w (does your terminal have Full Disk Access?)", err)
	}

	return db, nil
}

// appleNanosToTime converts Apple's CoreData timestamp (nanoseconds since
// 2001-01-01 00:00:00 UTC) to a Go time.Time.
func appleNanosToTime(ns int64) time.Time {
	if ns == 0 {
		return time.Time{}
	}
	const appleEpochOffset = 978307200 // seconds between Unix epoch and 2001-01-01
	secs := ns / 1_000_000_000
	remainder := ns % 1_000_000_000
	return time.Unix(secs+appleEpochOffset, remainder).UTC()
}

func FetchHandles(chatDB *sql.DB) ([]Handle, error) {
	rows, err := chatDB.Query(`SELECT id, service FROM handle`)
	if err != nil {
		return nil, fmt.Errorf("fetch handles: %w", err)
	}
	defer rows.Close()

	var handles []Handle
	for rows.Next() {
		var h Handle
		if err := rows.Scan(&h.ID, &h.Service); err != nil {
			return nil, fmt.Errorf("scan handle: %w", err)
		}
		handles = append(handles, h)
	}
	return handles, rows.Err()
}

func FetchChats(chatDB *sql.DB) ([]Chat, error) {
	rows, err := chatDB.Query(`
		SELECT c.guid, c.display_name, c.service_name,
			GROUP_CONCAT(h.id) as participants
		FROM chat c
		LEFT JOIN chat_handle_join chj ON chj.chat_id = c.ROWID
		LEFT JOIN handle h ON h.ROWID = chj.handle_id
		GROUP BY c.ROWID`)
	if err != nil {
		return nil, fmt.Errorf("fetch chats: %w", err)
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		var participants sql.NullString
		if err := rows.Scan(&c.GUID, &c.DisplayName, &c.ServiceName, &participants); err != nil {
			return nil, fmt.Errorf("scan chat: %w", err)
		}
		if participants.Valid && participants.String != "" {
			c.Participants = strings.Split(participants.String, ",")
		}
		c.IsGroup = len(c.Participants) > 1
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func FetchMessagesSince(chatDB *sql.DB, sinceROWID int64) ([]Message, int, error) {
	rows, err := chatDB.Query(`
		SELECT m.ROWID, m.guid, m.text, m.is_from_me, m.is_read,
			m.date, m.date_read,
			COALESCE(h.id, '') as sender_id,
			COALESCE(h.service, '') as sender_service,
			COALESCE(c.guid, '') as chat_guid,
			COALESCE(c.display_name, '') as chat_display_name,
			m.cache_has_attachments,
			COALESCE(m.thread_originator_guid, '') as reply_to_guid,
			COALESCE(m.associated_message_guid, '') as associated_msg_guid,
			m.associated_message_type
		FROM message m
		LEFT JOIN handle h ON h.ROWID = m.handle_id
		LEFT JOIN chat_message_join cmj ON cmj.message_id = m.ROWID
		LEFT JOIN chat c ON c.ROWID = cmj.chat_id
		WHERE m.ROWID > ?
		ORDER BY m.ROWID ASC`, sinceROWID)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	skipped := 0
	for rows.Next() {
		var m Message
		var isFromMe, isRead, hasAttachments int
		var dateNanos, dateReadNanos int64
		var text sql.NullString

		if err := rows.Scan(&m.AppleROWID, &m.GUID, &text, &isFromMe, &isRead,
			&dateNanos, &dateReadNanos,
			&m.SenderID, &m.SenderService, &m.ChatGUID, &m.ChatDisplayName,
			&hasAttachments, &m.ReplyToGUID, &m.AssociatedMessageGUID,
			&m.AssociatedMessageType); err != nil {
			return nil, 0, fmt.Errorf("scan message: %w", err)
		}

		if !text.Valid {
			skipped++
			continue
		}

		m.Text = text.String
		m.IsFromMe = isFromMe != 0
		m.IsRead = isRead != 0
		m.Date = appleNanosToTime(dateNanos)
		m.DateRead = appleNanosToTime(dateReadNanos)

		if hasAttachments != 0 {
			attachments, err := fetchAttachments(chatDB, m.AppleROWID)
			if err == nil && len(attachments) > 0 {
				data, _ := json.Marshal(attachments)
				m.AttachmentsJSON = string(data)
			}
		}

		messages = append(messages, m)
	}
	return messages, skipped, rows.Err()
}

func fetchAttachments(chatDB *sql.DB, messageROWID int64) ([]AttachmentMeta, error) {
	rows, err := chatDB.Query(`
		SELECT a.filename, a.mime_type, a.total_bytes
		FROM attachment a
		JOIN message_attachment_join maj ON maj.attachment_id = a.ROWID
		WHERE maj.message_id = ?`, messageROWID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []AttachmentMeta
	for rows.Next() {
		var a AttachmentMeta
		var filename, mimeType sql.NullString
		var totalBytes sql.NullInt64
		if err := rows.Scan(&filename, &mimeType, &totalBytes); err != nil {
			return nil, err
		}
		if filename.Valid {
			a.Filename = filename.String
		}
		if mimeType.Valid {
			a.MIMEType = mimeType.String
		}
		if totalBytes.Valid {
			a.TotalBytes = totalBytes.Int64
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}
