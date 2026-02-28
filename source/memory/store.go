package memory

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func UpsertConversation(db *store.DB, sessionID, cwd string) (int64, error) {
	// Try to find existing conversation first.
	var id int64
	err := db.QueryRow(
		db.Rebind("SELECT id FROM memory_conversations WHERE session_id = ?"),
		sessionID,
	).Scan(&id)
	if err == nil {
		_, err = db.Exec(
			db.Rebind("UPDATE memory_conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?"),
			id,
		)
		return id, err
	}

	res, err := db.Exec(
		db.Rebind("INSERT INTO memory_conversations (session_id, cwd, started_at, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"),
		sessionID, cwd,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate key") {
			err2 := db.QueryRow(
				db.Rebind("SELECT id FROM memory_conversations WHERE session_id = ?"),
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
		db.Rebind("INSERT INTO memory_messages (conversation_id, role, content) VALUES (?, ?, ?)"),
		convID, role, content,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

func CountConversations(db *store.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM memory_conversations").Scan(&count)
	return count, err
}

func LastCaptureTime(db *store.DB) (*time.Time, error) {
	var raw sql.NullString
	err := db.QueryRow("SELECT MAX(updated_at) FROM memory_conversations").Scan(&raw)
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

func MessageCountForSession(db *store.DB, sessionID string) (int, error) {
	var count int
	err := db.QueryRow(
		db.Rebind(`SELECT COUNT(*) FROM memory_messages m
			JOIN memory_conversations c ON c.id = m.conversation_id
			WHERE c.session_id = ?`),
		sessionID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count messages for session: %w", err)
	}
	return count, nil
}
