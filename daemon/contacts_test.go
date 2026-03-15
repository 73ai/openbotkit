package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/config"
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
