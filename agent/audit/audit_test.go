package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	l := OpenDefault(path)
	if l == nil {
		t.Fatal("OpenDefault returned nil")
	}
	defer l.Close()

	l.Log(Entry{
		Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Context:        "cli",
		ToolName:       "bash",
		InputSummary:   "echo hello",
		OutputSummary:  "hello",
		ApprovalStatus: "n/a",
	})

	entries := readJSONL(t, path)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].ToolName != "bash" {
		t.Errorf("tool_name = %q, want %q", entries[0].ToolName, "bash")
	}
	if entries[0].Context != "cli" {
		t.Errorf("context = %q, want %q", entries[0].Context, "cli")
	}
	if entries[0].Timestamp != "2026-01-01T00:00:00Z" {
		t.Errorf("timestamp = %q, want %q", entries[0].Timestamp, "2026-01-01T00:00:00Z")
	}
	if entries[0].Error != "" {
		t.Errorf("error should be omitted, got %q", entries[0].Error)
	}
}

func TestTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	l := OpenDefault(path)
	if l == nil {
		t.Fatal("OpenDefault returned nil")
	}
	defer l.Close()

	longInput := make([]byte, 500)
	for i := range longInput {
		longInput[i] = 'x'
	}
	l.Log(Entry{
		Context:      "cli",
		ToolName:     "bash",
		InputSummary: string(longInput),
	})

	entries := readJSONL(t, path)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if len(entries[0].InputSummary) > maxSummaryLen+10 {
		t.Errorf("input_summary len = %d, expected truncated to ~%d", len(entries[0].InputSummary), maxSummaryLen)
	}
}

func TestNilSafe(t *testing.T) {
	var l *Logger
	l.Log(Entry{ToolName: "bash"})
	if err := l.Close(); err != nil {
		t.Errorf("nil Close: %v", err)
	}
}

func TestClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	l := OpenDefault(path)
	if l == nil {
		t.Fatal("OpenDefault returned nil")
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Log after close should not panic.
	l.Log(Entry{ToolName: "bash", Context: "test"})
}

func TestOpenDefault_CreatesDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "dir", "audit.jsonl")
	l := OpenDefault(path)
	if l == nil {
		t.Fatal("OpenDefault returned nil")
	}
	defer l.Close()

	l.Log(Entry{Context: "test", ToolName: "bash", InputSummary: "echo hi"})
	entries := readJSONL(t, path)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
}

func TestOpenDefault_BadPath(t *testing.T) {
	l := OpenDefault("/bad\x00path/audit.jsonl")
	if l != nil {
		l.Close()
		t.Error("expected nil for bad path")
	}
}

func TestMultipleEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	l := OpenDefault(path)
	if l == nil {
		t.Fatal("OpenDefault returned nil")
	}
	defer l.Close()

	for i := 0; i < 5; i++ {
		l.Log(Entry{Context: "cli", ToolName: "bash"})
	}

	entries := readJSONL(t, path)
	if len(entries) != 5 {
		t.Fatalf("got %d entries, want 5", len(entries))
	}
}

func readJSONL(t *testing.T, path string) []jsonEntry {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var entries []jsonEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e jsonEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("parse JSON line: %v", err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return entries
}
