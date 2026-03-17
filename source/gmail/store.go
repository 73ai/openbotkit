package gmail

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/73ai/openbotkit/store"
)

func EmailExists(db *store.DB, messageID, account string) (bool, error) {
	var count int
	err := db.QueryRow(
		db.Rebind("SELECT COUNT(*) FROM emails WHERE message_id = ? AND account = ?"),
		messageID, account,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return count > 0, nil
}

func SaveEmail(db *store.DB, email *Email) (int64, error) {
	res, err := db.Exec(
		db.Rebind(`INSERT INTO emails (message_id, account, from_addr, to_addr, subject, date, body, html_body)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		email.MessageID, email.Account, email.From, email.To,
		email.Subject, email.Date.UTC(), email.Body, email.HTMLBody,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate key") {
			var id int64
			err2 := db.QueryRow(
				db.Rebind("SELECT id FROM emails WHERE message_id = ? AND account = ?"),
				email.MessageID, email.Account,
			).Scan(&id)
			if err2 != nil {
				return 0, fmt.Errorf("lookup existing email: %w", err2)
			}
			return id, nil
		}
		return 0, fmt.Errorf("insert email: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get email id: %w", err)
	}

	for _, att := range email.Attachments {
		_, err := db.Exec(
			db.Rebind(`INSERT INTO attachments (email_id, filename, mime_type, saved_path) VALUES (?, ?, ?, ?)`),
			id, att.Filename, att.MimeType, att.SavedPath,
		)
		if err != nil {
			return id, fmt.Errorf("insert attachment: %w", err)
		}
	}

	return id, nil
}

func ListEmails(db *store.DB, opts ListOptions) ([]Email, error) {
	var conditions []string
	var args []any

	if opts.Account != "" {
		conditions = append(conditions, "account = ?")
		args = append(args, opts.Account)
	}
	if opts.From != "" {
		conditions = append(conditions, "LOWER(from_addr) LIKE ?")
		args = append(args, "%"+strings.ToLower(opts.From)+"%")
	}
	if opts.Subject != "" {
		conditions = append(conditions, "LOWER(subject) LIKE ?")
		args = append(args, "%"+strings.ToLower(opts.Subject)+"%")
	}
	if opts.After != "" {
		conditions = append(conditions, "date >= ?")
		args = append(args, opts.After)
	}
	if opts.Before != "" {
		conditions = append(conditions, "date <= ?")
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
		`SELECT id, message_id, account, from_addr, to_addr, subject, date, body, html_body
		 FROM emails %s ORDER BY date DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, opts.Offset)

	rows, err := db.Query(db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("list emails: %w", err)
	}
	defer rows.Close()

	emails := []Email{}
	for rows.Next() {
		var e Email
		var id int64
		err := rows.Scan(&id, &e.MessageID, &e.Account, &e.From, &e.To, &e.Subject, &e.Date, &e.Body, &e.HTMLBody)
		if err != nil {
			return nil, fmt.Errorf("scan email: %w", err)
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

func GetEmail(db *store.DB, messageID string) (*Email, error) {
	var e Email
	var id int64
	err := db.QueryRow(
		db.Rebind(`SELECT id, message_id, account, from_addr, to_addr, subject, date, body, html_body
		 FROM emails WHERE message_id = ?`),
		messageID,
	).Scan(&id, &e.MessageID, &e.Account, &e.From, &e.To, &e.Subject, &e.Date, &e.Body, &e.HTMLBody)
	if err != nil {
		return nil, fmt.Errorf("get email: %w", err)
	}

	rows, err := db.Query(
		db.Rebind(`SELECT filename, mime_type, saved_path FROM attachments WHERE email_id = ?`),
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.Filename, &a.MimeType, &a.SavedPath); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		e.Attachments = append(e.Attachments, a)
	}

	return &e, rows.Err()
}

func SearchEmails(db *store.DB, query string, limit int) ([]Email, error) {
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.Query(
		db.Rebind(`SELECT id, message_id, account, from_addr, to_addr, subject, date, body, html_body
		 FROM emails
		 WHERE LOWER(subject) LIKE ? OR LOWER(body) LIKE ?
		 ORDER BY date DESC LIMIT ?`),
		pattern, pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search emails: %w", err)
	}
	defer rows.Close()

	emails := []Email{}
	for rows.Next() {
		var e Email
		var id int64
		if err := rows.Scan(&id, &e.MessageID, &e.Account, &e.From, &e.To, &e.Subject, &e.Date, &e.Body, &e.HTMLBody); err != nil {
			return nil, fmt.Errorf("scan email: %w", err)
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

func CountEmails(db *store.DB, account string) (int64, error) {
	query := "SELECT COUNT(*) FROM emails"
	var args []any
	if account != "" {
		query += " WHERE account = ?"
		args = append(args, account)
	}

	var count int64
	err := db.QueryRow(db.Rebind(query), args...).Scan(&count)
	return count, err
}

func LastSyncTime(db *store.DB) (*time.Time, error) {
	var raw sql.NullString
	err := db.QueryRow("SELECT MAX(fetched_at) FROM emails").Scan(&raw)
	if err != nil {
		return nil, err
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	// SQLite stores timestamps as strings; try common formats.
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

type AttachmentRow struct {
	EmailMessageID string
	Filename       string
	MimeType       string
	SavedPath      string
}

func GetSyncState(db *store.DB, account string) (*SyncState, error) {
	var s SyncState
	err := db.QueryRow(
		db.Rebind("SELECT account, history_id, updated_at FROM sync_state WHERE account = ?"),
		account,
	).Scan(&s.Account, &s.HistoryID, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sync state: %w", err)
	}
	return &s, nil
}

func SaveSyncState(db *store.DB, account string, historyID uint64) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO sync_state (account, history_id, updated_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT (account) DO UPDATE SET history_id = ?, updated_at = CURRENT_TIMESTAMP`),
		account, historyID, historyID,
	)
	if err != nil {
		return fmt.Errorf("save sync state: %w", err)
	}
	return nil
}

func ListAttachments(db *store.DB, emailMessageID string) ([]AttachmentRow, error) {
	query := `SELECT e.message_id, a.filename, a.mime_type, a.saved_path
		FROM attachments a
		JOIN emails e ON e.id = a.email_id`
	var args []any
	if emailMessageID != "" {
		query += " WHERE e.message_id = ?"
		args = append(args, emailMessageID)
	}
	query += " ORDER BY e.date DESC, a.filename"

	rows, err := db.Query(db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	results := []AttachmentRow{}
	for rows.Next() {
		var r AttachmentRow
		if err := rows.Scan(&r.EmailMessageID, &r.Filename, &r.MimeType, &r.SavedPath); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
