package settings

import (
	"fmt"
	"sort"
	"strconv"
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
		{Category: providersCategory(svc)},
		{Field: profileField(svc)},
		{Field: modelTierField(svc, "models.default", "Default Model", "default")},
		{Field: modelTierField(svc, "models.complex", "Complex Model", "complex")},
		{Field: modelTierField(svc, "models.fast", "Fast Model", "fast")},
		{Field: modelTierField(svc, "models.nano", "Nano Model", "nano")},
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
	}

	return &Category{
		Key:      "models",
		Label:    "Models",
		Children: children,
	}
}

func profileField(svc *Service) *Field {
	return &Field{
		Key:   "models.profile",
		Label: "Profile",
		Type:  TypeSelect,
		OptionsFunc: func(c *config.Config) []Option {
			configured := configuredProviders(c)
			opts := []Option{{"(none)", ""}}
			for _, name := range config.ProfileNames {
				p := config.Profiles[name]
				if !allProvidersConfigured(p.Providers, configured) {
					continue
				}
				opts = append(opts, Option{p.Label, name})
			}
			if c.Models != nil && len(c.Models.CustomProfiles) > 0 {
				var names []string
				for n := range c.Models.CustomProfiles {
					names = append(names, n)
				}
				sort.Strings(names)
				for _, n := range names {
					cp := c.Models.CustomProfiles[n]
					if !allProvidersConfigured(cp.Providers, configured) {
						continue
					}
					label := cp.Label
					if label == "" {
						label = n
					}
					opts = append(opts, Option{label + " (custom)", n})
				}
			}
			return opts
		},
		Get: func(c *config.Config) string {
			if c.Models == nil {
				return ""
			}
			return c.Models.Profile
		},
		Set: func(c *config.Config, v string) error {
			ensureModels(c)
			c.Models.Profile = v
			return nil
		},
	}
}

func modelTierField(svc *Service, key, label, tier string) *Field {
	return &Field{
		Key:   key,
		Label: label,
		Type:  TypeSelect,
		OptionsFunc: func(c *config.Config) []Option {
			return modelOptionsForTier(c, tier)
		},
		Get: func(c *config.Config) string {
			if c.Models == nil {
				return ""
			}
			switch key {
			case "models.default":
				return c.Models.Default
			case "models.complex":
				return c.Models.Complex
			case "models.fast":
				return c.Models.Fast
			case "models.nano":
				return c.Models.Nano
			}
			return ""
		},
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
	}
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

func allProvidersConfigured(required []string, configured []string) bool {
	set := make(map[string]bool, len(configured))
	for _, c := range configured {
		set[c] = true
	}
	for _, r := range required {
		if !set[r] {
			return false
		}
	}
	return true
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
				if svc.loadCred != nil {
					if _, err := svc.loadCred(pc.APIKeyRef); err == nil {
						return "configured"
					}
				}
				return "configured (ref: " + pc.APIKeyRef + ")"
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
