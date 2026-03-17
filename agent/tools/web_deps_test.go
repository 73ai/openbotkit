package tools

import (
	"context"
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
)

type stubProvider struct {
	name string
}

func (s *stubProvider) Chat(_ context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "ok"}},
		StopReason: provider.StopEndTurn,
	}, nil
}

func (s *stubProvider) StreamChat(_ context.Context, req provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent, 1)
	ch <- provider.StreamEvent{Type: provider.EventDone}
	close(ch)
	return ch, nil
}

func stubRegistry(providers map[string]provider.Provider) *provider.Registry {
	return provider.NewRegistryFromProviders(providers)
}

func TestResolveFastProvider_Configured(t *testing.T) {
	fastP := &stubProvider{name: "fast"}
	defaultP := &stubProvider{name: "default"}
	reg := stubRegistry(map[string]provider.Provider{"openai": fastP, "anthropic": defaultP})
	models := &config.ModelsConfig{
		Default: "anthropic/claude-sonnet-4-6",
		Fast:    "openai/gpt-4o-mini",
	}

	p, model := ResolveFastProvider(models, reg, defaultP, "claude-sonnet-4-6")
	if p != fastP {
		t.Error("expected fast provider, got default")
	}
	if model != "gpt-4o-mini" {
		t.Errorf("model = %q, want gpt-4o-mini", model)
	}
}

func TestResolveFastProvider_FallsBackToDefault(t *testing.T) {
	defaultP := &stubProvider{name: "default"}
	reg := stubRegistry(map[string]provider.Provider{"anthropic": defaultP})
	models := &config.ModelsConfig{
		Default: "anthropic/claude-sonnet-4-6",
	}

	p, model := ResolveFastProvider(models, reg, defaultP, "claude-sonnet-4-6")
	if p != defaultP {
		t.Error("expected default provider as fallback")
	}
	if model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", model)
	}
}

func TestResolveNanoProvider_Configured(t *testing.T) {
	nanoP := &stubProvider{name: "nano"}
	defaultP := &stubProvider{name: "default"}
	reg := stubRegistry(map[string]provider.Provider{"groq": nanoP, "anthropic": defaultP})
	models := &config.ModelsConfig{
		Default: "anthropic/claude-sonnet-4-6",
		Nano:    "groq/llama-3.1-8b-instant",
	}

	p, model := ResolveNanoProvider(models, reg, defaultP, "claude-sonnet-4-6")
	if p != nanoP {
		t.Error("expected nano provider, got something else")
	}
	if model != "llama-3.1-8b-instant" {
		t.Errorf("model = %q, want llama-3.1-8b-instant", model)
	}
}

func TestResolveNanoProvider_CascadesToFast(t *testing.T) {
	fastP := &stubProvider{name: "fast"}
	defaultP := &stubProvider{name: "default"}
	reg := stubRegistry(map[string]provider.Provider{"openai": fastP, "anthropic": defaultP})
	models := &config.ModelsConfig{
		Default: "anthropic/claude-sonnet-4-6",
		Fast:    "openai/gpt-4o-mini",
	}

	p, model := ResolveNanoProvider(models, reg, defaultP, "claude-sonnet-4-6")
	if p != fastP {
		t.Error("expected fast provider as fallback, got something else")
	}
	if model != "gpt-4o-mini" {
		t.Errorf("model = %q, want gpt-4o-mini", model)
	}
}

func TestResolveNanoProvider_CascadesToDefault(t *testing.T) {
	defaultP := &stubProvider{name: "default"}
	reg := stubRegistry(map[string]provider.Provider{"anthropic": defaultP})
	models := &config.ModelsConfig{
		Default: "anthropic/claude-sonnet-4-6",
	}

	p, model := ResolveNanoProvider(models, reg, defaultP, "claude-sonnet-4-6")
	if p != defaultP {
		t.Error("expected default provider as final fallback")
	}
	if model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", model)
	}
}

func TestResolveNanoProvider_NilModels(t *testing.T) {
	defaultP := &stubProvider{name: "default"}
	reg := stubRegistry(map[string]provider.Provider{})

	p, model := ResolveNanoProvider(nil, reg, defaultP, "fallback-model")
	if p != defaultP {
		t.Error("expected default provider when models is nil")
	}
	if model != "fallback-model" {
		t.Errorf("model = %q, want fallback-model", model)
	}
}
