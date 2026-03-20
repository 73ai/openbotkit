package history

import (
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}
	return NewStore(dir)
}

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("first ensure: %v", err)
	}

	// Verify sessions subdirectory was created.
	info, err := os.Stat(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatalf("sessions dir missing: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("sessions is not a directory")
	}
}

func TestEnsureDirIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("second should be idempotent: %v", err)
	}
}
