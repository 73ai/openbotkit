package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProviderDir(t *testing.T) {
	dir := ProviderDir("google")
	if !strings.HasSuffix(dir, filepath.Join("providers", "google")) {
		t.Fatalf("expected path ending in providers/google, got %q", dir)
	}
}

func TestEnsureProviderDir(t *testing.T) {
	old := os.Getenv("OBK_CONFIG_DIR")
	tmp := t.TempDir()
	os.Setenv("OBK_CONFIG_DIR", tmp)
	defer os.Setenv("OBK_CONFIG_DIR", old)

	if err := EnsureProviderDir("google"); err != nil {
		t.Fatalf("ensure provider dir: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmp, "providers", "google"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestScratchDir_Path(t *testing.T) {
	dir := ScratchDir("sess-123")
	if !strings.HasSuffix(dir, filepath.Join("scratch", "sess-123")) {
		t.Fatalf("expected path ending in scratch/sess-123, got %q", dir)
	}
}

func TestEnsureScratchDir_Creates(t *testing.T) {
	old := os.Getenv("OBK_CONFIG_DIR")
	tmp := t.TempDir()
	os.Setenv("OBK_CONFIG_DIR", tmp)
	defer os.Setenv("OBK_CONFIG_DIR", old)

	if err := EnsureScratchDir("sess-abc"); err != nil {
		t.Fatalf("ensure scratch dir: %v", err)
	}
	info, err := os.Stat(filepath.Join(tmp, "scratch", "sess-abc"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
		t.Errorf("perm = %o, want 0700", info.Mode().Perm())
	}
}

func TestCleanScratch_Removes(t *testing.T) {
	old := os.Getenv("OBK_CONFIG_DIR")
	tmp := t.TempDir()
	os.Setenv("OBK_CONFIG_DIR", tmp)
	defer os.Setenv("OBK_CONFIG_DIR", old)

	if err := EnsureScratchDir("sess-rm"); err != nil {
		t.Fatal(err)
	}
	// Write a file inside.
	os.WriteFile(filepath.Join(ScratchDir("sess-rm"), "test.txt"), []byte("hi"), 0600)

	if err := CleanScratch("sess-rm"); err != nil {
		t.Fatalf("clean scratch: %v", err)
	}
	if _, err := os.Stat(ScratchDir("sess-rm")); !os.IsNotExist(err) {
		t.Error("scratch dir should be removed")
	}
}

func TestCleanScratch_NonExistent(t *testing.T) {
	old := os.Getenv("OBK_CONFIG_DIR")
	tmp := t.TempDir()
	os.Setenv("OBK_CONFIG_DIR", tmp)
	defer os.Setenv("OBK_CONFIG_DIR", old)

	if err := CleanScratch("does-not-exist"); err != nil {
		t.Fatalf("clean non-existent: %v", err)
	}
}
