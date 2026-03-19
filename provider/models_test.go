package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestListModels_OpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "gpt-4o", "owned_by": "openai"},
				{"id": "gpt-4o-mini", "owned_by": "openai"},
				{"id": "text-embedding-ada-002", "owned_by": "openai"},
			},
		})
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), "openai", "test-key", config.ModelProviderConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 chat models (filtered), got %d", len(models))
	}
	if models[0].ID != "gpt-4o" || models[0].Provider != "openai" {
		t.Errorf("unexpected first model: %+v", models[0])
	}
}

func TestListModels_Anthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected auth header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "claude-sonnet-4-6-20260301", "display_name": "Claude Sonnet 4.6"},
				{"id": "claude-haiku-4-5-20251001", "display_name": "Claude Haiku 4.5"},
			},
			"has_more": false,
		})
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), "anthropic", "test-key", config.ModelProviderConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].DisplayName != "Claude Sonnet 4.6" {
		t.Errorf("unexpected display name: %s", models[0].DisplayName)
	}
}

func TestListModels_Gemini(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("unexpected key param")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"name":                       "models/gemini-2.5-flash",
					"displayName":                "Gemini 2.5 Flash",
					"supportedGenerationMethods": []string{"generateContent"},
				},
				{
					"name":                       "models/embedding-001",
					"displayName":                "Embedding 001",
					"supportedGenerationMethods": []string{"embedContent"},
				},
			},
		})
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), "gemini", "test-key", config.ModelProviderConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 chat model (filtered), got %d", len(models))
	}
	if models[0].ID != "gemini-2.5-flash" {
		t.Errorf("expected models/ prefix stripped, got %s", models[0].ID)
	}
}

func TestListModels_Groq(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "llama-3.1-8b-instant"},
				{"id": "llama-3.3-70b-versatile"},
			},
		})
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), "groq", "test-key", config.ModelProviderConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Provider != "groq" {
		t.Errorf("expected provider groq, got %s", models[0].Provider)
	}
}

func TestListModels_OpenRouter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "anthropic/claude-sonnet-4-6"},
				{"id": "google/gemini-2.5-flash"},
			},
		})
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), "openrouter", "test-key", config.ModelProviderConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Provider != "openrouter" {
		t.Errorf("expected provider openrouter, got %s", models[0].Provider)
	}
}

func TestListModels_UnknownProvider(t *testing.T) {
	_, err := ListModels(context.Background(), "unknown", "key", config.ModelProviderConfig{})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestListModels_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := ListModels(context.Background(), "openai", "bad-key", config.ModelProviderConfig{BaseURL: server.URL})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestListModels_AnthropicPagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "claude-sonnet-4-6", "display_name": "Claude Sonnet 4.6"},
				},
				"has_more": true,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "claude-haiku-4-5", "display_name": "Claude Haiku 4.5"},
				},
				"has_more": false,
			})
		}
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), "anthropic", "test-key", config.ModelProviderConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models after pagination, got %d", len(models))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}
