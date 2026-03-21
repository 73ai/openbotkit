package backup

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestDiff(t *testing.T) {
	old := &Manifest{
		Files: map[string]ManifestFile{
			"config.yaml":    {Hash: "sha256:aaa"},
			"gmail/data.db":  {Hash: "sha256:bbb"},
			"removed.db":     {Hash: "sha256:ccc"},
		},
	}

	current := map[string]string{
		"config.yaml":   "sha256:aaa", // unchanged
		"gmail/data.db": "sha256:ddd", // changed
		"new/data.db":   "sha256:eee", // new
	}

	diff := DiffManifest(old, current)

	changedSet := make(map[string]bool)
	for _, c := range diff.Changed {
		changedSet[c] = true
	}
	if !changedSet["gmail/data.db"] {
		t.Error("expected gmail/data.db to be changed")
	}
	if !changedSet["new/data.db"] {
		t.Error("expected new/data.db to be changed (new)")
	}
	if changedSet["config.yaml"] {
		t.Error("config.yaml should not be changed")
	}

	removedSet := make(map[string]bool)
	for _, r := range diff.Removed {
		removedSet[r] = true
	}
	if !removedSet["removed.db"] {
		t.Error("expected removed.db to be in removed list")
	}
}

func TestManifestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := NewManifest("testhost")
	m.Files["config.yaml"] = ManifestFile{
		Hash: "sha256:abc123",
		Size: 1024,
		CompressedSize: 512,
	}

	if err := SaveManifest(path, m); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Hostname != "testhost" {
		t.Errorf("hostname = %q, want testhost", loaded.Hostname)
	}
	if len(loaded.Files) != 1 {
		t.Errorf("files = %d, want 1", len(loaded.Files))
	}
	f := loaded.Files["config.yaml"]
	if f.Hash != "sha256:abc123" {
		t.Errorf("hash = %q, want sha256:abc123", f.Hash)
	}
}

func TestManifestLoadMissing(t *testing.T) {
	m, err := LoadManifest("/nonexistent/manifest.json")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if m.Files == nil {
		t.Error("expected empty but non-nil Files map")
	}
	if len(m.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(m.Files))
	}
}

func TestScanFiles(t *testing.T) {
	dir := t.TempDir()

	// Create included files.
	mkFile(t, dir, "config.yaml")
	mkFile(t, dir, "gmail/data.db")
	mkFile(t, dir, "learnings/topic/note.md")
	mkFile(t, dir, "models/custom.json")

	// Create excluded files.
	mkFile(t, dir, "gmail/data.db-wal")
	mkFile(t, dir, "daemon.log")
	mkFile(t, dir, "jobs.db")
	mkFile(t, dir, "backup/last_manifest.json")
	mkFile(t, dir, "scratch/session/tmp.txt")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	expected := []string{"config.yaml", "gmail/data.db", "learnings/topic/note.md", "models/custom.json"}
	for _, e := range expected {
		if !fileSet[e] {
			t.Errorf("expected %q to be included", e)
		}
	}

	excluded := []string{"gmail/data.db-wal", "daemon.log", "jobs.db", "backup/last_manifest.json", "scratch/session/tmp.txt"}
	for _, e := range excluded {
		if fileSet[e] {
			t.Errorf("expected %q to be excluded", e)
		}
	}
}

func TestLocalBackendPutGetHeadListDelete(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	ctx := context.Background()

	// Put
	data := []byte("hello world")
	if err := backend.Put(ctx, "objects/ab/test", bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("put: %v", err)
	}

	// Head
	exists, err := backend.Head(ctx, "objects/ab/test")
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if !exists {
		t.Error("expected object to exist")
	}

	// Head non-existent
	exists, err = backend.Head(ctx, "objects/ab/missing")
	if err != nil {
		t.Fatalf("head missing: %v", err)
	}
	if exists {
		t.Error("expected object to not exist")
	}

	// Get
	rc, err := backend.Get(ctx, "objects/ab/test")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}

	// List
	keys, err := backend.List(ctx, "objects/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0] != "objects/ab/test" {
		t.Errorf("list = %v, want [objects/ab/test]", keys)
	}

	// Delete
	if err := backend.Delete(ctx, "objects/ab/test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	exists, _ = backend.Head(ctx, "objects/ab/test")
	if exists {
		t.Error("expected object to be deleted")
	}
}

