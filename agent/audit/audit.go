package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
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

type jsonEntry struct {
	Timestamp      string `json:"timestamp"`
	Context        string `json:"context"`
	ToolName       string `json:"tool_name"`
	InputSummary   string `json:"input_summary"`
	OutputSummary  string `json:"output_summary"`
	ApprovalStatus string `json:"approval_status"`
	Error          string `json:"error,omitempty"`
}

// Logger writes audit entries to a JSONL file.
type Logger struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

// OpenDefault opens (or creates) the audit JSONL file at path,
// creating parent directories as needed.
// Returns nil if any step fails (errors are logged via slog).
func OpenDefault(path string) *Logger {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		slog.Debug("audit: cannot create dir", "error", err)
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		slog.Debug("audit: open file failed", "error", err)
		return nil
	}
	return &Logger{file: f, enc: json.NewEncoder(f)}
}

// Close closes the underlying file.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	err := l.file.Close()
	l.file = nil
	l.enc = nil
	return err
}

const maxSummaryLen = 200

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Log writes an audit entry as a JSON line. It never returns an error
// to the caller; failures are logged via slog.
func (l *Logger) Log(e Entry) {
	if l == nil || l.file == nil {
		return
	}
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	if e.ApprovalStatus == "" {
		e.ApprovalStatus = "n/a"
	}

	je := jsonEntry{
		Timestamp:      ts.Format(time.RFC3339),
		Context:        e.Context,
		ToolName:       e.ToolName,
		InputSummary:   truncate(e.InputSummary, maxSummaryLen),
		OutputSummary:  truncate(e.OutputSummary, maxSummaryLen),
		ApprovalStatus: e.ApprovalStatus,
		Error:          e.Error,
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.enc == nil {
		return
	}
	if err := l.enc.Encode(je); err != nil {
		slog.Error("audit log write failed", "tool", e.ToolName, "error", err)
	}
}
