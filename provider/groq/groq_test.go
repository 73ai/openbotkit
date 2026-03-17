package groq

import (
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
)

func TestGroqFactoryRegistered(t *testing.T) {
	f, ok := provider.GetFactory("groq")
	if !ok {
		t.Fatal("groq factory not registered")
	}
	p := f(config.ModelProviderConfig{}, "test-key")
	if p == nil {
		t.Fatal("factory returned nil provider")
	}
}

func TestGroqFactoryCustomBaseURL(t *testing.T) {
	f, _ := provider.GetFactory("groq")
	p := f(config.ModelProviderConfig{BaseURL: "https://custom.groq.example"}, "test-key")
	if p == nil {
		t.Fatal("factory returned nil provider with custom base URL")
	}
}

func TestGroqEnvVar(t *testing.T) {
	envVar, ok := provider.ProviderEnvVars["groq"]
	if !ok {
		t.Fatal("groq not in ProviderEnvVars")
	}
	if envVar != "GROQ_API_KEY" {
		t.Errorf("env var = %q, want GROQ_API_KEY", envVar)
	}
}
