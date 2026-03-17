package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/73ai/openbotkit/config"

	"github.com/73ai/openbotkit/oauth/google"
	gmailsrc "github.com/73ai/openbotkit/source/gmail"
	"github.com/73ai/openbotkit/store"
)

type GmailSyncArgs struct{}

func (GmailSyncArgs) Kind() string { return "gmail_sync" }

type GmailSyncWorker struct {
	river.WorkerDefaults[GmailSyncArgs]
	Cfg *config.Config
}

func (w *GmailSyncWorker) Work(ctx context.Context, job *river.Job[GmailSyncArgs]) error {
	if !config.IsSourceLinked("gmail") {
		slog.Info("gmail: not linked, skipping sync")
		return nil
	}

	slog.Info("starting gmail sync job")

	db, err := store.Open(store.Config{
		Driver: w.Cfg.Gmail.Storage.Driver,
		DSN:    w.Cfg.GmailDataDSN(),
	})
	if err != nil {
		return fmt.Errorf("open gmail db: %w", err)
	}
	defer db.Close()

	gp := google.New(google.Config{
		CredentialsFile: w.Cfg.GoogleCredentialsFile(),
		TokenDBPath:     w.Cfg.GoogleTokenDBPath(),
	})
	g := gmailsrc.New(gmailsrc.Config{Provider: gp})

	result, err := g.Sync(ctx, db, gmailsrc.SyncOptions{
		DownloadAttachments: w.Cfg.Gmail.DownloadAttachments,
		AttachmentsDir:      config.SourceDir("gmail"),
	})
	if err != nil {
		return fmt.Errorf("gmail sync: %w", err)
	}

	slog.Info("gmail sync complete", "fetched", result.Fetched, "skipped", result.Skipped, "errors", result.Errors)
	return nil
}

var _ river.Worker[GmailSyncArgs] = (*GmailSyncWorker)(nil)
