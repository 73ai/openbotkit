package daemon

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/73ai/openbotkit/config"
	ansrc "github.com/73ai/openbotkit/source/applenotes"
	"github.com/73ai/openbotkit/store"
)

const appleNotesSyncInterval = 30 * time.Second

// runAppleNotesSync starts a goroutine that periodically syncs Apple Notes.
// Only runs on macOS. Errors are sent on the returned channel.
func runAppleNotesSync(ctx context.Context, cfg *config.Config) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		if !config.IsSourceLinked("applenotes") {
			slog.Info("applenotes: not linked, skipping sync")
			return
		}

		if runtime.GOOS != "darwin" {
			slog.Info("applenotes: skipping sync (not macOS)")
			return
		}

		if err := config.EnsureSourceDir("applenotes"); err != nil {
			slog.Error("applenotes: failed to create dir", "error", err)
			errCh <- err
			return
		}

		db, err := store.Open(store.Config{
			Driver: cfg.AppleNotes.Storage.Driver,
			DSN:    cfg.AppleNotesDataDSN(),
		})
		if err != nil {
			slog.Error("applenotes: failed to open db", "error", err)
			errCh <- err
			return
		}
		defer db.Close()

		// Run initial sync immediately.
		syncAppleNotes(db)

		ticker := time.NewTicker(appleNotesSyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("applenotes: stopping sync")
				return
			case <-ticker.C:
				syncAppleNotes(db)
			}
		}
	}()

	return errCh
}

func syncAppleNotes(db *store.DB) {
	result, err := ansrc.Sync(db, ansrc.SyncOptions{})
	if err != nil {
		slog.Error("applenotes: sync error", "error", err)
		return
	}
	slog.Info("applenotes: sync complete", "synced", result.Synced, "skipped", result.Skipped, "errors", result.Errors)
}
