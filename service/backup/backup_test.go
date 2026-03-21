package backup

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestManifestDiff(t *testing.T) {
	old := &Manifest{
		Files: map[string]ManifestFile{
			"config.yaml":   {Hash: "sha256:aaa"},
			"gmail/data.db": {Hash: "sha256:bbb"},
			"removed.db":    {Hash: "sha256:ccc"},
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

func TestManifestDiffEmpty(t *testing.T) {
	old := &Manifest{Files: make(map[string]ManifestFile)}
	current := map[string]string{}
	diff := DiffManifest(old, current)
	if len(diff.Changed) != 0 {
		t.Errorf("expected 0 changed, got %d", len(diff.Changed))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(diff.Removed))
	}
}

func TestManifestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := NewManifest("testhost")
	m.Files["config.yaml"] = ManifestFile{
		Hash:           "sha256:abc123",
		Size:           1024,
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
	if f.Size != 1024 {
		t.Errorf("size = %d, want 1024", f.Size)
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
	mkFile(t, dir, "whatsapp/data.db")
	mkFile(t, dir, "whatsapp/session.db")
	mkFile(t, dir, "learnings/topic/note.md")
	mkFile(t, dir, "models/custom.json")
	mkFile(t, dir, "providers/google/creds.json")
	mkFile(t, dir, "skills/email/metadata.yaml")
	mkFile(t, dir, "env")
	mkFile(t, dir, "ngrok.yml")
	mkFile(t, dir, "applenotes/config.json")
	mkFile(t, dir, "applecontacts/config.json")

	// Create excluded files.
	mkFile(t, dir, "gmail/data.db-wal")
	mkFile(t, dir, "gmail/data.db-shm")
	mkFile(t, dir, "daemon.log")
	mkFile(t, dir, "server.log")
	mkFile(t, dir, "jobs.db")
	mkFile(t, dir, "backup/last_manifest.json")
	mkFile(t, dir, "scratch/session/tmp.txt")
	mkFile(t, dir, "bin/obk")
	mkFile(t, dir, "something.lock")

	files, err := ScanFiles(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	fileSet := make(map[string]bool)
	for _, f := range files {
		fileSet[f] = true
	}

	expected := []string{
		"config.yaml", "gmail/data.db", "whatsapp/data.db",
		"whatsapp/session.db", "learnings/topic/note.md",
		"models/custom.json", "providers/google/creds.json",
		"skills/email/metadata.yaml", "env", "ngrok.yml",
		"applenotes/config.json", "applecontacts/config.json",
	}
	for _, e := range expected {
		if !fileSet[e] {
			t.Errorf("expected %q to be included", e)
		}
	}

	excluded := []string{
		"gmail/data.db-wal", "gmail/data.db-shm", "daemon.log",
		"server.log", "jobs.db", "backup/last_manifest.json",
		"scratch/session/tmp.txt", "bin/obk", "something.lock",
	}
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

	data := []byte("hello world")
	if err := backend.Put(ctx, "objects/ab/test", bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("put: %v", err)
	}

	exists, err := backend.Head(ctx, "objects/ab/test")
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if !exists {
		t.Error("expected object to exist")
	}

	exists, err = backend.Head(ctx, "objects/ab/missing")
	if err != nil {
		t.Fatalf("head missing: %v", err)
	}
	if exists {
		t.Error("expected object to not exist")
	}

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

	keys, err := backend.List(ctx, "objects/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0] != "objects/ab/test" {
		t.Errorf("list = %v, want [objects/ab/test]", keys)
	}

	if err := backend.Delete(ctx, "objects/ab/test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	exists, _ = backend.Head(ctx, "objects/ab/test")
	if exists {
		t.Error("expected object to be deleted")
	}
}

func TestLocalBackendListEmpty(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	keys, err := backend.List(context.Background(), "nonexistent/")
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

func TestLocalBackendDeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	backend := NewLocalBackend(dir)
	if err := backend.Delete(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("delete nonexistent should succeed: %v", err)
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
	if len(compressed) == 0 {
		t.Fatal("compressed output is empty")
	}

	decompressed, err := decompressData(compressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}

	if string(decompressed) != content {
		t.Errorf("round-trip mismatch: got %q, want %q", decompressed, content)
	}
}

func TestVacuumInto(t *testing.T) {
	dir := t.TempDir()
	stagingDir := t.TempDir()

	// Create a real SQLite database.
	dbPath := filepath.Join(dir, "gmail", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, val TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO test (val) VALUES ('hello')"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	vacuumed, err := VacuumInto(dbPath, stagingDir, "gmail/data.db")
	if err != nil {
		t.Fatalf("vacuum into: %v", err)
	}

	// The vacuumed file should be at stagingDir/gmail/data.db.
	expectedPath := filepath.Join(stagingDir, "gmail", "data.db")
	if vacuumed != expectedPath {
		t.Errorf("vacuumed path = %q, want %q", vacuumed, expectedPath)
	}

	// Verify the vacuumed database is readable.
	vdb, err := sql.Open("sqlite", vacuumed)
	if err != nil {
		t.Fatal(err)
	}
	defer vdb.Close()
	var val string
	if err := vdb.QueryRow("SELECT val FROM test WHERE id = 1").Scan(&val); err != nil {
		t.Fatalf("read from vacuumed db: %v", err)
	}
	if val != "hello" {
		t.Errorf("val = %q, want hello", val)
	}
}

func TestVacuumIntoNoCollision(t *testing.T) {
	dir := t.TempDir()
	stagingDir := t.TempDir()

	// Create two databases with the same filename in different directories.
	for _, sub := range []string{"gmail", "whatsapp"} {
		dbPath := filepath.Join(dir, sub, "data.db")
		if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
			t.Fatal(err)
		}
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("CREATE TABLE test (source TEXT)"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("INSERT INTO test (source) VALUES (?)", sub); err != nil {
			t.Fatal(err)
		}
		db.Close()
	}

	// Vacuum both — they should NOT collide.
	v1, err := VacuumInto(filepath.Join(dir, "gmail", "data.db"), stagingDir, "gmail/data.db")
	if err != nil {
		t.Fatal(err)
	}
	v2, err := VacuumInto(filepath.Join(dir, "whatsapp", "data.db"), stagingDir, "whatsapp/data.db")
	if err != nil {
		t.Fatal(err)
	}

	if v1 == v2 {
		t.Fatalf("path collision: both vacuumed to %q", v1)
	}

	// Verify each database has the correct data.
	for _, tc := range []struct {
		path   string
		source string
	}{
		{v1, "gmail"},
		{v2, "whatsapp"},
	} {
		db, err := sql.Open("sqlite", tc.path)
		if err != nil {
			t.Fatal(err)
		}
		var source string
		if err := db.QueryRow("SELECT source FROM test").Scan(&source); err != nil {
			t.Fatal(err)
		}
		db.Close()
		if source != tc.source {
			t.Errorf("source = %q, want %q (path: %s)", source, tc.source, tc.path)
		}
	}
}

func TestServiceRun(t *testing.T) {
	baseDir := t.TempDir()
	remoteDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "last_manifest.json")
	stagingDir := t.TempDir()

	mkFileWithContent(t, baseDir, "config.yaml", "mode: local")
	mkFileWithContent(t, baseDir, "learnings/topic/note.md", "some learning")

	backend := NewLocalBackend(remoteDir)
	svc := NewWithPaths(backend, baseDir, manifestPath, stagingDir)
	ctx := context.Background()

	// First run: everything is new.
	result, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if result.Changed != 2 {
		t.Errorf("first run: changed = %d, want 2", result.Changed)
	}
	if result.Skipped != 0 {
		t.Errorf("first run: skipped = %d, want 0", result.Skipped)
	}
	if result.Uploaded == 0 {
		t.Error("first run: expected some bytes uploaded")
	}

	// Verify manifest was saved.
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if len(manifest.Files) != 2 {
		t.Errorf("manifest files = %d, want 2", len(manifest.Files))
	}

	// Verify objects were uploaded.
	objects, err := backend.List(ctx, "objects/")
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if len(objects) != 2 {
		t.Errorf("uploaded objects = %d, want 2", len(objects))
	}

	// Verify snapshot manifest was uploaded.
	snapshots, err := backend.List(ctx, "snapshots/")
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("snapshots = %d, want 1", len(snapshots))
	}

	// Second run: nothing changed.
	result2, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if result2.Changed != 0 {
		t.Errorf("second run: changed = %d, want 0", result2.Changed)
	}
	if result2.Skipped != 2 {
		t.Errorf("second run: skipped = %d, want 2", result2.Skipped)
	}
	if result2.Uploaded != 0 {
		t.Errorf("second run: uploaded = %d, want 0", result2.Uploaded)
	}
}

func TestServiceRunIncremental(t *testing.T) {
	baseDir := t.TempDir()
	remoteDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "last_manifest.json")
	stagingDir := t.TempDir()

	mkFileWithContent(t, baseDir, "config.yaml", "mode: local")
	backend := NewLocalBackend(remoteDir)
	svc := NewWithPaths(backend, baseDir, manifestPath, stagingDir)
	ctx := context.Background()

	// First run.
	result1, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if result1.Changed != 1 {
		t.Fatalf("first run: changed = %d, want 1", result1.Changed)
	}

	// Modify the file.
	mkFileWithContent(t, baseDir, "config.yaml", "mode: remote")

	// Second run: should detect the change.
	result2, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if result2.Changed != 1 {
		t.Errorf("second run: changed = %d, want 1", result2.Changed)
	}

	// Should have at least 1 snapshot (2 if runs happen in different seconds).
	snapshots, err := backend.List(ctx, "snapshots/")
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) < 1 {
		t.Errorf("snapshots = %d, want >= 1", len(snapshots))
	}

	// The manifest should reflect the latest state.
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := manifest.Files["config.yaml"]; !ok {
		t.Error("config.yaml missing from manifest")
	} else if f.Size == 0 {
		t.Error("config.yaml size should be > 0")
	}
}

func TestServiceRestore(t *testing.T) {
	baseDir := t.TempDir()
	remoteDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "last_manifest.json")
	stagingDir := t.TempDir()

	mkFileWithContent(t, baseDir, "config.yaml", "mode: local")
	mkFileWithContent(t, baseDir, "learnings/topic/note.md", "important note")

	backend := NewLocalBackend(remoteDir)
	svc := NewWithPaths(backend, baseDir, manifestPath, stagingDir)
	ctx := context.Background()

	// Run backup.
	_, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}

	// Get the snapshot ID.
	snapshots, err := svc.ListSnapshots(ctx)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
	snapshotID := snapshots[0]

	// Verify GetManifest works.
	manifest, err := svc.GetManifest(ctx, snapshotID)
	if err != nil {
		t.Fatalf("get manifest: %v", err)
	}
	if len(manifest.Files) != 2 {
		t.Errorf("manifest files = %d, want 2", len(manifest.Files))
	}

	// Restore to a fresh directory.
	restoreDir := t.TempDir()
	restoreSvc := NewWithPaths(backend, restoreDir, filepath.Join(t.TempDir(), "m.json"), t.TempDir())

	result, err := restoreSvc.Restore(ctx, snapshotID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if result.Restored != 2 {
		t.Errorf("restored = %d, want 2", result.Restored)
	}

	// Verify restored files.
	got, err := os.ReadFile(filepath.Join(restoreDir, "config.yaml"))
	if err != nil {
		t.Fatalf("read restored config.yaml: %v", err)
	}
	if string(got) != "mode: local" {
		t.Errorf("restored config.yaml = %q, want %q", got, "mode: local")
	}

	got, err = os.ReadFile(filepath.Join(restoreDir, "learnings/topic/note.md"))
	if err != nil {
		t.Fatalf("read restored note.md: %v", err)
	}
	if string(got) != "important note" {
		t.Errorf("restored note.md = %q, want %q", got, "important note")
	}
}

func TestServiceListSnapshotsEmpty(t *testing.T) {
	remoteDir := t.TempDir()
	backend := NewLocalBackend(remoteDir)
	svc := NewWithPaths(backend, t.TempDir(), filepath.Join(t.TempDir(), "m.json"), t.TempDir())

	snapshots, err := svc.ListSnapshots(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestServiceRunWithSQLiteDB(t *testing.T) {
	baseDir := t.TempDir()
	remoteDir := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "last_manifest.json")
	stagingDir := t.TempDir()

	// Create a real SQLite database.
	dbPath := filepath.Join(baseDir, "gmail", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE emails (id INTEGER PRIMARY KEY, subject TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO emails (subject) VALUES ('test email')"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	mkFileWithContent(t, baseDir, "config.yaml", "mode: local")

	backend := NewLocalBackend(remoteDir)
	svc := NewWithPaths(backend, baseDir, manifestPath, stagingDir)
	ctx := context.Background()

	result, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Changed != 2 {
		t.Errorf("changed = %d, want 2", result.Changed)
	}

	// Restore and verify DB is intact.
	restoreDir := t.TempDir()
	restoreSvc := NewWithPaths(backend, restoreDir, filepath.Join(t.TempDir(), "m.json"), t.TempDir())

	snapshots, _ := svc.ListSnapshots(ctx)
	_, err = restoreSvc.Restore(ctx, snapshots[0])
	if err != nil {
		t.Fatalf("restore: %v", err)
	}

	restoredDB, err := sql.Open("sqlite", filepath.Join(restoreDir, "gmail", "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer restoredDB.Close()

	var subject string
	if err := restoredDB.QueryRow("SELECT subject FROM emails WHERE id = 1").Scan(&subject); err != nil {
		t.Fatalf("read restored db: %v", err)
	}
	if subject != "test email" {
		t.Errorf("subject = %q, want 'test email'", subject)
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
