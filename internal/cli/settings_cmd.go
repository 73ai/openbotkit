package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/73ai/openbotkit/config"
	settingstui "github.com/73ai/openbotkit/internal/settings/tui"
	"github.com/73ai/openbotkit/oauth/google"
	"github.com/73ai/openbotkit/provider"
	_ "github.com/73ai/openbotkit/provider/anthropic"
	_ "github.com/73ai/openbotkit/provider/gemini"
	_ "github.com/73ai/openbotkit/provider/groq"
	_ "github.com/73ai/openbotkit/provider/openai"
	_ "github.com/73ai/openbotkit/provider/openrouter"
	_ "github.com/73ai/openbotkit/provider/zai"
	backupsvc "github.com/73ai/openbotkit/service/backup"
	"github.com/73ai/openbotkit/settings"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Browse and edit settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		svc := settings.New(cfg,
			settings.WithStoreCred(provider.StoreCredential),
			settings.WithLoadCred(provider.LoadCredential),
			settings.WithVerifyProvider(verifyProviderKey),
			settings.WithVerifyBackup(verifyBackupDest),
			settings.WithSetupGDrive(setupGDriveBackup),
			settings.WithTriggerBackup(triggerBackupNow),
		)
		return settingstui.Run(svc)
	},
}

// verifyProviderKey validates the API key by calling the free ListModels API.
func verifyProviderKey(name string, pcfg config.ModelProviderConfig) error {
	var apiKey string
	if pcfg.AuthMethod != "vertex_ai" {
		envVar := provider.ProviderEnvVars[name]
		var err error
		apiKey, err = provider.ResolveAPIKey(pcfg.APIKeyRef, envVar)
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := provider.ListModels(ctx, name, apiKey, pcfg)
	if err != nil {
		return err
	}

	// Cache the results.
	cache := provider.NewModelCache(config.ModelsDir())
	list := &provider.CachedModelList{
		Provider:  name,
		Models:    models,
		FetchedAt: time.Now(),
	}
	// Preserve existing verification data.
	if existing, loadErr := cache.Load(name); loadErr == nil && existing.VerifiedModels != nil {
		list.VerifiedModels = existing.VerifiedModels
	}
	_ = cache.Save(name, list)

	return nil
}

func verifyBackupDest(dest string, cfg *config.Config) error {
	if dest != "r2" {
		return nil
	}
	if cfg.Backup == nil || cfg.Backup.R2 == nil {
		return fmt.Errorf("R2 not configured")
	}
	r2 := cfg.Backup.R2
	accessKey, err := provider.LoadCredential(r2.AccessKeyRef)
	if err != nil {
		return fmt.Errorf("load access key: %w", err)
	}
	secretKey, err := provider.LoadCredential(r2.SecretKeyRef)
	if err != nil {
		return fmt.Errorf("load secret key: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return backupsvc.ValidateR2(ctx, r2.Endpoint, accessKey, secretKey, r2.Bucket)
}

func setupGDriveBackup(cfg *config.Config, folderName string) (string, error) {
	gp := google.New(google.Config{
		CredentialsFile: cfg.GoogleCredentialsFile(),
		TokenDBPath:     cfg.GoogleTokenDBPath(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	accounts, _ := gp.Accounts(ctx)
	var account string
	if len(accounts) > 0 {
		account = accounts[0]
	}

	scopes := []string{"https://www.googleapis.com/auth/drive.file"}
	if _, err := gp.GrantScopes(ctx, account, scopes); err != nil {
		return "", fmt.Errorf("Google auth: %w", err)
	}

	httpClient, err := gp.Client(ctx, account, scopes)
	if err != nil {
		return "", fmt.Errorf("get Drive client: %w", err)
	}

	folderID, err := backupsvc.FindOrCreateDriveFolder(ctx, httpClient, folderName)
	if err != nil {
		return "", fmt.Errorf("create Drive folder: %w", err)
	}

	return folderID, nil
}

func triggerBackupNow(cfg *config.Config) error {
	if cfg.Backup == nil || !cfg.Backup.Enabled {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	opts := backupsvc.ResolveBackendOpts{
		ResolveCred: provider.ResolveAPIKey,
		BackupDest:  cfg.Backup.Destination,
		GoogleClient: func(ctx context.Context, gcfg backupsvc.GoogleClientConfig) (*http.Client, error) {
			gp := google.New(google.Config{
				CredentialsFile: cfg.GoogleCredentialsFile(),
				TokenDBPath:     cfg.GoogleTokenDBPath(),
			})
			accounts, _ := gp.Accounts(ctx)
			if len(accounts) == 0 {
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

	backend, err := backupsvc.ResolveBackend(ctx, opts)
	if err != nil {
		return err
	}

	svc := backupsvc.New(backend, config.Dir())
	_, err = svc.Run(ctx)
	return err
}

func init() {
	rootCmd.AddCommand(settingsCmd)
}
