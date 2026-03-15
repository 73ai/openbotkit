package provider

import (
	"context"
	"testing"

	"github.com/priyanshujain/openbotkit/config"
)

// mockProvider implements Provider for testing the router.
type mockProvider struct {
	lastModel string
}

func (m *mockProvider) Chat(_ context.Context, req ChatRequest) (*ChatResponse, error) {
	m.lastModel = req.Model
	return &ChatResponse{
		Content:    []ContentBlock{{Type: ContentText, Text: "ok"}},
		StopReason: StopEndTurn,
	}, nil
}

func (m *mockProvider) StreamChat(_ context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	m.lastModel = req.Model
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventDone}
	close(ch)
	return ch, nil
}

func TestRouter_DefaultTier(t *testing.T) {
	mp := &mockProvider{}
	r := &Registry{providers: map[string]Provider{"anthropic": mp}}
	models := &config.ModelsConfig{Default: "anthropic/claude-sonnet-4-6"}

	router := NewRouter(r, models)
	resp, err := router.Chat(context.Background(), TierDefault, ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if mp.lastModel != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want %q", mp.lastModel, "claude-sonnet-4-6")
	}
	if resp.TextContent() != "ok" {
		t.Errorf("text = %q", resp.TextContent())
	}
}

func TestRouter_ComplexFallsBackToDefault(t *testing.T) {
	mp := &mockProvider{}
	r := &Registry{providers: map[string]Provider{"anthropic": mp}}
	models := &config.ModelsConfig{Default: "anthropic/claude-sonnet-4-6"}

	router := NewRouter(r, models)
	_, err := router.Chat(context.Background(), TierComplex, ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if mp.lastModel != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want default fallback", mp.lastModel)
	}
}

func TestRouter_FastTier(t *testing.T) {
	mp := &mockProvider{}
	r := &Registry{providers: map[string]Provider{"openai": mp}}
	models := &config.ModelsConfig{
		Default: "openai/gpt-4o",
		Fast:    "openai/gpt-4o-mini",
	}

	router := NewRouter(r, models)
	_, err := router.Chat(context.Background(), TierFast, ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if mp.lastModel != "gpt-4o-mini" {
		t.Errorf("model = %q, want %q", mp.lastModel, "gpt-4o-mini")
	}
}

func TestRouter_NoModelConfigured(t *testing.T) {
	r := &Registry{providers: map[string]Provider{}}
	models := &config.ModelsConfig{}

	router := NewRouter(r, models)
	_, err := router.Chat(context.Background(), TierDefault, ChatRequest{})
	if err == nil {
		t.Fatal("expected error when no model configured")
	}
}

func TestRouter_NanoTier(t *testing.T) {
	mp := &mockProvider{}
	r := &Registry{providers: map[string]Provider{"groq": mp}}
	models := &config.ModelsConfig{
		Default: "groq/llama-3.3-70b-versatile",
		Nano:    "groq/llama-3.1-8b-instant",
	}

	router := NewRouter(r, models)
	_, err := router.Chat(context.Background(), TierNano, ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if mp.lastModel != "llama-3.1-8b-instant" {
		t.Errorf("model = %q, want %q", mp.lastModel, "llama-3.1-8b-instant")
	}
}

func TestRouter_NanoCascadesToFast(t *testing.T) {
	mp := &mockProvider{}
	r := &Registry{providers: map[string]Provider{"openai": mp}}
	models := &config.ModelsConfig{
		Default: "openai/gpt-4o",
		Fast:    "openai/gpt-4o-mini",
	}

	router := NewRouter(r, models)
	_, err := router.Chat(context.Background(), TierNano, ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if mp.lastModel != "gpt-4o-mini" {
		t.Errorf("model = %q, want fast fallback %q", mp.lastModel, "gpt-4o-mini")
	}
}

func TestRouter_NanoCascadesToDefault(t *testing.T) {
	mp := &mockProvider{}
	r := &Registry{providers: map[string]Provider{"anthropic": mp}}
	models := &config.ModelsConfig{
		Default: "anthropic/claude-sonnet-4-6",
	}

	router := NewRouter(r, models)
	_, err := router.Chat(context.Background(), TierNano, ChatRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if mp.lastModel != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want default fallback %q", mp.lastModel, "claude-sonnet-4-6")
	}
}

func TestRouter_ProviderNotInRegistry(t *testing.T) {
	r := &Registry{providers: map[string]Provider{}}
	models := &config.ModelsConfig{Default: "anthropic/claude-sonnet-4-6"}

	router := NewRouter(r, models)
	_, err := router.Chat(context.Background(), TierDefault, ChatRequest{})
	if err == nil {
		t.Fatal("expected error when provider not in registry")
	}
}

func TestRouter_StreamChat(t *testing.T) {
	mp := &mockProvider{}
	r := &Registry{providers: map[string]Provider{"anthropic": mp}}
	models := &config.ModelsConfig{Default: "anthropic/claude-sonnet-4-6"}

	router := NewRouter(r, models)
	ch, err := router.StreamChat(context.Background(), TierDefault, ChatRequest{})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var gotDone bool
	for event := range ch {
		if event.Type == EventDone {
			gotDone = true
		}
	}
	if !gotDone {
		t.Error("expected Done event")
	}
}
