package openrouter

import (
	"testing"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
)

func TestOpenRouterFactoryRegistered(t *testing.T) {
	f, ok := provider.GetFactory("openrouter")
	if !ok {
		t.Fatal("openrouter factory not registered")
	}
	p := f(config.ModelProviderConfig{}, "test-key")
	if p == nil {
		t.Fatal("factory returned nil provider")
	}
}

func TestOpenRouterFactoryCustomBaseURL(t *testing.T) {
	f, _ := provider.GetFactory("openrouter")
	p := f(config.ModelProviderConfig{BaseURL: "https://custom.openrouter.example"}, "test-key")
	if p == nil {
		t.Fatal("factory returned nil provider with custom base URL")
	}
}

func TestOpenRouterEnvVar(t *testing.T) {
	envVar, ok := provider.ProviderEnvVars["openrouter"]
	if !ok {
		t.Fatal("openrouter not in ProviderEnvVars")
	}
	if envVar != "OPENROUTER_API_KEY" {
		t.Errorf("env var = %q, want OPENROUTER_API_KEY", envVar)
	}
}

func TestOpenRouterNestedModelSpec(t *testing.T) {
	// OpenRouter model specs have nested slashes: "openrouter/anthropic/claude-sonnet-4-6"
	// ParseModelSpec with SplitN(spec, "/", 2) should handle this correctly.
	prov, model, err := provider.ParseModelSpec("openrouter/anthropic/claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("ParseModelSpec: %v", err)
	}
	if prov != "openrouter" {
		t.Errorf("provider = %q, want openrouter", prov)
	}
	if model != "anthropic/claude-sonnet-4-6" {
		t.Errorf("model = %q, want anthropic/claude-sonnet-4-6", model)
	}
}
