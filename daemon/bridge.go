package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/remote"
	ansrc "github.com/73ai/openbotkit/source/applenotes"
	imsrc "github.com/73ai/openbotkit/source/imessage"
	"github.com/73ai/openbotkit/store"
)

// RunBridge syncs Apple Notes locally and pushes them to the remote server.
// Only works on macOS.
func RunBridge(ctx context.Context, cfg *config.Config, client *remote.Client) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("bridge mode requires macOS")
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

	if err := config.EnsureSourceDir("imessage"); err != nil {
		return fmt.Errorf("ensure imessage dir: %w", err)
	}

	imDB, err := store.Open(store.Config{
		Driver: cfg.IMessage.Storage.Driver,
		DSN:    cfg.IMessageDataDSN(),
	})
	if err != nil {
		return fmt.Errorf("open imessage db: %w", err)
	}
	defer imDB.Close()

	slog.Info("bridge: starting sync")

	b := &bridge{db: db, imDB: imDB, client: client}

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
	db             *store.DB
	imDB           *store.DB
	client         *remote.Client
	lastPushedAt   time.Time
	imLastPushedAt time.Time
}

func (b *bridge) syncAndPush() {
	b.syncAndPushAppleNotes()
	b.syncAndPushIMessage()
}

func (b *bridge) syncAndPushAppleNotes() {
	result, err := ansrc.Sync(b.db, ansrc.SyncOptions{})
	if err != nil {
		slog.Error("bridge: applenotes sync error", "error", err)
		return
	}
	slog.Info("bridge: applenotes sync complete", "synced", result.Synced, "skipped", result.Skipped, "errors", result.Errors)

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
		slog.Error("bridge: applenotes push error", "error", err)
	} else {
		b.lastPushedAt = time.Now()
		slog.Info("bridge: pushed notes to remote", "count", len(notes))
	}
}

func (b *bridge) syncAndPushIMessage() {
	if b.imDB == nil {
		return
	}

	result, err := imsrc.Sync(b.imDB, imsrc.SyncOptions{})
	if err != nil {
		slog.Error("bridge: imessage sync error", "error", err)
		return
	}
	slog.Info("bridge: imessage sync complete", "synced", result.Synced, "skipped", result.Skipped, "errors", result.Errors)

	if result.Synced == 0 {
		return
	}

	msgs, err := imsrc.ListMessagesModifiedSince(b.imDB, b.imLastPushedAt)
	if err != nil {
		slog.Error("bridge: list imessage error", "error", err)
		return
	}

	if len(msgs) == 0 {
		return
	}

	if err := b.client.IMessagePush(msgs); err != nil {
		slog.Error("bridge: imessage push error", "error", err)
	} else {
		b.imLastPushedAt = time.Now()
		slog.Info("bridge: pushed messages to remote", "count", len(msgs))
	}
}
