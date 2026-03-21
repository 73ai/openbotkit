package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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

	backend, err := resolveBackupBackend(ctx, w.Cfg)
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

func resolveBackupBackend(ctx context.Context, cfg *config.Config) (backupsvc.Backend, error) {
	switch cfg.Backup.Destination {
	case "r2":
		r2 := cfg.Backup.R2
		if r2 == nil {
			return nil, fmt.Errorf("R2 config missing")
		}
		accessKey, err := provider.ResolveAPIKey(r2.AccessKeyRef, "OBK_R2_ACCESS_KEY")
		if err != nil {
			return nil, fmt.Errorf("resolve R2 access key: %w", err)
		}
		secretKey, err := provider.ResolveAPIKey(r2.SecretKeyRef, "OBK_R2_SECRET_KEY")
		if err != nil {
			return nil, fmt.Errorf("resolve R2 secret key: %w", err)
		}
		endpoint := r2.Endpoint
		if e := os.Getenv("OBK_R2_ENDPOINT"); e != "" {
			endpoint = e
		}
		bucket := r2.Bucket
		if b := os.Getenv("OBK_R2_BUCKET"); b != "" {
			bucket = b
		}
		return backupsvc.NewR2Backend(endpoint, accessKey, secretKey, bucket)

	case "gdrive":
		gdrive := cfg.Backup.GDrive
		if gdrive == nil || gdrive.FolderID == "" {
			return nil, fmt.Errorf("Google Drive config missing")
		}
		gp := google.New(google.Config{
			CredentialsFile: cfg.GoogleCredentialsFile(),
			TokenDBPath:     cfg.GoogleTokenDBPath(),
		})
		accounts, err := gp.Accounts(ctx)
		if err != nil || len(accounts) == 0 {
			return nil, fmt.Errorf("no Google account found")
		}
		httpClient, err := gp.Client(ctx, accounts[0], []string{"https://www.googleapis.com/auth/drive.file"})
		if err != nil {
			return nil, fmt.Errorf("get Drive client: %w", err)
		}
		return backupsvc.NewGDriveBackend(ctx, httpClient, gdrive.FolderID)

	default:
		return nil, fmt.Errorf("unknown backup destination: %q", cfg.Backup.Destination)
	}
}

var _ river.Worker[BackupArgs] = (*BackupWorker)(nil)
