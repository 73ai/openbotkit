package backup

import (
	"context"
	"fmt"
	"net/http"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/oauth/google"
	"github.com/73ai/openbotkit/provider"
	backupsvc "github.com/73ai/openbotkit/service/backup"
)

func resolveBackend(ctx context.Context, cfg *config.Config) (backupsvc.Backend, error) {
	if cfg.Backup == nil || !cfg.Backup.Enabled {
		return nil, fmt.Errorf("backup not configured — run 'obk setup' and select Backup")
	}

	return backupsvc.ResolveBackend(ctx, backendOpts(cfg))
}

func backendOpts(cfg *config.Config) backupsvc.ResolveBackendOpts {
	opts := backupsvc.ResolveBackendOpts{
		ResolveCred:  provider.ResolveAPIKey,
		BackupDest:   cfg.Backup.Destination,
		GoogleClient: makeGoogleClient(cfg),
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

func makeGoogleClient(cfg *config.Config) backupsvc.GoogleClientFactory {
	return func(ctx context.Context, gcfg backupsvc.GoogleClientConfig) (*http.Client, error) {
		gp := google.New(google.Config{
			CredentialsFile: cfg.GoogleCredentialsFile(),
			TokenDBPath:     cfg.GoogleTokenDBPath(),
		})
		accounts, err := gp.Accounts(ctx)
		if err != nil || len(accounts) == 0 {
			return nil, fmt.Errorf("no Google account found — run 'obk setup'")
		}
		return gp.Client(ctx, accounts[0], gcfg.Scopes)
	}
}
