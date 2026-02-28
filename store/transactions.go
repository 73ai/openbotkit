package store

import (
	"fmt"
	"time"
)

// Transaction represents a reimbursable transaction stored in the database.
type Transaction struct {
	ID             int64
	EmailID        int64
	Service        string
	Account        string
	Date           time.Time
	Subject        string
	Amount         string
	Currency       string
	AttachmentPath string
}

// SaveTransaction inserts a transaction into the database.
func (s *Store) SaveTransaction(tx *Transaction) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO transactions (email_id, service, account, date, subject, amount, currency, attachment_path)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.EmailID, tx.Service, tx.Account, tx.Date,
		tx.Subject, tx.Amount, tx.Currency, tx.AttachmentPath,
	)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

// TransactionExists checks if a transaction for the given email and service exists.
func (s *Store) TransactionExists(emailID int64, service string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE email_id = ? AND service = ?",
		emailID, service,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check transaction exists: %w", err)
	}
	return count > 0, nil
}

// GetAllTransactions retrieves all transactions ordered by date.
func (s *Store) GetAllTransactions() ([]Transaction, error) {
	rows, err := s.db.Query(
		`SELECT id, email_id, service, account, date, subject, amount, currency, attachment_path
		 FROM transactions ORDER BY date`,
	)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var txns []Transaction
	for rows.Next() {
		var tx Transaction
		err := rows.Scan(
			&tx.ID, &tx.EmailID, &tx.Service, &tx.Account,
			&tx.Date, &tx.Subject, &tx.Amount, &tx.Currency, &tx.AttachmentPath,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, tx)
	}
	return txns, rows.Err()
}
