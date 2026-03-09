package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
	_ "github.com/priyanshujain/openbotkit/provider/anthropic"
	_ "github.com/priyanshujain/openbotkit/provider/gemini"
	_ "github.com/priyanshujain/openbotkit/provider/openai"
	"github.com/spf13/cobra"
)

// providerInfo describes a configurable LLM provider.
type providerInfo struct {
	name        string
	label       string
	supportsVAI bool // supports Vertex AI
	models      []huh.Option[string]
}

var llmProviders = []providerInfo{
	{
		name:        "anthropic",
		label:       "Anthropic (Claude)",
		supportsVAI: true,
		models: []huh.Option[string]{
			huh.NewOption("claude-sonnet-4-6 (recommended)", "claude-sonnet-4-6"),
			huh.NewOption("claude-opus-4-6 (most capable)", "claude-opus-4-6"),
			huh.NewOption("claude-haiku-4-5 (fastest)", "claude-haiku-4-5"),
		},
	},
	{
		name:        "openai",
		label:       "OpenAI (GPT)",
		supportsVAI: false,
		models: []huh.Option[string]{
			huh.NewOption("gpt-4o (most capable)", "gpt-4o"),
			huh.NewOption("gpt-4o-mini (faster, cheaper)", "gpt-4o-mini"),
		},
	},
	{
		name:        "gemini",
		label:       "Google Gemini",
		supportsVAI: true,
		models: []huh.Option[string]{
			huh.NewOption("gemini-2.5-pro (most capable)", "gemini-2.5-pro"),
			huh.NewOption("gemini-2.5-flash (fast, good balance)", "gemini-2.5-flash"),
			huh.NewOption("gemini-2.0-flash (fastest, cheapest)", "gemini-2.0-flash"),
		},
	},
}

var vertexRegions = []huh.Option[string]{
	huh.NewOption("us-central1 (Iowa)", "us-central1"),
	huh.NewOption("us-east5 (Columbus)", "us-east5"),
	huh.NewOption("us-east4 (N. Virginia)", "us-east4"),
	huh.NewOption("us-west1 (Oregon)", "us-west1"),
	huh.NewOption("europe-west1 (Belgium)", "europe-west1"),
	huh.NewOption("europe-west4 (Netherlands)", "europe-west4"),
	huh.NewOption("asia-northeast1 (Tokyo)", "asia-northeast1"),
	huh.NewOption("asia-southeast1 (Singapore)", "asia-southeast1"),
}

var setupModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Configure LLM model providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := setupModels(cfg); err != nil {
			return err
		}
		return cfg.Save()
	},
}

func init() {
	setupCmd.AddCommand(setupModelsCmd)
}

func setupModels(cfg *config.Config) error {
	if cfg.Models == nil {
		cfg.Models = &config.ModelsConfig{}
	}
	if cfg.Models.Providers == nil {
		cfg.Models.Providers = make(map[string]config.ModelProviderConfig)
	}

	// Step 1: Provider selection.
	var selectedProviders []string
	var providerOptions []huh.Option[string]
	for _, p := range llmProviders {
		opt := huh.NewOption(p.label, p.name)
		if _, exists := cfg.Models.Providers[p.name]; exists {
			opt = opt.Selected(true)
		}
		providerOptions = append(providerOptions, opt)
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which LLM providers would you like to configure?").
				Options(providerOptions...).
				Value(&selectedProviders),
		),
	).Run()
	if err != nil {
		return err
	}

	if len(selectedProviders) == 0 {
		fmt.Println("  No providers selected.")
		return nil
	}

	// Step 2-4: Configure each provider.
	providerModels := make(map[string]string) // name → selected model
	for _, name := range selectedProviders {
		info := findProvider(name)
		if info == nil {
			continue
		}

		existing := cfg.Models.Providers[name]

		pcfg, model, err := configureProvider(info, existing)
		if err != nil {
			return err
		}
		cfg.Models.Providers[name] = pcfg
		providerModels[name] = model
	}

	// Step 5: Tier configuration.
	if err := configureTiers(cfg, providerModels); err != nil {
		return err
	}

	// Step 6: Validation.
	fmt.Println("\n  Validating providers...")
	for _, name := range selectedProviders {
		pcfg := cfg.Models.Providers[name]
		model := providerModels[name]
		if err := validateProvider(name, pcfg, model); err != nil {
			fmt.Printf("  ✗ %s: %v\n", name, err)
			var retry string
			retryErr := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title(fmt.Sprintf("%s validation failed. What would you like to do?", name)).
						Options(
							huh.NewOption("Skip (keep config anyway)", "skip"),
							huh.NewOption("Re-enter credentials", "retry"),
						).
						Value(&retry),
				),
			).Run()
			if retryErr != nil {
				return retryErr
			}
			if retry == "retry" {
				info := findProvider(name)
				if info != nil {
					pcfg, model, err := configureProvider(info, config.ModelProviderConfig{})
					if err != nil {
						return err
					}
					cfg.Models.Providers[name] = pcfg
					providerModels[name] = model
				}
			}
		} else {
			fmt.Printf("  ✓ %s\n", name)
		}
	}

	fmt.Println("\n  LLM model configuration complete!")
	return nil
}

