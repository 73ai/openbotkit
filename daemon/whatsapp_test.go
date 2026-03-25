package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
)

func TestRunWhatsAppSyncForAccount_ShutdownOnCancel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	cfg := config.Default()
	cfg.WhatsApp.Storage.DSN = tmpDir + "/wa-test.db"

	ctx, cancel := context.WithCancel(context.Background())

	// Account is not linked, so sync will return quickly.
	errCh := runWhatsAppSyncForAccount(ctx, cfg, "testaccount", nil)

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runWhatsAppSyncForAccount returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("whatsapp sync did not stop within 5s")
	}
}

func TestRunWhatsAppSyncForAccount_DefaultLabel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	cfg := config.Default()
	cfg.WhatsApp.Storage.DSN = tmpDir + "/wa-test.db"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// "default" label delegates to IsSourceLinked("whatsapp"), which is false.
	errCh := runWhatsAppSyncForAccount(ctx, cfg, "default", nil)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not return within 5s")
	}
}

func TestDaemon_MultiAccountDispatch(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	cfg := config.Default()
	cfg.Daemon.JobsStorage.DSN = tmpDir + "/test-jobs.db"
	cfg.WhatsApp.Storage.DSN = tmpDir + "/wa-data.db"
	cfg.WhatsApp.Accounts = map[string]*config.WhatsAppAccount{
		"personal":  {Role: "source"},
		"assistant": {Role: "channel", OwnerJID: "123@s.whatsapp.net"},
	}
	cfg.Scheduler = &config.SchedulerConfig{
		Storage: config.StorageConfig{Driver: "sqlite", DSN: tmpDir + "/scheduler.db"},
	}

	d := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	// Give daemon time to start.
	time.Sleep(2 * time.Second)

	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("Daemon.Run returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not shut down within 10s")
	}
}
