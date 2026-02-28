package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/priyanshujain/reimbursement/search"
)

// EmailResult holds an email row with its first attachment path for recon use.
type EmailResult struct {
	ID             int64
	Account        string
	From           string
	Subject        string
	Date           time.Time
	Body           string
	HTMLBody       string
	AttachmentPath string
}

// EmailExists checks if an email with the given message ID and account already exists.
func (s *Store) EmailExists(messageID, account string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM emails WHERE message_id = ? AND account = ?",
		messageID, account,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return count > 0, nil
}

// SaveEmail inserts an email and its attachments into the database.
// Returns the email's database ID.
func (s *Store) SaveEmail(email *search.Email) (int64, error) {
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO emails (message_id, account, from_addr, to_addr, subject, date, body, html_body)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		email.MessageID, email.Account, email.From, email.To,
		email.Subject, email.Date, email.Body, email.HTMLBody,
	)
	if err != nil {
		return 0, fmt.Errorf("insert email: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get email id: %w", err)
	}

	// If the row was ignored (already exists), look up the existing ID
	if id == 0 {
		err = s.db.QueryRow(
			"SELECT id FROM emails WHERE message_id = ? AND account = ?",
			email.MessageID, email.Account,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("lookup existing email: %w", err)
		}
	}

	// Save attachment metadata
	for _, att := range email.Attachments {
		_, err := s.db.Exec(
			`INSERT INTO attachments (email_id, filename, mime_type, saved_path) VALUES (?, ?, ?, ?)`,
			id, att.Filename, att.MimeType, att.SavedPath,
		)
		if err != nil {
			return id, fmt.Errorf("insert attachment: %w", err)
		}
	}

	return id, nil
}

// SearchEmails finds emails matching from-address substrings after a given date.
// fromPatterns: at least one must match (OR). Only filters by from + date.
// All subject-based filtering is the responsibility of the recon/matching layer.
func (s *Store) SearchEmails(fromPatterns []string, after time.Time) ([]EmailResult, error) {
	var conditions []string
	var args []interface{}

	// From filter: any of the patterns
	if len(fromPatterns) > 0 {
		var fromConds []string
		for _, p := range fromPatterns {
			fromConds = append(fromConds, "LOWER(e.from_addr) LIKE ?")
			args = append(args, "%"+strings.ToLower(p)+"%")
		}
		conditions = append(conditions, "("+strings.Join(fromConds, " OR ")+")")
	}

	// Date filter: exclude the AfterDate entirely (it was already reimbursed)
	afterNextDay := after.AddDate(0, 0, 1)
	conditions = append(conditions, "e.date >= ?")
	args = append(args, afterNextDay)

	where := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT e.id, e.account, e.from_addr, e.subject, e.date, e.body, e.html_body,
		       COALESCE((SELECT a.saved_path FROM attachments a WHERE a.email_id = e.id LIMIT 1), '')
		FROM emails e
		WHERE %s
		ORDER BY e.date`, where)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search emails: %w", err)
	}
	defer rows.Close()

	var results []EmailResult
	for rows.Next() {
		var r EmailResult
		err := rows.Scan(&r.ID, &r.Account, &r.From, &r.Subject, &r.Date, &r.Body, &r.HTMLBody, &r.AttachmentPath)
		if err != nil {
			return nil, fmt.Errorf("scan email: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
