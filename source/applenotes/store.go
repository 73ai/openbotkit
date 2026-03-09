package applenotes

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func SaveNote(db *store.DB, note *Note) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO applenotes_notes
			(apple_id, title, body, folder, folder_id, account, password_protected, created_at, modified_at, synced_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (apple_id) DO UPDATE SET
				title = ?, body = ?, folder = ?, folder_id = ?, account = ?,
				password_protected = ?, modified_at = ?, synced_at = CURRENT_TIMESTAMP`),
		note.AppleID, note.Title, note.Body, note.Folder, note.FolderID, note.Account, note.PasswordProtected, note.CreatedAt, note.ModifiedAt,
		note.Title, note.Body, note.Folder, note.FolderID, note.Account, note.PasswordProtected, note.ModifiedAt,
	)
	if err != nil {
		return fmt.Errorf("save note: %w", err)
	}
	return nil
}

func SaveFolder(db *store.DB, folder *Folder) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO applenotes_folders (apple_id, name, parent_apple_id, account)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (apple_id) DO UPDATE SET
				name = ?, parent_apple_id = ?, account = ?`),
		folder.AppleID, folder.Name, folder.ParentAppleID, folder.Account,
		folder.Name, folder.ParentAppleID, folder.Account,
	)
	if err != nil {
		return fmt.Errorf("save folder: %w", err)
	}
	return nil
}

func GetNote(db *store.DB, appleID string) (*Note, error) {
	var n Note
	err := db.QueryRow(
		db.Rebind(`SELECT apple_id, title, body, folder, folder_id, account,
			password_protected, created_at, modified_at
			FROM applenotes_notes WHERE apple_id = ?`),
		appleID,
	).Scan(&n.AppleID, &n.Title, &n.Body, &n.Folder, &n.FolderID, &n.Account,
		&n.PasswordProtected, &n.CreatedAt, &n.ModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("get note: %w", err)
	}
	return &n, nil
}

func ListNotes(db *store.DB, opts ListOptions) ([]Note, error) {
	var conditions []string
	var args []any

	if opts.Folder != "" {
		conditions = append(conditions, "LOWER(folder) = ?")
		args = append(args, strings.ToLower(opts.Folder))
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
		`SELECT apple_id, title, body, folder, folder_id, account,
			password_protected, created_at, modified_at
		FROM applenotes_notes %s ORDER BY modified_at DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, opts.Offset)

	rows, err := db.Query(db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		err := rows.Scan(&n.AppleID, &n.Title, &n.Body, &n.Folder, &n.FolderID, &n.Account,
			&n.PasswordProtected, &n.CreatedAt, &n.ModifiedAt)
		if err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func ListNotesModifiedSince(db *store.DB, since time.Time) ([]Note, error) {
	rows, err := db.Query(
		db.Rebind(`SELECT apple_id, title, body, folder, folder_id, account,
			password_protected, created_at, modified_at
		FROM applenotes_notes WHERE synced_at > ? ORDER BY modified_at DESC`),
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("list notes modified since: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.AppleID, &n.Title, &n.Body, &n.Folder, &n.FolderID, &n.Account,
			&n.PasswordProtected, &n.CreatedAt, &n.ModifiedAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func SearchNotes(db *store.DB, query string, limit int) ([]Note, error) {
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.Query(
		db.Rebind(`SELECT apple_id, title, body, folder, folder_id, account,
			password_protected, created_at, modified_at
		FROM applenotes_notes
		WHERE LOWER(title) LIKE ? OR LOWER(body) LIKE ?
		ORDER BY modified_at DESC LIMIT ?`),
		pattern, pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.AppleID, &n.Title, &n.Body, &n.Folder, &n.FolderID, &n.Account,
			&n.PasswordProtected, &n.CreatedAt, &n.ModifiedAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func CountNotes(db *store.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM applenotes_notes").Scan(&count)
	return count, err
}

func LastSyncTime(db *store.DB) (*time.Time, error) {
	var raw sql.NullString
	err := db.QueryRow("SELECT MAX(synced_at) FROM applenotes_notes").Scan(&raw)
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
