package provider

import (
	"testing"

	"github.com/priyanshujain/openbotkit/config"
)

func TestParseModelSpec(t *testing.T) {
	tests := []struct {
		spec     string
		provider string
		model    string
		wantErr  bool
	}{
		{"anthropic/claude-sonnet-4-6", "anthropic", "claude-sonnet-4-6", false},
		{"openai/gpt-4o-mini", "openai", "gpt-4o-mini", false},
		{"gemini/gemini-2.0-flash", "gemini", "gemini-2.0-flash", false},
		{"provider/model/extra", "provider", "model/extra", false}, // SplitN with 2
		{"no-slash", "", "", true},
		{"", "", "", true},
		{"/model", "", "", true},
		{"provider/", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			p, m, err := ParseModelSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if p != tt.provider {
				t.Errorf("provider = %q, want %q", p, tt.provider)
			}
			if m != tt.model {
				t.Errorf("model = %q, want %q", m, tt.model)
			}
		})
	}
}

func TestNewRegistry_NilModels(t *testing.T) {
	r, err := NewRegistry(nil)
	if err != nil {
		t.Fatalf("NewRegistry(nil): %v", err)
	}
	if _, ok := r.Get("anthropic"); ok {
		t.Error("expected no providers in empty registry")
	}
}

func TestGetFactory_Registered(t *testing.T) {
	RegisterFactory("test-provider", func(cfg config.ModelProviderConfig, apiKey string) Provider {
		return nil
	})
	defer delete(factories, "test-provider")

	f, ok := GetFactory("test-provider")
	if !ok {
		t.Fatal("expected factory to be found")
	}
	if f == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestGetFactory_NotFound(t *testing.T) {
	_, ok := GetFactory("nonexistent-provider")
	if ok {
		t.Fatal("expected ok=false for unregistered provider")
	}
}

func TestCreateProvider_VertexAI_SkipsAPIKey(t *testing.T) {
	RegisterFactory("test-vertex", func(cfg config.ModelProviderConfig, apiKey string) Provider {
		if apiKey != "" {
			t.Errorf("expected empty apiKey for vertex_ai, got %q", apiKey)
		}
		if cfg.AuthMethod != "vertex_ai" {
			t.Errorf("expected auth_method vertex_ai, got %q", cfg.AuthMethod)
		}
		return &mockProvider{}
	})
	defer delete(factories, "test-vertex")

	p, err := createProvider("test-vertex", config.ModelProviderConfig{
		AuthMethod:    "vertex_ai",
		VertexProject: "my-project",
		VertexRegion:  "us-central1",
	})
	if err != nil {
		t.Fatalf("createProvider: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestRegistryGet_NotFound(t *testing.T) {
	r, _ := NewRegistry(nil)
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for nonexistent provider")
	}
}