func configureProvider(info *providerInfo, existing config.ModelProviderConfig) (config.ModelProviderConfig, string, error) {
	pcfg := existing

	// Auth method selection.
	if info.supportsVAI {
		authMethod := existing.AuthMethod
		if authMethod == "" {
			authMethod = "api_key"
		}
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("How would you like to authenticate with %s?", info.label)).
					Options(
						huh.NewOption("API Key", "api_key"),
						huh.NewOption("Google Vertex AI", "vertex_ai"),
					).
					Value(&authMethod),
			),
		).Run()
		if err != nil {
			return pcfg, "", err
		}
		pcfg.AuthMethod = authMethod
	} else {
		pcfg.AuthMethod = "api_key"
	}

	// Auth config.
	if pcfg.AuthMethod == "vertex_ai" {
		if err := configureVertexAI(&pcfg, existing); err != nil {
			return pcfg, "", err
		}
	} else {
		if err := configureAPIKey(&pcfg, info.name, existing); err != nil {
			return pcfg, "", err
		}
	}

	// Model selection.
	model := defaultModelForList(info.models)
	// Try to pre-select from existing tier configs.
	for _, opt := range info.models {
		if opt.Value == existingModelFor(info.name, model) {
			model = opt.Value
			break
		}
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(fmt.Sprintf("Which %s model to use?", info.label)).
				Options(info.models...).
				Value(&model),
		),
	).Run()
	if err != nil {
		return pcfg, "", err
	}

	return pcfg, model, nil
}

func configureAPIKey(pcfg *config.ModelProviderConfig, name string, existing config.ModelProviderConfig) error {
	var apiKey string
	placeholder := ""
	if existing.APIKeyRef != "" {
		placeholder = "(leave blank to keep existing)"
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Enter your %s API key", name)).
				Placeholder(placeholder).
				EchoMode(huh.EchoModePassword).
				Value(&apiKey),
		),
	).Run()
	if err != nil {
		return err
	}

	apiKey = cleanPath(apiKey)
	if apiKey != "" {
		ref := fmt.Sprintf("keychain:obk/%s", name)
		if err := provider.StoreCredential(ref, apiKey); err != nil {
			return fmt.Errorf("store API key: %w", err)
		}
		pcfg.APIKeyRef = ref
		fmt.Printf("  API key stored securely as %s\n", ref)
	}
	return nil
}

