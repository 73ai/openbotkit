package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/remote"
	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	"github.com/priyanshujain/openbotkit/store"
)

// RunBridge syncs Apple Notes locally and pushes them to the remote server.
// Only works on macOS.
func RunBridge(ctx context.Context, cfg *config.Config, client *remote.Client) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("bridge mode requires macOS (for Apple Notes)")
	}

	if err := config.EnsureSourceDir("applenotes"); err != nil {
		return fmt.Errorf("ensure applenotes dir: %w", err)
	}

	db, err := store.Open(store.Config{
		Driver: cfg.AppleNotes.Storage.Driver,
		DSN:    cfg.AppleNotesDataDSN(),
	})
	if err != nil {
		return fmt.Errorf("open applenotes db: %w", err)
	}
	defer db.Close()

	slog.Info("bridge: starting apple notes sync")

	b := &bridge{db: db, client: client}

	// Initial sync + push
	b.syncAndPush()

	ticker := time.NewTicker(appleNotesSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("bridge: stopping")
			return nil
		case <-ticker.C:
			b.syncAndPush()
		}
	}
}

type bridge struct {
	db           *store.DB
	client       *remote.Client
	lastPushedAt time.Time
}

func (b *bridge) syncAndPush() {
	result, err := ansrc.Sync(b.db, ansrc.SyncOptions{})
	if err != nil {
		slog.Error("bridge: sync error", "error", err)
		return
	}
	slog.Info("bridge: sync complete", "synced", result.Synced, "skipped", result.Skipped, "errors", result.Errors)

	if result.Synced == 0 {
		return
	}

	notes, err := ansrc.ListNotesModifiedSince(b.db, b.lastPushedAt)
	if err != nil {
		slog.Error("bridge: list notes error", "error", err)
		return
	}

	if len(notes) == 0 {
		return
	}

	if err := b.client.AppleNotesPush(notes); err != nil {
		slog.Error("bridge: push error", "error", err)
	} else {
		b.lastPushedAt = time.Now()
		slog.Info("bridge: pushed notes to remote", "count", len(notes))
	}
}
