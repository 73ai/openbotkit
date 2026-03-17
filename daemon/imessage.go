package daemon

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/73ai/openbotkit/config"
	imsrc "github.com/73ai/openbotkit/source/imessage"
	"github.com/73ai/openbotkit/store"
)

const iMessageSyncInterval = 30 * time.Second

func runIMessageSync(ctx context.Context, cfg *config.Config) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		if !config.IsSourceLinked("imessage") {
			slog.Info("imessage: not linked, skipping sync")
			return
		}

		if runtime.GOOS != "darwin" {
			slog.Info("imessage: skipping sync (not macOS)")
			return
		}

		if err := config.EnsureSourceDir("imessage"); err != nil {
			slog.Error("imessage: failed to create dir", "error", err)
			errCh <- err
			return
		}

		db, err := store.Open(store.Config{
			Driver: cfg.IMessage.Storage.Driver,
			DSN:    cfg.IMessageDataDSN(),
		})
		if err != nil {
			slog.Error("imessage: failed to open db", "error", err)
			errCh <- err
			return
		}
		defer db.Close()

		syncIMessage(db)

		ticker := time.NewTicker(iMessageSyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("imessage: stopping sync")
				return
			case <-ticker.C:
				syncIMessage(db)
			}
		}
	}()

	return errCh
}

func syncIMessage(db *store.DB) {
	result, err := imsrc.Sync(db, imsrc.SyncOptions{})
	if err != nil {
		slog.Error("imessage: sync error", "error", err)
		return
	}
	slog.Info("imessage: sync complete", "synced", result.Synced, "skipped", result.Skipped, "errors", result.Errors)
}
