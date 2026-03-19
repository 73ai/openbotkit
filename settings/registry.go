package settings

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/73ai/openbotkit/config"
)

func BuildTree(svc *Service) []Node {
	return []Node{
		{Category: generalCategory()},
		{Category: modelsCategory(svc)},
		{Category: channelsCategory()},
		{Category: dataSourcesCategory()},
		{Category: integrationsCategory()},
		{Category: advancedCategory()},
	}
}

func generalCategory() *Category {
	return &Category{
		Key:   "general",
		Label: "General",
		Children: []Node{
			{Field: &Field{
				Key:   "mode",
				Label: "Mode",
				Type:  TypeSelect,
				Options: []Option{
					{"Local", string(config.ModeLocal)},
					{"Remote", string(config.ModeRemote)},
					{"Server", string(config.ModeServer)},
				},
				Get: func(c *config.Config) string {
					return string(c.ResolvedMode())
				},
				Set: func(c *config.Config, v string) error {
					c.Mode = config.Mode(v)
					return nil
				},
			}},
			{Field: &Field{
				Key:         "timezone",
				Label:       "Timezone",
				Description: "IANA timezone (e.g. America/New_York)",
				Type:        TypeString,
				Get: func(c *config.Config) string {
					return c.Timezone
				},
				Set: func(c *config.Config, v string) error {
					c.Timezone = v
					return nil
				},
				Validate: func(v string) error {
					if v == "" {
						return nil
					}
					_, err := time.LoadLocation(v)
					if err != nil {
						return fmt.Errorf("invalid timezone: %w", err)
					}
					return nil
				},
			}},
		},
	}
}

func modelsCategory(svc *Service) *Category {
	children := []Node{
		{Field: profileField(svc)},
		{Field: modelTierDisplay("models.default", "Default Model", func(c *config.Config) string {
			if c.Models == nil {
				return ""
			}
			return c.Models.Default
		})},
		{Field: modelTierDisplay("models.complex", "Complex Model", func(c *config.Config) string {
			if c.Models == nil {
				return ""
			}
			return c.Models.Complex
		})},
		{Field: modelTierDisplay("models.fast", "Fast Model", func(c *config.Config) string {
			if c.Models == nil {
				return ""
			}
			return c.Models.Fast
		})},
		{Field: modelTierDisplay("models.nano", "Nano Model", func(c *config.Config) string {
			if c.Models == nil {
				return ""
			}
			return c.Models.Nano
		})},
		{Field: &Field{
			Key:   "models.context_window",
			Label: "Context Window",
			Type:  TypeNumber,
			Get: func(c *config.Config) string {
				if c.Models == nil || c.Models.ContextWindow == 0 {
					return ""
				}
				return strconv.Itoa(c.Models.ContextWindow)
			},
			Set: func(c *config.Config, v string) error {
				ensureModels(c)
				if v == "" {
					c.Models.ContextWindow = 0
					return nil
				}
				n, err := strconv.Atoi(v)
				if err != nil {
					return fmt.Errorf("invalid number: %w", err)
				}
				c.Models.ContextWindow = n
				return nil
			},
			Validate: validateNumber,
		}},
		{Field: &Field{
			Key:         "models.compaction_threshold",
			Label:       "Compaction Threshold",
			Description: "0.0–1.0",
			Type:        TypeNumber,
			Get: func(c *config.Config) string {
				if c.Models == nil || c.Models.CompactionThreshold == 0 {
					return ""
				}
				return strconv.FormatFloat(c.Models.CompactionThreshold, 'f', -1, 64)
			},
			Set: func(c *config.Config, v string) error {
				ensureModels(c)
				if v == "" {
					c.Models.CompactionThreshold = 0
					return nil
				}
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return fmt.Errorf("invalid number: %w", err)
				}
				c.Models.CompactionThreshold = f
				return nil
			},
		}},
		{Category: providersCategory(svc)},
	}

	return &Category{
		Key:      "models",
		Label:    "LLM Models",
		Children: children,
	}
}

func profileField(svc *Service) *Field {
	return &Field{
		Key:   "models.profile",
		Label: "LLM Cost Profile",
		Type:  TypeString,
		Get: func(c *config.Config) string {
			if c.Models == nil || c.Models.Profile == "" {
				if c.Models != nil && c.Models.Default != "" {
					return "(custom)"
				}
				return "(not configured)"
			}
			name := c.Models.Profile
			if p, ok := config.Profiles[name]; ok {
				return p.Label
			}
			if c.Models.CustomProfiles != nil {
				if cp, ok := c.Models.CustomProfiles[name]; ok {
					label := cp.Label
					if label == "" {
						label = name
					}
					return label + " (custom)"
				}
			}
			return name
		},
		Set:      func(c *config.Config, v string) error { return nil },
		EditFunc: profileWizard,
	}
}

