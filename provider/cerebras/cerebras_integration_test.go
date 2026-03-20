package cerebras_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	_ "github.com/73ai/openbotkit/provider/cerebras"
)

func TestCerebrasIntegration_ListModels(t *testing.T) {
	apiKey := os.Getenv("CEREBRAS_API_KEY")
	if apiKey == "" {
		t.Skip("CEREBRAS_API_KEY not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	models, err := provider.ListModels(ctx, "cerebras", apiKey, config.ModelProviderConfig{})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected at least one model from Cerebras")
	}
	t.Logf("found %d models:", len(models))
	for _, m := range models {
		t.Logf("  %s (provider=%s)", m.ID, m.Provider)
	}
}

func TestCerebrasIntegration_Chat(t *testing.T) {
	apiKey := os.Getenv("CEREBRAS_API_KEY")
	if apiKey == "" {
		t.Skip("CEREBRAS_API_KEY not set")
	}

	factory, ok := provider.GetFactory("cerebras")
	if !ok {
		t.Fatal("cerebras factory not registered")
	}

	p := factory(config.ModelProviderConfig{}, apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := p.Chat(ctx, provider.ChatRequest{
		Model:     "llama3.1-8b",
		System:    "Reply with exactly one word.",
		Messages:  []provider.Message{provider.NewTextMessage(provider.RoleUser, "Say hello")},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	text := resp.TextContent()
	if text == "" {
		t.Fatal("expected non-empty response")
	}
	t.Logf("response: %q (input=%d, output=%d)", text, resp.Usage.InputTokens, resp.Usage.OutputTokens)
}
