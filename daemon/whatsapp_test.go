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

func TestWatchWhatsAppAccountsStartsAccountsLinkedLater(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	if err := config.LinkWhatsAppAccount("default"); err != nil {
		t.Fatalf("link default: %v", err)
	}
	cfg := config.Default()
	cfg.SetWhatsAppAccount("assistant", config.WhatsAppAccount{Role: "channel", OwnerJID: "919876543210@s.whatsapp.net"})
	cfg.SetWhatsAppAccount("work", config.WhatsAppAccount{Role: "source"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	started := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	done := watchWhatsAppAccounts(ctx, 10*time.Millisecond, func(ctx context.Context, _ *config.Config, label string) <-chan error {
		started <- label
		errCh := make(chan error, 1)
		go func() { <-ctx.Done(); close(errCh) }()
		return errCh
	})

	expect := func(want string) {
		t.Helper()
		select {
		case got := <-started:
			if got != want {
				t.Fatalf("started %q, want %q", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%q not started", want)
		}
	}
	expect("default")

	cfg.SetWhatsAppAccount("family", config.WhatsAppAccount{Role: "both", OwnerJID: "919812345678@s.whatsapp.net"})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := config.LinkWhatsAppAccount("family"); err != nil {
		t.Fatalf("link family: %v", err)
	}
	expect("family")
	if err := config.LinkWhatsAppAccount("work"); err != nil {
		t.Fatalf("link work: %v", err)
	}
	expect("work")

	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop")
	}
	if len(started) != 0 {
		t.Fatalf("started again: %q", <-started)
	}
}
