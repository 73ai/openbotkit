package daemon

import (
	"context"
	"log/slog"

	"github.com/73ai/openbotkit/config"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

// runWhatsAppSync starts a WhatsApp sync goroutine that runs until ctx is cancelled.
// Errors are sent on the returned channel (non-blocking).
func runWhatsAppSync(ctx context.Context, cfg *config.Config) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		if !config.IsSourceLinked("whatsapp") {
			slog.Info("whatsapp: not linked, skipping sync")
			return
		}

		client, err := wasrc.NewClient(ctx, cfg.WhatsAppSessionDBPath())
		if err != nil {
			slog.Error("whatsapp: failed to create client", "error", err)
			errCh <- err
			return
		}

		if !client.IsAuthenticated() {
			slog.Warn("whatsapp: not authenticated, skipping sync (run 'obk whatsapp auth' first)")
			return
		}

		db, err := store.Open(store.Config{
			Driver: cfg.WhatsApp.Storage.Driver,
			DSN:    cfg.WhatsAppDataDSN(),
		})
		if err != nil {
			slog.Error("whatsapp: failed to open db", "error", err)
			errCh <- err
			return
		}
		defer db.Close()

		slog.Info("whatsapp: starting sync")
		result, err := wasrc.Sync(ctx, client, db, wasrc.SyncOptions{
			Follow: true,
		})
		if err != nil {
			slog.Error("whatsapp: sync error", "error", err)
			errCh <- err
			return
		}

		slog.Info("whatsapp: sync stopped", "received", result.Received, "history", result.HistoryMessages, "errors", result.Errors)
	}()

	return errCh
}
