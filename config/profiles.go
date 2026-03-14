package config

// ModelProfile defines a preset tier→model mapping based on budget.
type ModelProfile struct {
	Name        string
	Label       string
	Description string
	Tiers       ProfileTiers
	Providers   []string
}

// ProfileTiers maps each tier to a model spec (provider/model).
type ProfileTiers struct {
	Default string
	Complex string
	Fast    string
	Nano    string
}

// Profiles contains the built-in model profile presets.
var Profiles = map[string]ModelProfile{
	"starter": {
		Name:        "starter",
		Label:       "Starter (~$20/mo)",
		Description: "Good quality, budget-friendly. Mistral for conversation, free nano.",
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
var ProfileNames = []string{"starter", "standard", "premium"}
