package audit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit_test.db")
	db, err := store.Open(store.SQLiteConfig(path))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	// Idempotent.
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate (2nd call): %v", err)
	}
}

func TestLogger_Log(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	l := NewLogger(db)
	l.Log(Entry{
		Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Context:        "cli",
		ToolName:       "bash",
		InputSummary:   "echo hello",
		OutputSummary:  "hello",
		ApprovalStatus: "n/a",
	})

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	var toolName, ctx string
	err := db.QueryRow("SELECT tool_name, context FROM audit_log WHERE id=1").Scan(&toolName, &ctx)
	if err != nil {
		t.Fatalf("query row: %v", err)
	}
	if toolName != "bash" {
		t.Errorf("tool_name = %q, want %q", toolName, "bash")
	}
	if ctx != "cli" {
		t.Errorf("context = %q, want %q", ctx, "cli")
	}
}

func TestLogger_Truncation(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	l := NewLogger(db)
	longInput := make([]byte, 500)
	for i := range longInput {
		longInput[i] = 'x'
	}
	l.Log(Entry{
		Context:      "cli",
		ToolName:     "bash",
		InputSummary: string(longInput),
	})

	var inputSum string
	err := db.QueryRow("SELECT input_summary FROM audit_log WHERE id=1").Scan(&inputSum)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(inputSum) > maxSummaryLen+10 {
		t.Errorf("input_summary len = %d, expected truncated to ~%d", len(inputSum), maxSummaryLen)
	}
}

func TestLogger_NilSafe(t *testing.T) {
	var l *Logger
	// Should not panic.
	l.Log(Entry{ToolName: "bash"})
	if err := l.Close(); err != nil {
		t.Errorf("nil Close: %v", err)
	}
}

func TestOpenDefault(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit", "data.db")
	l := OpenDefault(dbPath)
	if l == nil {
		t.Fatal("OpenDefault returned nil")
	}
	defer l.Close()
	l.Log(Entry{Context: "test", ToolName: "bash", InputSummary: "echo hi"})

	var count int
	if err := l.db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestOpenDefault_BadPath(t *testing.T) {
	// A null byte in the path is invalid on all platforms.
	l := OpenDefault("/bad\x00path/data.db")
	if l != nil {
		l.Close()
		t.Error("expected nil for bad path")
	}
}

func TestLogger_Close(t *testing.T) {
	path := filepath.Join(t.TempDir(), "close_test.db")
	db, err := store.Open(store.SQLiteConfig(path))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	l := NewLogger(db)
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Log after close should not panic (fire-and-forget).
	l.Log(Entry{ToolName: "bash", Context: "test"})
}