func configureVertexAI(pcfg *config.ModelProviderConfig, existing config.ModelProviderConfig) error {
	// Account selection.
	accounts, err := gcloudAccounts()
	if err != nil {
		return fmt.Errorf("list gcloud accounts: %w", err)
	}

	account := existing.VertexAccount
	if len(accounts) > 0 {
		var accountOptions []huh.Option[string]
		for _, a := range accounts {
			accountOptions = append(accountOptions, huh.NewOption(a, a))
		}
		if account == "" {
			account = accounts[0]
		}
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select gcloud account for Vertex AI").
					Description("Account not listed? Run 'gcloud auth login' and restart setup.").
					Options(accountOptions...).
					Value(&account),
			),
		).Run()
	} else {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter gcloud account email for Vertex AI").
					Value(&account),
			),
		).Run()
	}
	if err != nil {
		return err
	}
	pcfg.VertexAccount = account

	// Project selection.
	projects, projErr := gcloudProjects(account)
	if projErr != nil {
		return fmt.Errorf("list gcloud projects: %w", projErr)
	}

	project := existing.VertexProject
	if len(projects) > 0 {
		var projectOptions []huh.Option[string]
		for _, p := range projects {
			projectOptions = append(projectOptions, huh.NewOption(p, p))
		}
		if project == "" {
			project = projects[0]
		}
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select GCP project").
					Options(projectOptions...).
					Value(&project),
			),
		).Run()
	} else {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter GCP project ID").
					Value(&project),
			),
		).Run()
	}
	if err != nil {
		return err
	}
	pcfg.VertexProject = project

	// Region selection.
	region := existing.VertexRegion
	if region == "" {
		region = "us-central1"
	}
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Vertex AI region").
				Options(vertexRegions...).
				Value(&region),
		),
	).Run()
	if err != nil {
		return err
	}
	pcfg.VertexRegion = region

	return nil
}

func configureTiers(cfg *config.Config, providerModels map[string]string) error {
	// Build the list of available model specs.
	var modelOptions []huh.Option[string]
	for name, model := range providerModels {
		spec := name + "/" + model
		modelOptions = append(modelOptions, huh.NewOption(spec, spec))
	}

	if len(modelOptions) == 0 {
		return nil
	}

	// Add "none" option for optional tiers.
	noneOption := huh.NewOption("(none — use default)", "")

	// Default tier.
	defaultSpec := cfg.Models.Default
	if defaultSpec == "" && len(modelOptions) > 0 {
		defaultSpec = modelOptions[0].Value
	}
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select default model (required)").
				Options(modelOptions...).
				Value(&defaultSpec),
		),
	).Run()
	if err != nil {
		return err
	}
	cfg.Models.Default = defaultSpec

	if len(modelOptions) > 1 {
		// Fast tier.
		fastSpec := cfg.Models.Fast
		fastOptions := append([]huh.Option[string]{noneOption}, modelOptions...)
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select fast model (optional)").
					Options(fastOptions...).
					Value(&fastSpec),
			),
		).Run()
		if err != nil {
			return err
		}
		cfg.Models.Fast = fastSpec

		// Complex tier.
		complexSpec := cfg.Models.Complex
		complexOptions := append([]huh.Option[string]{noneOption}, modelOptions...)
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select complex model (optional)").
					Options(complexOptions...).
					Value(&complexSpec),
			),
		).Run()
		if err != nil {
			return err
		}
		cfg.Models.Complex = complexSpec
	}

	return nil
}

func validateProvider(name string, cfg config.ModelProviderConfig, model string) error {
	factory, ok := provider.GetFactory(name)
	if !ok {
		return fmt.Errorf("unknown provider %q", name)
	}

	var apiKey string
	if cfg.AuthMethod != "vertex_ai" {
		envVar := provider.ProviderEnvVars[name]
		var err error
		apiKey, err = provider.ResolveAPIKey(cfg.APIKeyRef, envVar)
		if err != nil {
			return err
		}
	}

	p := factory(cfg, apiKey)
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

func findProvider(name string) *providerInfo {
	for i := range llmProviders {
		if llmProviders[i].name == name {
			return &llmProviders[i]
		}
	}
	return nil
}

func defaultModelForList(models []huh.Option[string]) string {
	if len(models) > 0 {
		return models[0].Value
	}
	return ""
}

// existingModelFor extracts the model from an existing tier spec like "anthropic/claude-sonnet-4-6".
func existingModelFor(providerName, fallback string) string {
	return fallback
}
