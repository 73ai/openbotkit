package daemon

import (
	"context"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/store"
)

func TestRunContactsSync_StartAndStop(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	cfg := config.Default()
	cfg.Contacts.Storage.DSN = tmpDir + "/contacts-test.db"

	ctx, cancel := context.WithCancel(context.Background())

	errCh := runContactsSync(ctx, cfg)

	// Give the goroutine time to start and run initial sync.
	time.Sleep(500 * time.Millisecond)

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runContactsSync returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("contacts sync did not stop within 5s")
	}
}

func TestLinkedSources_OnlyDBSources(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	sourceDBs := map[string]*store.DB{
		"whatsapp": nil,
		"gmail":    nil,
	}
	got := linkedSources(sourceDBs)
	sort.Strings(got)
	want := []string{"gmail", "whatsapp"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLinkedSources_IncludesAppleContactsWhenLinked(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	if err := config.LinkSource("applecontacts"); err != nil {
		t.Fatal(err)
	}
	sourceDBs := map[string]*store.DB{
		"whatsapp": nil,
	}
	got := linkedSources(sourceDBs)
	sort.Strings(got)
	want := []string{"applecontacts", "whatsapp"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLinkedSources_ExcludesAppleContactsWhenNotLinked(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	sourceDBs := map[string]*store.DB{}
	got := linkedSources(sourceDBs)
	if len(got) != 0 {
		t.Fatalf("expected empty sources, got %v", got)
	}
}

func TestMigrateContactsLinking(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("migrateContactsLinking only runs on macOS")
	}
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	// Link "contacts" (the old incorrect name)
	if err := config.LinkSource("contacts"); err != nil {
		t.Fatal(err)
	}

	migrateContactsLinking()

	if !config.IsSourceLinked("applecontacts") {
		t.Error("expected applecontacts to be linked after migration")
	}
}

func TestMigrateContactsLinking_NoOp(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	// Neither "contacts" nor "applecontacts" linked
	migrateContactsLinking()

	if config.IsSourceLinked("applecontacts") {
		t.Error("expected applecontacts to remain unlinked")
	}
}

func TestMigrateContactsLinking_AlreadyMigrated(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("migrateContactsLinking only runs on macOS")
	}
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	// Both linked — should not error
	if err := config.LinkSource("contacts"); err != nil {
		t.Fatal(err)
	}
	if err := config.LinkSource("applecontacts"); err != nil {
		t.Fatal(err)
	}

	migrateContactsLinking()

	if !config.IsSourceLinked("applecontacts") {
		t.Error("expected applecontacts to remain linked")
	}
}

func TestRunContactsSync_NoLinkedSources(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	cfg := config.Default()
	cfg.Contacts.Storage.DSN = tmpDir + "/contacts-test.db"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := runContactsSync(ctx, cfg)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runContactsSync returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("contacts sync did not stop within 5s")
	}
}
