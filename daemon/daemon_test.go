package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
)

func TestDaemon_RunAndShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmpDir)

	cfg := config.Default()
	cfg.Daemon.JobsStorage.DSN = tmpDir + "/test-jobs.db"
	cfg.WhatsApp.Storage.DSN = tmpDir + "/wa-data.db"
	cfg.Scheduler = &config.SchedulerConfig{
		Storage: config.StorageConfig{Driver: "sqlite", DSN: tmpDir + "/scheduler.db"},
	}

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

func TestNewChannelRegistryRegistersConfiguredTelegram(t *testing.T) {
	cfg := config.Default()
	if _, err := newChannelRegistry(cfg).Get("telegram"); err == nil {
		t.Fatal("telegram registered without a bot token")
	}
	cfg.Channels = &config.ChannelsConfig{Telegram: &config.TelegramConfig{BotToken: "7012345678:AAH-test-token", OwnerID: 8084721928}}
	if _, err := newChannelRegistry(cfg).Get("telegram"); err != nil {
		t.Fatalf("configured telegram not registered: %v", err)
	}
}
