package daemon

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/channel"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/hooks"
	"github.com/73ai/openbotkit/store"
)

func TestNewRiverClient(t *testing.T) {
	cfg := config.Default()
	cfg.Daemon.GmailSyncPeriod = "1m"

	tmpDir := t.TempDir()
	cfg.Daemon.JobsStorage.DSN = tmpDir + "/test-jobs.db"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hooksDB, err := store.Open(store.SQLiteConfig(filepath.Join(tmpDir, "hooks.db")))
	if err != nil {
		t.Fatalf("open hooks db: %v", err)
	}
	defer hooksDB.Close()
	hooks.Migrate(hooksDB)

	client, db, err := newRiverClient(ctx, cfg, NewSyncNotifier(), channel.NewRegistry(), hooksDB)
	if err != nil {
		t.Fatalf("newRiverClient failed: %v", err)
	}
	defer db.Close()

	if client == nil {
		t.Fatal("client is nil")
	}

	// Start and stop to verify it's functional.
	if err := client.Start(ctx); err != nil {
		t.Fatalf("client.Start failed: %v", err)
	}

	if err := client.Stop(ctx); err != nil {
		t.Fatalf("client.Stop failed: %v", err)
	}
}
