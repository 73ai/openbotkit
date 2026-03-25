package daemon

import (
	"context"
	"log/slog"
	"time"

	"github.com/73ai/openbotkit/config"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

// runWhatsAppSync starts a WhatsApp sync goroutine that runs until ctx is cancelled.
// Errors are sent on the returned channel (non-blocking).
func runWhatsAppSync(ctx context.Context, cfg *config.Config, notifier *SyncNotifier) <-chan error {
	return runWhatsAppSyncForAccount(ctx, cfg, "default", notifier)
}

// runWhatsAppSyncForAccount runs sync for a specific account label.
func runWhatsAppSyncForAccount(ctx context.Context, cfg *config.Config, label string, notifier *SyncNotifier) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		if !config.IsWhatsAppAccountLinked(label) {
			slog.Info("whatsapp: not linked, skipping sync", "account", label)
			return
		}

		sessionDBPath := cfg.WhatsAppAccountSessionDBPath(label)
		client, err := wasrc.NewClient(ctx, sessionDBPath)
		if err != nil {
			slog.Error("whatsapp: failed to create client", "account", label, "error", err)
			errCh <- err
			return
		}

		if !client.IsAuthenticated() {
			slog.Warn("whatsapp: not authenticated, skipping sync", "account", label)
			return
		}

		db, err := store.Open(store.Config{
			Driver: cfg.WhatsApp.Storage.Driver,
			DSN:    cfg.WhatsAppDataDSN(),
		})
		if err != nil {
			slog.Error("whatsapp: failed to open db", "account", label, "error", err)
			errCh <- err
			return
		}
		defer db.Close()

		if notifier != nil {
			go func() {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						notifier.Notify("whatsapp")
					}
				}
			}()
		}

		slog.Info("whatsapp: starting sync", "account", label)
		result, err := wasrc.Sync(ctx, client, db, wasrc.SyncOptions{
			Follow: true,
		})
		if err != nil {
			slog.Error("whatsapp: sync error", "account", label, "error", err)
			errCh <- err
			return
		}

		slog.Info("whatsapp: sync stopped", "account", label,
			"received", result.Received, "history", result.HistoryMessages, "errors", result.Errors)
	}()

	return errCh
}
