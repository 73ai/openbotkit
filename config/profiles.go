package config

import (
	"fmt"
	"regexp"
)

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
	"free": {
		Name:        "free",
		Label:       "Free ($0/mo)",
		Description: "Free tier using Gemini + Cerebras. No credit card required.",
		Category:    "multi",
		Tiers: ProfileTiers{
			Default: "gemini/gemini-2.5-flash",
			Complex: "cerebras/qwen-3-235b-a22b-instruct-2507",
			Fast:    "cerebras/llama3.1-8b",
			Nano:    "gemini/gemini-2.5-flash-lite",
		},
		Providers: []string{"gemini", "cerebras"},
	},
	"gemini": {
		Name:        "gemini",
		Label:       "Gemini (single provider)",
		Description: "Google Gemini models. Free tier available.",
		Category:    "single",
		Tiers: ProfileTiers{
			Default: "gemini/gemini-2.5-flash",
			Complex: "gemini/gemini-2.5-pro",
			Fast:    "gemini/gemini-2.5-flash-lite",
			Nano:    "gemini/gemini-2.5-flash-lite",
		},
		Providers: []string{"gemini"},
	},
	"anthropic": {
		Name:        "anthropic",
		Label:       "Anthropic (single provider)",
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
	"groq": {
		Name:        "groq",
		Label:       "Groq (single provider)",
		Description: "Open-source Llama models with fast inference via Groq.",
		Category:    "single",
		Tiers: ProfileTiers{
			Default: "groq/llama-3.3-70b-versatile",
			Complex: "groq/llama-4-scout-17b-16e",
			Fast:    "groq/llama-3.1-8b-instant",
			Nano:    "groq/llama-3.1-8b-instant",
		},
		Providers: []string{"groq"},
	},
	"openrouter": {
		Name:        "openrouter",
		Label:       "OpenRouter (single provider)",
		Description: "Access 500+ models through OpenRouter.",
		Category:    "single",
		Tiers: ProfileTiers{
			Default: "openrouter/anthropic/claude-haiku-4-5",
			Complex: "openrouter/anthropic/claude-sonnet-4-6",
			Fast:    "openrouter/google/gemini-2.5-flash-lite",
			Nano:    "openrouter/google/gemini-2.5-flash-lite",
		},
		Providers: []string{"openrouter"},
	},
	"zai": {
		Name:        "zai",
		Label:       "Z.AI (single provider)",
		Description: "GLM models from Z.AI. Free tier available.",
		Category:    "single",
		Tiers: ProfileTiers{
			Default: "zai/glm-4.5-flash",
			Complex: "zai/glm-4.7",
			Fast:    "zai/glm-4.5-flash",
			Nano:    "zai/glm-4.5-flash",
		},
		Providers: []string{"zai"},
	},
	"openai": {
		Name:        "openai",
		Label:       "OpenAI (single provider)",
		Description: "GPT models from OpenAI.",
		Category:    "single",
		Tiers: ProfileTiers{
			Default: "openai/gpt-4o-mini",
			Complex: "openai/gpt-4o",
			Fast:    "openai/gpt-4o-mini",
			Nano:    "openai/gpt-4o-mini",
		},
		Providers: []string{"openai"},
	},
	"starter": {
		Name:        "starter",
		Label:       "Starter (~$20/mo)",
		Description: "Good quality, budget-friendly. Mistral for conversation, free nano.",
		Category:    "multi",
		Tiers: ProfileTiers{
			Default: "openrouter/mistralai/mistral-medium-3.1",
			Complex: "openrouter/mistralai/mistral-medium-3.1",
			Fast:    "gemini/gemini-2.5-flash-lite",
			Nano:    "gemini/gemini-2.5-flash-lite",
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
			Fast:    "gemini/gemini-2.5-flash-lite",
			Nano:    "gemini/gemini-2.5-flash-lite",
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
			Nano:    "gemini/gemini-2.5-flash-lite",
		},
		Providers: []string{"openrouter", "gemini"},
	},
}

// ProfileNames returns profile names in display order.
var ProfileNames = []string{
	"free", "starter", "standard", "premium",
	"gemini", "anthropic", "openai", "groq", "openrouter", "zai",
}

var profileNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,29}$`)

// ValidateProfileName checks that a custom profile name is valid.
func ValidateProfileName(name string) error {
	if !profileNameRe.MatchString(name) {
		return fmt.Errorf("profile name must be 2-30 lowercase alphanumeric characters or hyphens, starting with a letter")
	}
	if _, ok := Profiles[name]; ok {
		return fmt.Errorf("profile name %q conflicts with a built-in profile", name)
	}
	if name == "custom" {
		return fmt.Errorf("profile name %q is reserved", name)
	}
	return nil
}
