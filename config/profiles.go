package config

// ModelProfile defines a preset tier→model mapping based on budget.
type ModelProfile struct {
	Name        string
	Label       string
	Description string
	Category    string // "single" or "multi"
	Tiers       ProfileTiers
	Providers   []string
}

// ProfileTiers maps each tier to a model spec (provider/model).
type ProfileTiers struct {
	Default string `yaml:"default,omitempty"`
	Complex string `yaml:"complex,omitempty"`
	Fast    string `yaml:"fast,omitempty"`
	Nano    string `yaml:"nano,omitempty"`
}

// Profiles contains the built-in model profile presets.
var Profiles = map[string]ModelProfile{
	"gemini": {
		Name:        "gemini",
		Label:       "Gemini (1 API key)",
		Description: "Google Gemini models. Free tier available.",
		Category:    "single",
		Tiers: ProfileTiers{
			Default: "gemini/gemini-2.5-flash",
			Complex: "gemini/gemini-2.5-pro",
			Fast:    "gemini/gemini-2.0-flash-lite",
			Nano:    "gemini/gemini-2.0-flash-lite",
		},
		Providers: []string{"gemini"},
	},
	"anthropic": {
		Name:        "anthropic",
		Label:       "Anthropic (1 API key)",
		Description: "Claude models from Anthropic.",
		Category:    "single",
		Tiers: ProfileTiers{
			Default: "anthropic/claude-haiku-4-5",
			Complex: "anthropic/claude-sonnet-4-6",
			Fast:    "anthropic/claude-haiku-4-5",
			Nano:    "anthropic/claude-haiku-4-5",
		},
		Providers: []string{"anthropic"},
	},
	"starter": {
		Name:        "starter",
		Label:       "Starter (~$20/mo)",
		Description: "Good quality, budget-friendly. Mistral for conversation, free nano.",
		Category:    "multi",
		Tiers: ProfileTiers{
			Default: "openrouter/mistralai/mistral-medium-3.1",
			Complex: "openrouter/mistralai/mistral-medium-3.1",
			Fast:    "gemini/gemini-2.0-flash-lite",
			Nano:    "gemini/gemini-2.0-flash-lite",
		},
		Providers: []string{"openrouter", "gemini"},
	},
	"standard": {
		Name:        "standard",
		Label:       "Standard (~$50/mo)",
		Description: "Strong quality with Claude. Free nano.",
		Category:    "multi",
		Tiers: ProfileTiers{
			Default: "openrouter/anthropic/claude-haiku-4-5",
			Complex: "openrouter/anthropic/claude-sonnet-4-6",
			Fast:    "gemini/gemini-2.0-flash-lite",
			Nano:    "gemini/gemini-2.0-flash-lite",
		},
		Providers: []string{"openrouter", "gemini"},
	},
	"premium": {
		Name:        "premium",
		Label:       "Premium (~$100/mo)",
		Description: "Best quality everywhere. Claude across all tiers.",
		Category:    "multi",
		Tiers: ProfileTiers{
			Default: "openrouter/anthropic/claude-sonnet-4-6",
			Complex: "openrouter/anthropic/claude-opus-4-6",
			Fast:    "openrouter/anthropic/claude-haiku-4-5",
			Nano:    "gemini/gemini-2.0-flash-lite",
		},
		Providers: []string{"openrouter", "gemini"},
	},
}

// ProfileNames returns profile names in display order.
var ProfileNames = []string{"gemini", "anthropic", "starter", "standard", "premium"}