// modelTierDisplay creates a display-only field for a model tier.
// Editable only when profile is custom (no fixed profile set).
func modelTierDisplay(key, label string, getter func(*config.Config) string) *Field {
	return &Field{
		Key:   key,
		Label: label,
		Type:  TypeSelect,
		Get:   getter,
		Set: func(c *config.Config, v string) error {
			ensureModels(c)
			switch key {
			case "models.default":
				c.Models.Default = v
			case "models.complex":
				c.Models.Complex = v
			case "models.fast":
				c.Models.Fast = v
			case "models.nano":
				c.Models.Nano = v
			}
			return nil
		},
		OptionsFunc: func(c *config.Config) []Option {
			return modelOptionsForTier(c, tierFromKey(key))
		},
		ReadOnly: func(c *config.Config) bool {
			if c.Models == nil || c.Models.Profile == "" {
				return false
			}
			if _, ok := config.Profiles[c.Models.Profile]; ok {
				return true
			}
			return false
		},
	}
}

func tierFromKey(key string) string {
	switch key {
	case "models.default":
		return "default"
	case "models.complex":
		return "complex"
	case "models.fast":
		return "fast"
	case "models.nano":
		return "nano"
	}
	return ""
}

// profileWizard runs the multi-step profile configuration flow.
func profileWizard(svc *Service) (string, error) {
	cfg := svc.cfg

	// Step 1: Select profile.
	var profileName string
	profileOptions := buildProfileSelectOptions()

	if cfg.Models != nil && cfg.Models.Profile != "" {
		profileName = cfg.Models.Profile
	}

	err := runProfileSelect(&profileName, profileOptions)
	if err != nil {
		return "", err
	}

	if profileName == "custom" {
		return setupCustomProfile(svc)
	}

	return setupFixedProfile(svc, profileName)
}

func buildProfileSelectOptions() []Option {
	var opts []Option
	for _, name := range config.ProfileNames {
		p := config.Profiles[name]
		opts = append(opts, Option{p.Label + " — " + p.Description, name})
	}
	opts = append(opts, Option{"Custom (choose models manually)", "custom"})
	return opts
}

