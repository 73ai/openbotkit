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
	"github.com/73ai/openbotkit/settings"
	"github.com/spf13/cobra"
)

// testModels maps provider name to the cheapest model for verification.
var testModels = map[string]string{
	"anthropic":  "claude-haiku-4-5",
	"openai":     "gpt-4o-mini",
	"gemini":     "gemini-2.0-flash",
	"groq":       "llama-3.1-8b-instant",
	"openrouter": "google/gemini-2.0-flash",
}

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
		)
		return settingstui.Run(svc)
	},
}

func verifyProviderKey(name string, pcfg config.ModelProviderConfig) error {
	factory, ok := provider.GetFactory(name)
	if !ok {
		return fmt.Errorf("unknown provider %q", name)
	}

	var apiKey string
	if pcfg.AuthMethod != "vertex_ai" {
		envVar := provider.ProviderEnvVars[name]
		var err error
		apiKey, err = provider.ResolveAPIKey(pcfg.APIKeyRef, envVar)
		if err != nil {
			return err
		}
	}

	model := testModels[name]
	if model == "" {
		return nil
	}

	p := factory(pcfg, apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := p.Chat(ctx, provider.ChatRequest{
		Model:     model,
		System:    "Reply with OK",
		Messages:  []provider.Message{provider.NewTextMessage(provider.RoleUser, "hi")},
		MaxTokens: 5,
	})
	return err
}

func init() {
	rootCmd.AddCommand(settingsCmd)
}
