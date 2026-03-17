package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestDBCmd_InvalidSource(t *testing.T) {
	cfg := config.Default()
	err := dbLocal(cfg, "unknown", "SELECT 1")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestDBCmd_MissingDB(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cfg := config.Default()
	err := dbLocal(cfg, "gmail", "SELECT 1")
	if err == nil {
		t.Fatal("expected error for missing database")
	}
}

func TestDBLocal_QueryExecution(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not in PATH")
	}

	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	gmailDir := filepath.Join(dir, "gmail")
	if err := os.MkdirAll(gmailDir, 0700); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(gmailDir, "data.db")
	c := exec.Command("sqlite3", dbPath,
		"CREATE TABLE test_table (name TEXT); INSERT INTO test_table VALUES ('hello');")
	if err := c.Run(); err != nil {
		t.Fatalf("create test db: %v", err)
	}

	cfg := config.Default()
	err := dbLocal(cfg, "gmail", "SELECT name FROM test_table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDBRemote_NotImplemented(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("mode: remote\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.IsRemote() {
		t.Fatal("expected remote mode")
	}

	// The db command should return not-implemented for remote mode
	err = dbCmd.RunE(dbCmd, []string{"gmail", "SELECT 1"})
	if err == nil {
		t.Fatal("expected error for remote mode")
	}
}