// ProfilePreview returns a human-readable preview for a profile name.
func ProfilePreview(name string) string {
	p, ok := config.Profiles[name]
	if !ok {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Profile: %s\n", p.Label)
	fmt.Fprintf(&b, "  Default: %s\n", p.Tiers.Default)
	fmt.Fprintf(&b, "  Complex: %s\n", p.Tiers.Complex)
	fmt.Fprintf(&b, "  Fast:    %s\n", p.Tiers.Fast)
	fmt.Fprintf(&b, "  Nano:    %s\n", p.Tiers.Nano)
	fmt.Fprintf(&b, "  Providers: %s", strings.Join(p.Providers, ", "))
	return b.String()
}

func setupFixedProfile(svc *Service, profileName string) (string, error) {
	p, ok := config.Profiles[profileName]
	if !ok {
		return "", fmt.Errorf("unknown profile %q", profileName)
	}

	fmt.Printf("\n%s\n\n", ProfilePreview(profileName))

	// Configure required providers.
	for _, provName := range p.Providers {
		if err := configureProviderAuth(svc, provName); err != nil {
			return "", err
		}
	}

	// Verify all providers.
	fmt.Println("\n  Verifying providers...")
	for _, provName := range p.Providers {
		pcfg := providerConfig(svc.cfg, provName)
		if err := svc.VerifyProvider(provName, pcfg); err != nil {
			return "", fmt.Errorf("  %s: verification failed: %w\n\n  Profile not saved. Fix provider auth and try again", provName, err)
		}
		fmt.Printf("  %s: verified\n", provName)
	}

	// All verified — save.
	ensureModels(svc.cfg)
	svc.cfg.Models.Profile = profileName
	svc.cfg.Models.Default = p.Tiers.Default
	svc.cfg.Models.Complex = p.Tiers.Complex
	svc.cfg.Models.Fast = p.Tiers.Fast
	svc.cfg.Models.Nano = p.Tiers.Nano

	if err := svc.saveFn(svc.cfg); err != nil {
		return "", fmt.Errorf("save: %w", err)
	}

	svc.RebuildTree()
	return "Profile saved!", nil
}

func setupCustomProfile(svc *Service) (string, error) {
	cfg := svc.cfg
	configured := configuredProviders(cfg)

	if len(configured) == 0 {
		// Need at least one provider.
		fmt.Println("\n  No providers configured. Set up at least one provider first.")
		provName, err := selectProvider()
		if err != nil {
			return "", err
		}
		if err := configureProviderAuth(svc, provName); err != nil {
			return "", err
		}
		configured = []string{provName}
	}

	// Select models for each tier.
	ensureModels(cfg)
	tiers := []struct {
		label string
		tier  string
		dest  *string
	}{
		{"Default model", "default", &cfg.Models.Default},
		{"Complex model", "complex", &cfg.Models.Complex},
		{"Fast model", "fast", &cfg.Models.Fast},
		{"Nano model", "nano", &cfg.Models.Nano},
	}

	// Collect which providers are actually needed.
	neededProviders := make(map[string]bool)

	for _, td := range tiers {
		opts := modelOptionsForTier(cfg, td.tier)
		selected, err := selectModel(td.label, opts, *td.dest)
		if err != nil {
			return "", err
		}
		*td.dest = selected
		if selected != "" {
			parts := strings.SplitN(selected, "/", 2)
			if len(parts) >= 1 {
				neededProviders[parts[0]] = true
			}
		}
	}

	// Configure auth for any needed providers that aren't configured.
	for provName := range neededProviders {
		pcfg := providerConfig(cfg, provName)
		if pcfg.APIKeyRef == "" && pcfg.AuthMethod != "vertex_ai" {
			fmt.Printf("\n  Provider %q requires authentication.\n", provName)
			if err := configureProviderAuth(svc, provName); err != nil {
				return "", err
			}
		}
	}

	// Verify all needed providers.
	fmt.Println("\n  Verifying providers...")
	for provName := range neededProviders {
		pcfg := providerConfig(cfg, provName)
		if err := svc.VerifyProvider(provName, pcfg); err != nil {
			return "", fmt.Errorf("  %s: verification failed: %w\n\n  Profile not saved. Fix provider auth and try again", provName, err)
		}
		fmt.Printf("  %s: verified\n", provName)
	}

	cfg.Models.Profile = ""
	if err := svc.saveFn(cfg); err != nil {
		return "", fmt.Errorf("save: %w", err)
	}

	svc.RebuildTree()
	return "Custom profile saved!", nil
}

// configureProviderAuth prompts for API key if not already configured.
func configureProviderAuth(svc *Service, provName string) error {
	pcfg := providerConfig(svc.cfg, provName)

	if pcfg.APIKeyRef != "" {
		masked := maskCredential(svc, pcfg.APIKeyRef)
		fmt.Printf("  %s: %s (leave blank to keep)\n", provName, masked)
	} else {
		fmt.Printf("  %s: not configured\n", provName)
	}

	var apiKey string
	apiKey, err := promptAPIKey(provName, pcfg.APIKeyRef != "")
	if err != nil {
		return err
	}

	if apiKey == "" && pcfg.APIKeyRef != "" {
		return nil
	}
	if apiKey == "" {
		return fmt.Errorf("%s API key is required", provName)
	}

	ref := fmt.Sprintf("keychain:obk/%s", provName)
	if err := svc.StoreCredential(ref, apiKey); err != nil {
		return fmt.Errorf("store credential: %w", err)
	}

	ensureModels(svc.cfg)
	if svc.cfg.Models.Providers == nil {
		svc.cfg.Models.Providers = make(map[string]config.ModelProviderConfig)
	}
	pc := svc.cfg.Models.Providers[provName]
	pc.APIKeyRef = ref
	svc.cfg.Models.Providers[provName] = pc
	return nil
}

// maskCredential loads a credential and returns a masked version like "sk-ant...4x2f".
func maskCredential(svc *Service, ref string) string {
	if svc.loadCred == nil {
		return "(configured)"
	}
	key, err := svc.loadCred(ref)
	if err != nil || key == "" {
		return "(configured)"
	}
	return MaskKey(key)
}

// MaskKey masks an API key showing first 6 and last 4 chars.
func MaskKey(key string) string {
	if len(key) <= 10 {
		return "****"
	}
	return key[:6] + "..." + key[len(key)-4:]
}

func providerConfig(cfg *config.Config, name string) config.ModelProviderConfig {
	if cfg.Models != nil && cfg.Models.Providers != nil {
		return cfg.Models.Providers[name]
	}
	return config.ModelProviderConfig{}
}

// modelOptionsForTier returns select options for a tier based on configured providers.
func modelOptionsForTier(c *config.Config, tier string) []Option {
	configured := configuredProviders(c)
	if len(configured) == 0 {
		return []Option{{"(configure a provider first)", ""}}
	}

	available := config.ModelsForProviders(configured)
	recommended := config.ModelsForTier(available, tier)

	seen := make(map[string]bool)
	opts := []Option{{"(none)", ""}}

	for _, m := range recommended {
		spec := m.Provider + "/" + m.ID
		seen[spec] = true
		opts = append(opts, Option{m.Label + " *", spec})
	}
	for _, m := range available {
		spec := m.Provider + "/" + m.ID
		if !seen[spec] {
			seen[spec] = true
			opts = append(opts, Option{m.Label, spec})
		}
	}
	return opts
}

// configuredProviders returns names of providers that have API keys or vertex_ai auth.
func configuredProviders(c *config.Config) []string {
	if c.Models == nil || c.Models.Providers == nil {
		return nil
	}
	var names []string
	for name, pc := range c.Models.Providers {
		if pc.APIKeyRef != "" || pc.AuthMethod == "vertex_ai" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func providersCategory(svc *Service) *Category {
	type providerDef struct {
		key        string
		label      string
		hasAuth    bool
		authOpts   []Option
		keychainID string
	}

	providers := []providerDef{
		{
			key: "anthropic", label: "Anthropic", hasAuth: true,
			authOpts:   []Option{{"API Key", "api_key"}, {"Vertex AI", "vertex_ai"}},
			keychainID: "obk/anthropic",
		},
		{key: "openai", label: "OpenAI", keychainID: "obk/openai"},
		{
			key: "gemini", label: "Gemini", hasAuth: true,
			authOpts:   []Option{{"API Key", "api_key"}, {"Vertex AI", "vertex_ai"}},
			keychainID: "obk/gemini",
		},
		{key: "groq", label: "Groq", keychainID: "obk/groq"},
		{key: "openrouter", label: "OpenRouter", keychainID: "obk/openrouter"},
	}

	var children []Node
	for _, p := range providers {
		var fields []Node
		ref := "keychain:" + p.keychainID
		provKey := p.key

		fields = append(fields, Node{Field: &Field{
			Key:   "models.providers." + p.key + ".api_key",
			Label: "API Key",
			Type:  TypePassword,
			Get: func(c *config.Config) string {
				if c.Models == nil || c.Models.Providers == nil {
					return "not configured"
				}
				pc, ok := c.Models.Providers[provKey]
				if !ok || pc.APIKeyRef == "" {
					return "not configured"
				}
				return maskCredential(svc, pc.APIKeyRef)
			},
			Set: func(c *config.Config, v string) error {
				if v == "" {
					return nil
				}
				ensureModels(c)
				if c.Models.Providers == nil {
					c.Models.Providers = make(map[string]config.ModelProviderConfig)
				}
				if err := svc.StoreCredential(ref, v); err != nil {
					return fmt.Errorf("store credential: %w", err)
				}
				pc := c.Models.Providers[provKey]
				pc.APIKeyRef = ref
				c.Models.Providers[provKey] = pc
				return nil
			},
			AfterSet: func(s *Service) string {
				if s.cfg.Models == nil || s.cfg.Models.Providers == nil {
					return ""
				}
				pc, ok := s.cfg.Models.Providers[provKey]
				if !ok || pc.APIKeyRef == "" {
					return ""
				}
				err := s.VerifyProvider(provKey, pc)
				if err != nil {
					return fmt.Sprintf("Warning: verification failed: %v", err)
				}
				return "API key verified"
			},
		}})

		if p.hasAuth {
			fields = append(fields, Node{Field: &Field{
				Key:     "models.providers." + p.key + ".auth_method",
				Label:   "Auth Method",
				Type:    TypeSelect,
				Options: p.authOpts,
				Get: func(c *config.Config) string {
					if c.Models == nil || c.Models.Providers == nil {
						return "api_key"
					}
					pc, ok := c.Models.Providers[provKey]
					if !ok || pc.AuthMethod == "" {
						return "api_key"
					}
					return pc.AuthMethod
				},
				Set: func(c *config.Config, v string) error {
					ensureModels(c)
					if c.Models.Providers == nil {
						c.Models.Providers = make(map[string]config.ModelProviderConfig)
					}
					pc := c.Models.Providers[provKey]
					pc.AuthMethod = v
					c.Models.Providers[provKey] = pc
					return nil
				},
			}})
		}

		children = append(children, Node{Category: &Category{
			Key:      "models.providers." + p.key,
			Label:    p.label,
			Children: fields,
		}})
	}

	return &Category{
		Key:      "models.providers",
		Label:    "Providers",
		Children: children,
	}
}

func channelsCategory() *Category {
	return &Category{
		Key:   "channels",
		Label: "Channels",
		Children: []Node{
			{Category: &Category{
				Key:   "channels.telegram",
				Label: "Telegram",
				Children: []Node{
					{Field: &Field{
						Key:   "channels.telegram.bot_token",
						Label: "Bot Token",
						Type:  TypePassword,
						Get: func(c *config.Config) string {
							if c.Channels == nil || c.Channels.Telegram == nil || c.Channels.Telegram.BotToken == "" {
								return "not configured"
							}
							return "configured"
						},
						Set: func(c *config.Config, v string) error {
							if c.Channels == nil {
								c.Channels = &config.ChannelsConfig{}
							}
							if c.Channels.Telegram == nil {
								c.Channels.Telegram = &config.TelegramConfig{}
							}
							c.Channels.Telegram.BotToken = v
							return nil
						},
					}},
					{Field: &Field{
						Key:   "channels.telegram.owner_id",
						Label: "Owner ID",
						Type:  TypeNumber,
						Get: func(c *config.Config) string {
							if c.Channels == nil || c.Channels.Telegram == nil || c.Channels.Telegram.OwnerID == 0 {
								return ""
							}
							return strconv.FormatInt(c.Channels.Telegram.OwnerID, 10)
						},
						Set: func(c *config.Config, v string) error {
							if c.Channels == nil {
								c.Channels = &config.ChannelsConfig{}
							}
							if c.Channels.Telegram == nil {
								c.Channels.Telegram = &config.TelegramConfig{}
							}
							if v == "" {
								c.Channels.Telegram.OwnerID = 0
								return nil
							}
							n, err := strconv.ParseInt(v, 10, 64)
							if err != nil {
								return fmt.Errorf("invalid number: %w", err)
							}
							c.Channels.Telegram.OwnerID = n
							return nil
						},
						Validate: validateNumber,
					}},
				},
			}},
		},
	}
}

func dataSourcesCategory() *Category {
	return &Category{
		Key:   "datasources",
		Label: "Data Sources",
		Children: []Node{
			{Category: &Category{
				Key:   "datasources.gmail",
				Label: "Gmail",
				Children: []Node{
					{Field: &Field{
						Key:   "gmail.sync_days",
						Label: "Sync Days",
						Type:  TypeNumber,
						Get: func(c *config.Config) string {
							if c.Gmail == nil || c.Gmail.SyncDays == 0 {
								return ""
							}
							return strconv.Itoa(c.Gmail.SyncDays)
						},
						Set: func(c *config.Config, v string) error {
							if c.Gmail == nil {
								c.Gmail = &config.GmailConfig{}
							}
							if v == "" {
								c.Gmail.SyncDays = 0
								return nil
							}
							n, err := strconv.Atoi(v)
							if err != nil {
								return fmt.Errorf("invalid number: %w", err)
							}
							c.Gmail.SyncDays = n
							return nil
						},
						Validate: validateNumber,
					}},
					{Field: &Field{
						Key:   "gmail.download_attachments",
						Label: "Download Attachments",
						Type:  TypeBool,
						Get: func(c *config.Config) string {
							if c.Gmail == nil {
								return "false"
							}
							return strconv.FormatBool(c.Gmail.DownloadAttachments)
						},
						Set: func(c *config.Config, v string) error {
							if c.Gmail == nil {
								c.Gmail = &config.GmailConfig{}
							}
							b, err := strconv.ParseBool(v)
							if err != nil {
								return fmt.Errorf("invalid boolean: %w", err)
							}
							c.Gmail.DownloadAttachments = b
							return nil
						},
					}},
				},
			}},
			{Category: &Category{
				Key:   "datasources.websearch",
				Label: "Web Search",
				Children: []Node{
					{Field: &Field{
						Key:   "websearch.proxy",
						Label: "Proxy",
						Type:  TypeString,
						Get: func(c *config.Config) string {
							if c.WebSearch == nil {
								return ""
							}
							return c.WebSearch.Proxy
						},
						Set: func(c *config.Config, v string) error {
							if c.WebSearch == nil {
								c.WebSearch = &config.WebSearchConfig{}
							}
							c.WebSearch.Proxy = v
							return nil
						},
					}},
					{Field: &Field{
						Key:   "websearch.timeout",
						Label: "Timeout",
						Type:  TypeString,
						Get: func(c *config.Config) string {
							if c.WebSearch == nil {
								return ""
							}
							return c.WebSearch.Timeout
						},
						Set: func(c *config.Config, v string) error {
							if c.WebSearch == nil {
								c.WebSearch = &config.WebSearchConfig{}
							}
							c.WebSearch.Timeout = v
							return nil
						},
					}},
					{Field: &Field{
						Key:   "websearch.cache_ttl",
						Label: "Cache TTL",
						Type:  TypeString,
						Get: func(c *config.Config) string {
							if c.WebSearch == nil {
								return ""
							}
							return c.WebSearch.CacheTTL
						},
						Set: func(c *config.Config, v string) error {
							if c.WebSearch == nil {
								c.WebSearch = &config.WebSearchConfig{}
							}
							c.WebSearch.CacheTTL = v
							return nil
						},
					}},
				},
			}},
		},
	}
}

func integrationsCategory() *Category {
	return &Category{
		Key:   "integrations",
		Label: "Integrations",
		Children: []Node{
			{Category: &Category{
				Key:   "integrations.gws",
				Label: "Google Workspace",
				Children: []Node{
					{Field: &Field{
						Key:   "integrations.gws.enabled",
						Label: "Enabled",
						Type:  TypeBool,
						Get: func(c *config.Config) string {
							if c.Integrations == nil || c.Integrations.GWS == nil {
								return "false"
							}
							return strconv.FormatBool(c.Integrations.GWS.Enabled)
						},
						Set: func(c *config.Config, v string) error {
							ensureGWS(c)
							b, err := strconv.ParseBool(v)
							if err != nil {
								return fmt.Errorf("invalid boolean: %w", err)
							}
							c.Integrations.GWS.Enabled = b
							return nil
						},
					}},
					{Field: &Field{
						Key:   "integrations.gws.callback_url",
						Label: "Callback URL",
						Type:  TypeString,
						Get: func(c *config.Config) string {
							if c.Integrations == nil || c.Integrations.GWS == nil {
								return ""
							}
							return c.Integrations.GWS.CallbackURL
						},
						Set: func(c *config.Config, v string) error {
							ensureGWS(c)
							c.Integrations.GWS.CallbackURL = v
							return nil
						},
					}},
					{Field: &Field{
						Key:   "integrations.gws.ngrok_domain",
						Label: "Ngrok Domain",
						Type:  TypeString,
						Get: func(c *config.Config) string {
							if c.Integrations == nil || c.Integrations.GWS == nil {
								return ""
							}
							return c.Integrations.GWS.NgrokDomain
						},
						Set: func(c *config.Config, v string) error {
							ensureGWS(c)
							c.Integrations.GWS.NgrokDomain = v
							return nil
						},
					}},
				},
			}},
		},
	}
}

func advancedCategory() *Category {
	return &Category{
		Key:   "advanced",
		Label: "Advanced",
		Children: []Node{
			{Category: &Category{
				Key:   "advanced.daemon",
				Label: "Daemon",
				Children: []Node{
					{Field: &Field{
						Key:         "daemon.gmail_sync_period",
						Label:       "Gmail Sync Period",
						Description: "e.g. 15m, 1h",
						Type:        TypeString,
						Get: func(c *config.Config) string {
							if c.Daemon == nil {
								return ""
							}
							return c.Daemon.GmailSyncPeriod
						},
						Set: func(c *config.Config, v string) error {
							if c.Daemon == nil {
								c.Daemon = &config.DaemonConfig{}
							}
							c.Daemon.GmailSyncPeriod = v
							return nil
						},
					}},
				},
			}},
		},
	}
}

func ensureModels(c *config.Config) {
	if c.Models == nil {
		c.Models = &config.ModelsConfig{}
	}
}

func ensureGWS(c *config.Config) {
	if c.Integrations == nil {
		c.Integrations = &config.IntegrationsConfig{}
	}
	if c.Integrations.GWS == nil {
		c.Integrations.GWS = &config.GWSConfig{}
	}
}

func validateNumber(v string) error {
	if v == "" {
		return nil
	}
	if _, err := strconv.ParseFloat(v, 64); err != nil {
		return fmt.Errorf("must be a number")
	}
	return nil
}
