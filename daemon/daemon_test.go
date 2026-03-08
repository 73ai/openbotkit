package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/config"
)

func TestDaemon_RunAndShutdown(t *testing.T) {
	cfg := config.Default()

	tmpDir := t.TempDir()
	cfg.Daemon.JobsStorage.DSN = tmpDir + "/test-jobs.db"
	// Point WhatsApp session DB to tmp so it doesn't conflict.
	cfg.WhatsApp.Storage.DSN = tmpDir + "/wa-data.db"

	d := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(ctx)
	}()

	// Give the daemon time to start. Pure-Go SQLite (modernc) is slower
	// to initialize on Windows, so we need more than 500ms.
	time.Sleep(2 * time.Second)

	// Signal shutdown.
	cancel()

	select {
	case err := <-errCh:
		// context.Canceled is expected since we explicitly canceled.
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Daemon.Run returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("daemon did not shut down within 10s")
	}
}
