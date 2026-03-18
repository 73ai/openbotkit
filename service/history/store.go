package history

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/73ai/openbotkit/store"
)

func UpsertConversation(db *store.DB, sessionID, cwd string) (int64, error) {
	var id int64
	err := db.QueryRow(
		db.Rebind("SELECT id FROM history_conversations WHERE session_id = ?"),
		sessionID,
	).Scan(&id)
	if err == nil {
		_, err = db.Exec(
			db.Rebind("UPDATE history_conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?"),
			id,
		)
		return id, err
	}

	res, err := db.Exec(
		db.Rebind("INSERT INTO history_conversations (session_id, cwd, started_at, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"),
		sessionID, cwd,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate key") {
			err2 := db.QueryRow(
				db.Rebind("SELECT id FROM history_conversations WHERE session_id = ?"),
				sessionID,
			).Scan(&id)
			if err2 != nil {
				return 0, fmt.Errorf("lookup existing conversation: %w", err2)
			}
			return id, nil
		}
		return 0, fmt.Errorf("insert conversation: %w", err)
	}
	return res.LastInsertId()
}

func SaveMessage(db *store.DB, convID int64, role, content string) error {
	_, err := db.Exec(
		db.Rebind("INSERT INTO history_messages (conversation_id, role, content) VALUES (?, ?, ?)"),
		convID, role, content,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func CountConversations(db *store.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM history_conversations").Scan(&count)
	return count, err
}

func parseTimestamp(raw string) *time.Time {
	for _, f := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	} {
		if t, err := time.Parse(f, raw); err == nil {
			return &t
		}
	}
	return nil
}

func LastCaptureTime(db *store.DB) (*time.Time, error) {
	var raw sql.NullString
	err := db.QueryRow("SELECT MAX(updated_at) FROM history_conversations").Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	return parseTimestamp(raw.String), nil
}

type RecentSession struct {
	SessionID string
	UpdatedAt time.Time
}

func EndSession(db *store.DB, sessionID string) error {
	_, err := db.Exec(
		db.Rebind("UPDATE history_conversations SET ended_at = CURRENT_TIMESTAMP WHERE session_id = ?"),
		sessionID,
	)
	return err
}

func LoadRecentSession(db *store.DB, cwd string, maxAge time.Duration) (*RecentSession, error) {
	var sessionID string
	var raw string
	err := db.QueryRow(
		db.Rebind("SELECT session_id, updated_at FROM history_conversations WHERE cwd = ? AND ended_at IS NULL ORDER BY updated_at DESC LIMIT 1"),
		cwd,
	).Scan(&sessionID, &raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load recent session: %w", err)
	}
	ts := parseTimestamp(raw)
	if ts == nil {
		return nil, nil
	}
	if time.Since(*ts) > maxAge {
		return nil, nil
	}
	return &RecentSession{SessionID: sessionID, UpdatedAt: *ts}, nil
}

func LoadSessionMessages(db *store.DB, sessionID string, limit int) ([]Message, error) {
	rows, err := db.Query(
		db.Rebind(`SELECT m.conversation_id, m.role, m.content, m.timestamp
			FROM history_messages m
			JOIN history_conversations c ON c.id = m.conversation_id
			WHERE c.session_id = ?
			ORDER BY m.timestamp ASC, m.id ASC
			LIMIT ?`),
		sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("load session messages: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var raw string
		if err := rows.Scan(&m.ConversationID, &m.Role, &m.Content, &raw); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if ts := parseTimestamp(raw); ts != nil {
			m.Timestamp = *ts
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func MessageCountForSession(db *store.DB, sessionID string) (int, error) {
	var count int
	err := db.QueryRow(
		db.Rebind(`SELECT COUNT(*) FROM history_messages m
			JOIN history_conversations c ON c.id = m.conversation_id
			WHERE c.session_id = ?`),
		sessionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count messages for session: %w", err)
	}
	return count, nil
}
