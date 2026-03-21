package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/73ai/openbotkit/config"
	settingstui "github.com/73ai/openbotkit/internal/settings/tui"
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

func init() {
	rootCmd.AddCommand(settingsCmd)
}
