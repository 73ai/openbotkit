package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestBackupArgs_Kind(t *testing.T) {
	args := BackupArgs{}
	if args.Kind() != "backup" {
		t.Errorf("Kind() = %q, want %q", args.Kind(), "backup")
	}
}

func TestBackupWorker_notifyFailure_NoTelegram(t *testing.T) {
	w := &BackupWorker{Cfg: &config.Config{}}
	// Should not panic when Channels is nil.
	w.notifyFailure(context.Background(), errors.New("test error"))
}

func TestBackupWorker_notifyFailure_EmptyBotToken(t *testing.T) {
	w := &BackupWorker{
		Cfg: &config.Config{
			Channels: &config.ChannelsConfig{
				Telegram: &config.TelegramConfig{BotToken: "", OwnerID: 0},
			},
		},
	}
	// Should return early when BotToken is empty.
	w.notifyFailure(context.Background(), errors.New("test error"))
}
