package audit

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/73ai/openbotkit/store"
)

// Entry represents a single audit log record.
type Entry struct {
	Timestamp      time.Time
	Context        string // "cli", "telegram", "scheduled", "delegated"
	ToolName       string
	InputSummary   string
	OutputSummary  string
	ApprovalStatus string // "approved", "denied", "auto", "n/a"
	Error          string
}

// Logger writes audit entries to a database.
type Logger struct {
	db *store.DB
}

// NewLogger creates an audit logger backed by the given database.
func NewLogger(db *store.DB) *Logger {
	return &Logger{db: db}
}

// OpenDefault opens (or creates) the audit database at dbPath,
// runs migrations, and returns a ready Logger.
// Returns nil if any step fails (errors are logged via slog).
func OpenDefault(dbPath string) *Logger {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		slog.Debug("audit: cannot create dir", "error", err)
		return nil
	}
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		slog.Debug("audit: open db failed", "error", err)
		return nil
	}
	if err := Migrate(db); err != nil {
		db.Close()
		slog.Debug("audit: migrate failed", "error", err)
		return nil
	}
	return NewLogger(db)
}

// Close closes the underlying database connection.
func (l *Logger) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}

const maxSummaryLen = 200

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Log writes an audit entry. It never returns an error to the caller;
// failures are logged via slog.
func (l *Logger) Log(e Entry) {
	if l == nil || l.db == nil {
		return
	}
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	inputSum := truncate(e.InputSummary, maxSummaryLen)
	outputSum := truncate(e.OutputSummary, maxSummaryLen)
	if e.ApprovalStatus == "" {
		e.ApprovalStatus = "n/a"
	}

	query := l.db.Rebind(`INSERT INTO audit_log (timestamp, context, tool_name, input_summary, output_summary, approval_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	_, err := l.db.Exec(query,
		ts.Format(time.RFC3339),
		e.Context,
		e.ToolName,
		inputSum,
		outputSum,
		e.ApprovalStatus,
		e.Error,
	)
	if err != nil {
		slog.Error("audit log write failed", "tool", e.ToolName, "error", err)
	}
}

// Migrate creates the audit_log table if it doesn't exist.
func Migrate(db *store.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL DEFAULT (datetime('now')),
			context TEXT NOT NULL,
			tool_name TEXT NOT NULL,
			input_summary TEXT,
			output_summary TEXT,
			approval_status TEXT DEFAULT 'n/a',
			error TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
		CREATE INDEX IF NOT EXISTS idx_audit_tool ON audit_log(tool_name);
	`)
	return err
}
