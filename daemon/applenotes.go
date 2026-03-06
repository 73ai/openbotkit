package daemon

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/priyanshujain/openbotkit/config"
	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	"github.com/priyanshujain/openbotkit/store"
)

const appleNotesSyncInterval = 30 * time.Second

// runAppleNotesSync starts a goroutine that periodically syncs Apple Notes.
// Only runs on macOS. Errors are sent on the returned channel.
func runAppleNotesSync(ctx context.Context, cfg *config.Config) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		if !config.IsSourceLinked("applenotes") {
			log.Println("applenotes: not linked, skipping sync")
			return
		}

		if runtime.GOOS != "darwin" {
			log.Println("applenotes: skipping sync (not macOS)")
			return
		}

		if err := config.EnsureSourceDir("applenotes"); err != nil {
			log.Printf("applenotes: failed to create dir: %v", err)
			errCh <- err
			return
		}

		db, err := store.Open(store.Config{
			Driver: cfg.AppleNotes.Storage.Driver,
			DSN:    cfg.AppleNotesDataDSN(),
		})
		if err != nil {
			log.Printf("applenotes: failed to open db: %v", err)
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
				log.Println("applenotes: stopping sync")
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
		log.Printf("applenotes: sync error: %v", err)
		return
	}
	log.Printf("applenotes: synced=%d skipped=%d errors=%d",
		result.Synced, result.Skipped, result.Errors)
}
