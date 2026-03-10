package memory

import (
	"context"
	"os"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/provider/anthropic"
	"github.com/priyanshujain/openbotkit/provider/gemini"
	"github.com/priyanshujain/openbotkit/internal/envload"
	"github.com/priyanshujain/openbotkit/provider/openai"
)

type providerTestCase struct {
	name     string
	provider provider.Provider
	model    string
}

func availableProviders(t *testing.T) []providerTestCase {
	t.Helper()
	envload.Load(t)
	var providers []providerTestCase

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		providers = append(providers, providerTestCase{
			name:     "anthropic",
			provider: anthropic.New(key),
			model:    "claude-sonnet-4-6",
		})
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, providerTestCase{
			name:     "openai",
			provider: openai.New(key),
			model:    "gpt-4o-mini",
		})
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		providers = append(providers, providerTestCase{
			name:     "gemini",
			provider: gemini.New(key),
			model:    "gemini-2.5-flash",
		})
	}
	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		region := os.Getenv("GOOGLE_CLOUD_REGION")
		if region == "" {
			region = "us-east5"
		}
		account := os.Getenv("GOOGLE_CLOUD_ACCOUNT")
		providers = append(providers, providerTestCase{
			name:     "gemini-vertex",
			provider: gemini.New("", gemini.WithVertexAI(project, region), gemini.WithTokenSource(provider.GcloudTokenSource(account))),
			model:    "gemini-2.5-flash",
		})
	}

	if len(providers) == 0 {
		t.Skip("no API keys or Vertex AI config set — skipping integration tests")
	}
	return providers
}

type providerLLM struct {
	p     provider.Provider
	model string
}

func (pl *providerLLM) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	req.Model = pl.model
	return pl.p.Chat(ctx, req)
}

// TestExtract_WithRealLLM verifies fact extraction from conversation messages
// produces valid categorized facts.
func TestExtract_WithRealLLM(t *testing.T) {
	for _, tc := range availableProviders(t) {
		t.Run(tc.name, func(t *testing.T) {
			llm := &providerLLM{p: tc.provider, model: tc.model}

			messages := []string{
				"My name is Alice and I'm a software engineer at TechCorp",
				"I really prefer using Go for backend development over Python",
				"I'm currently building a personal assistant called BotKit",
			}

			facts, err := Extract(context.Background(), llm, messages)
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}

			if len(facts) == 0 {
				t.Fatal("expected at least 1 fact extracted")
			}

			// Verify facts have valid categories.
			validCategories := map[string]bool{
				"identity": true, "preference": true,
				"relationship": true, "project": true,
			}
			for _, f := range facts {
				if f.Content == "" {
					t.Error("fact has empty content")
				}
				if !validCategories[f.Category] {
					t.Errorf("fact has invalid category %q: %q", f.Category, f.Content)
				}
			}
		})
	}
}

// TestExtractAndReconcile_WithRealLLM verifies the full pipeline: extract facts
// from messages, reconcile against existing DB (add new, update changed, skip
// duplicates).
func TestExtractAndReconcile_WithRealLLM(t *testing.T) {
	for _, tc := range availableProviders(t) {
		t.Run(tc.name, func(t *testing.T) {
			db := testDB(t)
			if err := Migrate(db); err != nil {
				t.Fatalf("migrate: %v", err)
			}

			llm := &providerLLM{p: tc.provider, model: tc.model}

			messages := []string{
				"My name is Bob and I live in San Francisco",
				"I prefer dark mode in all my code editors",
				"I'm working on an open source project called DataFlow",
			}

			// Extract facts.
			facts, err := Extract(context.Background(), llm, messages)
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			if len(facts) == 0 {
				t.Fatal("expected at least 1 fact")
			}

			// Reconcile into empty DB (should all ADD).
			result, err := Reconcile(context.Background(), db, llm, facts)
			if err != nil {
				t.Fatalf("Reconcile: %v", err)
			}

			if result.Added == 0 {
				t.Error("expected at least 1 add")
			}

			count, _ := Count(db)
			if count == 0 {
				t.Fatal("expected memories in DB after reconciliation")
			}

			// Verify memories are retrievable.
			memories, err := List(db)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			for _, m := range memories {
				if m.Content == "" {
					t.Error("memory has empty content")
				}
				if m.Source != "history" {
					t.Errorf("memory source = %q, want 'history'", m.Source)
				}
			}

			// Second extraction with same facts — should mostly NOOP/skip.
			result2, err := Reconcile(context.Background(), db, llm, facts)
			if err != nil {
				t.Fatalf("second Reconcile: %v", err)
			}

			// Count should not have grown much (some ADD is OK if LLM decides differently).
			count2, _ := Count(db)
			t.Logf("first run: added=%d, count=%d; second run: added=%d, updated=%d, skipped=%d, count=%d",
				result.Added, count, result2.Added, result2.Updated, result2.Skipped, count2)
		})
	}
}
