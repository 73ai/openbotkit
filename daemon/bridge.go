package daemon

import (
	"context"
	"fmt"
	"log"
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

	log.Println("bridge: starting Apple Notes sync")

	// Initial sync + push
	bridgeSyncAndPush(db, client)

	ticker := time.NewTicker(appleNotesSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("bridge: stopping")
			return nil
		case <-ticker.C:
			bridgeSyncAndPush(db, client)
		}
	}
}

func bridgeSyncAndPush(db *store.DB, client *remote.Client) {
	result, err := ansrc.Sync(db, ansrc.SyncOptions{})
	if err != nil {
		log.Printf("bridge: sync error: %v", err)
		return
	}
	log.Printf("bridge: synced=%d skipped=%d errors=%d",
		result.Synced, result.Skipped, result.Errors)

	if result.Synced == 0 {
		return
	}

	notes, err := ansrc.ListNotes(db, ansrc.ListOptions{Limit: result.Synced})
	if err != nil {
		log.Printf("bridge: list notes error: %v", err)
		return
	}

	if err := client.AppleNotesPush(notes); err != nil {
		log.Printf("bridge: push error: %v", err)
	} else {
		log.Printf("bridge: pushed %d notes to remote", len(notes))
	}
}
