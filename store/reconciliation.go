package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ReconStore wraps a SQLite database for reconciliation result persistence.
type ReconStore struct {
	db *sql.DB
}

// ReconRow represents a row in the reconciliation table.
type ReconRow struct {
	ID             int64
	Service        string
	Status         string
	Date           time.Time
	SourceAmount   string
	DestAmount     string
	Description    string
	Account        string
	Source         string
	Destination    string
	AttachmentPath string
	DestRef        string
	Notes          string
	AmountINR      string
	UpdatedAt      time.Time
}

// NewReconStore opens (or creates) the reconciliation database and runs migrations.
func NewReconStore(dbPath string) (*ReconStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open recon db: %w", err)
	}

	s := &ReconStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate recon db: %w", err)
	}
	return s, nil
}

// Close closes the database connection.
func (s *ReconStore) Close() error {
	return s.db.Close()
}

func (s *ReconStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS reconciliation (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service TEXT NOT NULL,
		status TEXT NOT NULL,
		date DATE NOT NULL,
		source_amount TEXT,
		dest_amount TEXT,
		description TEXT NOT NULL,
		account TEXT,
		source TEXT,
		destination TEXT,
		attachment_path TEXT,
		dest_ref TEXT,
		notes TEXT,
		amount_inr TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(service, date, description)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Clear deletes all rows from the reconciliation table.
func (s *ReconStore) Clear() error {
	_, err := s.db.Exec("DELETE FROM reconciliation")
	return err
}

// Upsert inserts or updates a reconciliation row.
// On conflict (same service + date + description), updates the mutable fields.
func (s *ReconStore) Upsert(r *ReconRow) error {
	_, err := s.db.Exec(`
		INSERT INTO reconciliation
			(service, status, date, source_amount, dest_amount, description,
			 account, source, destination, attachment_path, dest_ref, notes, amount_inr, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(service, date, description) DO UPDATE SET
			status = excluded.status,
			source_amount = excluded.source_amount,
			dest_amount = excluded.dest_amount,
			account = excluded.account,
			source = excluded.source,
			destination = excluded.destination,
			attachment_path = excluded.attachment_path,
			dest_ref = excluded.dest_ref,
			notes = excluded.notes,
			amount_inr = excluded.amount_inr,
			updated_at = CURRENT_TIMESTAMP
	`, r.Service, r.Status, r.Date, r.SourceAmount, r.DestAmount, r.Description,
		r.Account, r.Source, r.Destination, r.AttachmentPath, r.DestRef, r.Notes, r.AmountINR,
	)
	if err != nil {
		return fmt.Errorf("upsert reconciliation: %w", err)
	}
	return nil
}

// GetByStatus returns all reconciliation rows with the given status, grouped by service then sorted by date.
func (s *ReconStore) GetByStatus(status string) ([]ReconRow, error) {
	return s.query("SELECT "+reconColumns+" FROM reconciliation WHERE status = ? ORDER BY service, date", status)
}

// GetAll returns all reconciliation rows grouped by service then sorted by date.
func (s *ReconStore) GetAll() ([]ReconRow, error) {
	return s.query("SELECT " + reconColumns + " FROM reconciliation ORDER BY service, date")
}

// CountByStatus returns a map of status → count.
func (s *ReconStore) CountByStatus() (map[string]int, error) {
	rows, err := s.db.Query("SELECT status, COUNT(*) FROM reconciliation GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// SumByService returns a map of service → total reconciled INR amount (as float64).
func (s *ReconStore) SumByService() (map[string]float64, error) {
	rows, err := s.db.Query(`
		SELECT service, SUM(CAST(REPLACE(dest_amount, ',', '') AS REAL))
		FROM reconciliation
		WHERE status = 'RECONCILED' AND dest_amount != 'MANUAL' AND dest_amount != ''
		GROUP BY service
	`)
	if err != nil {
		return nil, fmt.Errorf("sum by service: %w", err)
	}
	defer rows.Close()

	sums := make(map[string]float64)
	for rows.Next() {
		var service string
		var total float64
		if err := rows.Scan(&service, &total); err != nil {
			return nil, err
		}
		sums[service] = total
	}
	return sums, rows.Err()
}

const reconColumns = `id, service, status, date, source_amount, dest_amount, description,
	account, source, destination, attachment_path, dest_ref, notes, amount_inr, updated_at`

func (s *ReconStore) query(q string, args ...interface{}) ([]ReconRow, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query reconciliation: %w", err)
	}
	defer rows.Close()

	var results []ReconRow
	for rows.Next() {
		var r ReconRow
		err := rows.Scan(
			&r.ID, &r.Service, &r.Status, &r.Date, &r.SourceAmount, &r.DestAmount,
			&r.Description, &r.Account, &r.Source, &r.Destination,
			&r.AttachmentPath, &r.DestRef, &r.Notes, &r.AmountINR, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan reconciliation: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
