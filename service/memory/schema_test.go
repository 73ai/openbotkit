package memory

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
	dir := filepath.Join(t.TempDir(), "sub", "dir")
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestEnsureDirIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("second: %v", err)
	}
}
