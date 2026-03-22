package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/riverqueue/river"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/oauth/google"
	"github.com/73ai/openbotkit/provider"
	backupsvc "github.com/73ai/openbotkit/service/backup"
)

type BackupArgs struct{}

func (BackupArgs) Kind() string { return "backup" }

type BackupWorker struct {
	river.WorkerDefaults[BackupArgs]
	Cfg *config.Config
}

func (w *BackupWorker) Work(ctx context.Context, job *river.Job[BackupArgs]) error {
	if w.Cfg.Backup == nil || !w.Cfg.Backup.Enabled {
		slog.Info("backup: not enabled, skipping")
		return nil
	}

	if !config.IsSourceLinked("backup") {
		slog.Info("backup: not linked, skipping")
		return nil
	}

	slog.Info("starting backup job")

	backend, err := backupsvc.ResolveBackend(ctx, backendOpts(w.Cfg))
	if err != nil {
		return fmt.Errorf("resolve backend: %w", err)
	}

	svc := backupsvc.New(backend, config.Dir())
	result, err := svc.Run(ctx)
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	slog.Info("backup complete",
		"changed", result.Changed,
		"skipped", result.Skipped,
		"uploaded", result.Uploaded,
		"duration", result.Duration,
	)
	return nil
}

func backendOpts(cfg *config.Config) backupsvc.ResolveBackendOpts {
	opts := backupsvc.ResolveBackendOpts{
		ResolveCred: provider.ResolveAPIKey,
		BackupDest:  cfg.Backup.Destination,
		GoogleClient: func(ctx context.Context, gcfg backupsvc.GoogleClientConfig) (*http.Client, error) {
			gp := google.New(google.Config{
				CredentialsFile: cfg.GoogleCredentialsFile(),
				TokenDBPath:     cfg.GoogleTokenDBPath(),
			})
			accounts, err := gp.Accounts(ctx)
			if err != nil || len(accounts) == 0 {
				return nil, fmt.Errorf("no Google account found")
			}
			return gp.Client(ctx, accounts[0], gcfg.Scopes)
		},
	}
	if cfg.Backup.R2 != nil {
		opts.R2Bucket = cfg.Backup.R2.Bucket
		opts.R2Endpoint = cfg.Backup.R2.Endpoint
		opts.R2AccessRef = cfg.Backup.R2.AccessKeyRef
		opts.R2SecretRef = cfg.Backup.R2.SecretKeyRef
	}
	if cfg.Backup.GDrive != nil {
		opts.GDriveFolderID = cfg.Backup.GDrive.FolderID
	}
	return opts
}

var _ river.Worker[BackupArgs] = (*BackupWorker)(nil)
