package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/priyanshujain/reimbursement/parser"
)

// StatementStore wraps a SQLite database for financial statement persistence.
type StatementStore struct {
	db *sql.DB
}

// StatementTxn represents a transaction row from the statement_transactions table.
type StatementTxn struct {
	ID          int64
	StatementID int64
	Date        time.Time
	Description string
	Amount      float64
	Currency    string
	TxnType     string
	RawLine     string
	// Joined from statements table:
	Provider    string
	AccountType string
	FilePath    string
}

// NewStatementStore opens (or creates) the statements database and runs migrations.
func NewStatementStore(dbPath string) (*StatementStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open statement db: %w", err)
	}

	s := &StatementStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate statement db: %w", err)
	}
	return s, nil
}

// Close closes the database connection.
func (s *StatementStore) Close() error {
	return s.db.Close()
}

func (s *StatementStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS statements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		account_type TEXT NOT NULL,
		period_from DATE,
		period_to DATE,
		file_path TEXT NOT NULL,
		parsed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(file_path)
	);

	CREATE TABLE IF NOT EXISTS statement_transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		statement_id INTEGER REFERENCES statements(id),
		date DATE NOT NULL,
		description TEXT NOT NULL,
		amount REAL NOT NULL,
		currency TEXT DEFAULT 'INR',
		txn_type TEXT NOT NULL,
		raw_line TEXT,
		UNIQUE(statement_id, date, description, amount)
	);

	CREATE TABLE IF NOT EXISTS invoices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		invoice_number TEXT NOT NULL,
		date DATE NOT NULL,
		amount REAL NOT NULL,
		currency TEXT NOT NULL,
		period_from DATE,
		period_to DATE,
		payment_method TEXT,
		file_path TEXT NOT NULL,
		parsed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(invoice_number)
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// StatementExists checks if a statement file has already been parsed.
func (s *StatementStore) StatementExists(filePath string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM statements WHERE file_path = ?", filePath).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check statement exists: %w", err)
	}
	return count > 0, nil
}

// SaveStatement inserts a parsed statement and all its transactions.
func (s *StatementStore) SaveStatement(stmt *parser.Statement) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT OR IGNORE INTO statements (provider, account_type, period_from, period_to, file_path)
		 VALUES (?, ?, ?, ?, ?)`,
		stmt.Provider, stmt.AccountType, stmt.PeriodFrom, stmt.PeriodTo, stmt.FilePath,
	)
	if err != nil {
		return fmt.Errorf("insert statement: %w", err)
	}

	stmtID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get statement id: %w", err)
	}
	if stmtID == 0 {
		// Already existed
		err = s.db.QueryRow("SELECT id FROM statements WHERE file_path = ?", stmt.FilePath).Scan(&stmtID)
		if err != nil {
			return fmt.Errorf("lookup existing statement: %w", err)
		}
	}

	for _, txn := range stmt.Transactions {
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO statement_transactions
			 (statement_id, date, description, amount, currency, txn_type, raw_line)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			stmtID, txn.Date, txn.Description, txn.Amount, txn.Currency, txn.Type, txn.RawLine,
		)
		if err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	return tx.Commit()
}

// SearchTransactions finds transactions whose description matches a LIKE pattern (case-insensitive).
func (s *StatementStore) SearchTransactions(pattern string) ([]StatementTxn, error) {
	rows, err := s.db.Query(
		`SELECT st.id, st.statement_id, st.date, st.description, st.amount, st.currency, st.txn_type, st.raw_line,
		        s.provider, s.account_type, s.file_path
		 FROM statement_transactions st
		 JOIN statements s ON s.id = st.statement_id
		 WHERE LOWER(st.description) LIKE LOWER(?)
		 ORDER BY st.date`,
		pattern,
	)
	if err != nil {
		return nil, fmt.Errorf("search transactions: %w", err)
	}
	defer rows.Close()

	return scanStatementTxns(rows)
}

// GetAllTransactions retrieves all statement transactions ordered by date.
func (s *StatementStore) GetAllTransactions() ([]StatementTxn, error) {
	rows, err := s.db.Query(
		`SELECT st.id, st.statement_id, st.date, st.description, st.amount, st.currency, st.txn_type, st.raw_line,
		        s.provider, s.account_type, s.file_path
		 FROM statement_transactions st
		 JOIN statements s ON s.id = st.statement_id
		 ORDER BY st.date`,
	)
	if err != nil {
		return nil, fmt.Errorf("query all transactions: %w", err)
	}
	defer rows.Close()

	return scanStatementTxns(rows)
}

// InvoiceRow represents a parsed dashboard invoice from the invoices table.
type InvoiceRow struct {
	ID            int64
	Provider      string
	InvoiceNumber string
	Date          time.Time
	Amount        float64
	Currency      string
	PeriodFrom    time.Time
	PeriodTo      time.Time
	PaymentMethod string
	FilePath      string
}

// InvoiceExists checks if an invoice has already been parsed.
func (s *StatementStore) InvoiceExists(invoiceNumber string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM invoices WHERE invoice_number = ?", invoiceNumber).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check invoice exists: %w", err)
	}
	return count > 0, nil
}

// SaveInvoice inserts a parsed dashboard invoice.
func (s *StatementStore) SaveInvoice(inv *parser.Invoice) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO invoices
		 (provider, invoice_number, date, amount, currency, period_from, period_to, payment_method, file_path)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.Provider, inv.InvoiceNumber, inv.Date, inv.Amount, inv.Currency,
		inv.PeriodFrom, inv.PeriodTo, inv.PaymentMethod, inv.FilePath,
	)
	if err != nil {
		return fmt.Errorf("insert invoice: %w", err)
	}
	return nil
}

// SearchInvoices finds invoices by provider name.
func (s *StatementStore) SearchInvoices(provider string) ([]InvoiceRow, error) {
	rows, err := s.db.Query(
		`SELECT id, provider, invoice_number, date, amount, currency,
		        period_from, period_to, payment_method, file_path
		 FROM invoices
		 WHERE LOWER(provider) = LOWER(?)
		 ORDER BY date`,
		provider,
	)
	if err != nil {
		return nil, fmt.Errorf("search invoices: %w", err)
	}
	defer rows.Close()

	var results []InvoiceRow
	for rows.Next() {
		var r InvoiceRow
		err := rows.Scan(&r.ID, &r.Provider, &r.InvoiceNumber, &r.Date, &r.Amount,
			&r.Currency, &r.PeriodFrom, &r.PeriodTo, &r.PaymentMethod, &r.FilePath)
		if err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func scanStatementTxns(rows *sql.Rows) ([]StatementTxn, error) {
	var txns []StatementTxn
	for rows.Next() {
		var t StatementTxn
		err := rows.Scan(
			&t.ID, &t.StatementID, &t.Date, &t.Description, &t.Amount,
			&t.Currency, &t.TxnType, &t.RawLine,
			&t.Provider, &t.AccountType, &t.FilePath,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, t)
	}
	return txns, rows.Err()
}
