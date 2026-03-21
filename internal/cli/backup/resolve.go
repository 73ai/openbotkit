package backup

import (
	"context"
	"fmt"
	"os"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/oauth/google"
	"github.com/73ai/openbotkit/provider"
	backupsvc "github.com/73ai/openbotkit/service/backup"
)

func resolveBackend(ctx context.Context, cfg *config.Config) (backupsvc.Backend, error) {
	if cfg.Backup == nil || !cfg.Backup.Enabled {
		return nil, fmt.Errorf("backup not configured — run 'obk setup' and select Backup")
	}

	switch cfg.Backup.Destination {
	case "r2":
		return resolveR2Backend(ctx, cfg)
	case "gdrive":
		return resolveGDriveBackend(ctx, cfg)
	default:
		return nil, fmt.Errorf("unknown backup destination: %q", cfg.Backup.Destination)
	}
}

func resolveR2Backend(_ context.Context, cfg *config.Config) (backupsvc.Backend, error) {
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
}

func resolveGDriveBackend(ctx context.Context, cfg *config.Config) (backupsvc.Backend, error) {
	gdrive := cfg.Backup.GDrive
	if gdrive == nil || gdrive.FolderID == "" {
		return nil, fmt.Errorf("Google Drive config missing — run 'obk setup' and select Backup")
	}

	gp := google.New(google.Config{
		CredentialsFile: cfg.GoogleCredentialsFile(),
		TokenDBPath:     cfg.GoogleTokenDBPath(),
	})

	accounts, err := gp.Accounts(ctx)
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("no Google account found — run 'obk setup'")
	}

	httpClient, err := gp.Client(ctx, accounts[0], []string{"https://www.googleapis.com/auth/drive.file"})
	if err != nil {
		return nil, fmt.Errorf("get Drive client: %w", err)
	}

	return backupsvc.NewGDriveBackend(ctx, httpClient, gdrive.FolderID)
}
