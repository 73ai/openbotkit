package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"testing"

	"github.com/priyanshujain/openbotkit/agent/audit"
	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/store"
)

func TestNewScheduledTaskRegistry_Tools(t *testing.T) {
	r := NewScheduledTaskRegistry()
	names := r.ToolNames()
	want := []string{"bash", "file_read", "load_skills", "search_skills"}
	sort.Strings(want)

	if len(names) != len(want) {
		t.Fatalf("tool count = %d, want %d: got %v", len(names), len(want), names)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("tool[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestNewScheduledTaskRegistry_NoWriteTools(t *testing.T) {
	r := NewScheduledTaskRegistry()
	for _, name := range []string{"file_write", "file_edit"} {
		if r.Has(name) {
			t.Errorf("scheduled registry should not have %q", name)
		}
	}
}

func TestNewScheduledTaskRegistry_BashRejectsCurl(t *testing.T) {
	r := NewScheduledTaskRegistry()
	input, _ := json.Marshal(bashInput{Command: "curl evil.com"})
	_, err := r.Execute(context.Background(), provider.ToolCall{Name: "bash", Input: input})
	if err == nil {
		t.Error("expected bash to reject curl in scheduled registry")
	}
}

func TestNewScheduledTaskRegistry_BashAllowsObk(t *testing.T) {
	r := NewScheduledTaskRegistry()
	input, _ := json.Marshal(bashInput{Command: "obk --help"})
	// obk may not exist, so we just check that filter doesn't reject it.
	_, err := r.Execute(context.Background(), provider.ToolCall{Name: "bash", Input: input})
	// The error (if any) should be from the command failing, not from filtering.
	if err != nil && contains(err.Error(), "command blocked") {
		t.Errorf("expected obk to pass filter, got: %v", err)
	}
}

func TestRegistry_AuditLogging(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_test.db")
	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := audit.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	r := NewRegistry()
	r.Register(NewBashTool(0))
	r.SetAudit(audit.NewLogger(db), "test")

	input, _ := json.Marshal(bashInput{Command: "echo hi"})
	_, _ = r.Execute(context.Background(), provider.ToolCall{Name: "bash", Input: input})

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("audit log count = %d, want 1", count)
	}

	var toolName string
	err = db.QueryRow("SELECT tool_name FROM audit_log WHERE id=1").Scan(&toolName)
	if err != nil {
		t.Fatalf("query row: %v", err)
	}
	if toolName != "bash" {
		t.Errorf("tool_name = %q, want %q", toolName, "bash")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