func TestFullBackupFlow(t *testing.T) {
	// Set up a fake ~/.obk with some files.
	baseDir := t.TempDir()
	mkFile(t, baseDir, "config.yaml")
	mkFileWithContent(t, baseDir, "learnings/topic/note.md", "some learning")

	// Set up remote storage (local backend).
	remoteDir := t.TempDir()
	backend := NewLocalBackend(remoteDir)

	// Override the backup paths to use temp dirs.
	backupDir := t.TempDir()
	manifestPath := filepath.Join(backupDir, "last_manifest.json")

	// We need to override where the manifest is saved.
	// For testing, we'll run the lower-level functions directly.
	ctx := context.Background()

	// First backup: everything is new.
	files, err := ScanFiles(baseDir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(files) < 1 {
		t.Fatalf("expected at least 1 file, got %d", len(files))
	}

	hashes := make(map[string]string)
	for _, rel := range files {
		absPath := filepath.Join(baseDir, rel)
		hash, err := hashFile(absPath)
		if err != nil {
			t.Fatalf("hash %s: %v", rel, err)
		}
		hashes[rel] = "sha256:" + hash
	}

	lastManifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	diff := DiffManifest(lastManifest, hashes)
	if len(diff.Changed) != len(files) {
		t.Errorf("first backup: changed = %d, want %d", len(diff.Changed), len(files))
	}

	// Upload changed files.
	manifest := NewManifest("testhost")
	for _, rel := range diff.Changed {
		absPath := filepath.Join(baseDir, rel)
		compressed, err := compressFile(absPath)
		if err != nil {
			t.Fatalf("compress %s: %v", rel, err)
		}

		hash := hashes[rel]
		objectKey := objectKeyFromHash(hash)

		if err := backend.Put(ctx, objectKey, bytes.NewReader(compressed), int64(len(compressed))); err != nil {
			t.Fatalf("upload %s: %v", rel, err)
		}

		info, _ := os.Stat(absPath)
		manifest.Files[rel] = ManifestFile{
			Hash:           hash,
			Size:           info.Size(),
			CompressedSize: int64(len(compressed)),
		}
	}

	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	// Second backup: nothing changed.
	hashes2 := make(map[string]string)
	for _, rel := range files {
		absPath := filepath.Join(baseDir, rel)
		hash, err := hashFile(absPath)
		if err != nil {
			t.Fatalf("hash %s: %v", rel, err)
		}
		hashes2[rel] = "sha256:" + hash
	}

	lastManifest2, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest 2: %v", err)
	}

	diff2 := DiffManifest(lastManifest2, hashes2)
	if len(diff2.Changed) != 0 {
		t.Errorf("second backup: changed = %d, want 0", len(diff2.Changed))
	}
}

func TestObjectKeyFromHash(t *testing.T) {
	key := objectKeyFromHash("sha256:abcdef0123456789")
	want := "objects/ab/abcdef0123456789"
	if key != want {
		t.Errorf("objectKeyFromHash = %q, want %q", key, want)
	}
}

func TestCompressDecompress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello world! this is a test of compression."
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	compressed, err := compressFile(path)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}

	decompressed, err := decompressData(compressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}

	if string(decompressed) != content {
		t.Errorf("round-trip mismatch: got %q, want %q", decompressed, content)
	}
}

func mkFile(t *testing.T, base, rel string) {
	t.Helper()
	mkFileWithContent(t, base, rel, "test content")
}

func mkFileWithContent(t *testing.T, base, rel, content string) {
	t.Helper()
	path := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

