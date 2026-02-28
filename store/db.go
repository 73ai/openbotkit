package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Store wraps a SQLite database for email persistence.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database and initializes the schema.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id TEXT NOT NULL,
		account TEXT NOT NULL,
		from_addr TEXT,
		to_addr TEXT,
		subject TEXT,
		date DATETIME,
		body TEXT,
		html_body TEXT,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id, account)
	);

	CREATE TABLE IF NOT EXISTS attachments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email_id INTEGER REFERENCES emails(id),
		filename TEXT,
		mime_type TEXT,
		saved_path TEXT
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email_id INTEGER REFERENCES emails(id),
		service TEXT NOT NULL,
		account TEXT NOT NULL,
		date DATETIME,
		subject TEXT,
		amount TEXT,
		currency TEXT,
		attachment_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(email_id, service)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}
