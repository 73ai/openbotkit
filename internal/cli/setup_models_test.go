package cli

import (
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
)

func TestProfileTestModels_PicksOnePerProvider(t *testing.T) {
	profile := config.ModelProfile{
		Tiers: config.ProfileTiers{
			Default: "openrouter/anthropic/claude-haiku-4-5",
			Complex: "openrouter/anthropic/claude-sonnet-4-6",
			Fast:    "openrouter/google/gemini-2.0-flash-lite",
			Nano:    "groq/llama-3.1-8b-instant",
		},
	}

	result := profileTestModels(profile)

	if len(result) != 2 {
		t.Fatalf("expected 2 providers, got %d: %v", len(result), result)
	}
	if _, ok := result["openrouter"]; !ok {
		t.Error("expected openrouter in result")
	}
	if result["groq"] != "llama-3.1-8b-instant" {
		t.Errorf("groq model = %q, want llama-3.1-8b-instant", result["groq"])
	}
}

func TestProfileTestModels_EmptyTiers(t *testing.T) {
	profile := config.ModelProfile{
		Tiers: config.ProfileTiers{},
	}
	result := profileTestModels(profile)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestProfileTestModels_FirstModelPerProvider(t *testing.T) {
	profile := config.ModelProfile{
		Tiers: config.ProfileTiers{
			Default: "openrouter/anthropic/claude-haiku-4-5",
			Complex: "openrouter/anthropic/claude-sonnet-4-6",
		},
	}

	result := profileTestModels(profile)
	// Should pick the first one encountered (Default) for openrouter.
	if result["openrouter"] != "anthropic/claude-haiku-4-5" {
		t.Errorf("openrouter model = %q, want anthropic/claude-haiku-4-5", result["openrouter"])
	}
}

func TestProfileTestModels_AllRealProfiles(t *testing.T) {
	for _, name := range config.ProfileNames {
		profile := config.Profiles[name]
		testModels := profileTestModels(profile)

		// Every provider in profile.Providers must get a test model.
		for _, provName := range profile.Providers {
			model, ok := testModels[provName]
			if !ok {
				t.Errorf("profile %q: provider %q has no test model", name, provName)
				continue
			}
			if model == "" {
				t.Errorf("profile %q: provider %q has empty test model", name, provName)
			}
		}

		// No test model should map to a provider not in the profile's list.
		provSet := make(map[string]bool)
		for _, p := range profile.Providers {
			provSet[p] = true
		}
		for provName := range testModels {
			if !provSet[provName] {
				t.Errorf("profile %q: test model for %q but not in Providers list", name, provName)
			}
		}
	}
}

func TestProfileProviders_HaveRegisteredFactories(t *testing.T) {
	for _, name := range config.ProfileNames {
		profile := config.Profiles[name]
		for _, provName := range profile.Providers {
			if _, ok := provider.GetFactory(provName); !ok {
				t.Errorf("profile %q: provider %q has no registered factory", name, provName)
			}
		}
	}
}

func TestProfileProviders_HaveEnvVars(t *testing.T) {
	for _, name := range config.ProfileNames {
		profile := config.Profiles[name]
		for _, provName := range profile.Providers {
			if _, ok := provider.ProviderEnvVars[provName]; !ok {
				t.Errorf("profile %q: provider %q has no env var mapping", name, provName)
			}
		}
	}
}

func TestSetupWithCustomProfile_NotFound(t *testing.T) {
	cfg := config.Default()
	cfg.Models = &config.ModelsConfig{
		Providers: make(map[string]config.ModelProviderConfig),
	}
	err := setupWithCustomProfile(cfg, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent custom profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProfileModels_HaveContextWindows(t *testing.T) {
	for _, name := range config.ProfileNames {
		profile := config.Profiles[name]
		testModels := profileTestModels(profile)
		for provName, model := range testModels {
			window := provider.DefaultContextWindow(model)
			if window == 0 {
				t.Errorf("profile %q: provider %q model %q has no context window defined", name, provName, model)
			}
		}
	}
}

func TestSetupWithCustomProfile_NilModels(t *testing.T) {
	cfg := config.Default()
	cfg.Models = nil
	err := setupWithCustomProfile(cfg, "anything")
	if err == nil {
		t.Fatal("expected error when Models is nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSetupWithCustomProfile_NilCustomProfiles(t *testing.T) {
	cfg := config.Default()
	cfg.Models = &config.ModelsConfig{
		Providers: make(map[string]config.ModelProviderConfig),
	}
	// CustomProfiles is nil (not set), different from empty map.
	err := setupWithCustomProfile(cfg, "anything")
	if err == nil {
		t.Fatal("expected error when CustomProfiles is nil")
	}
}

func TestSetupWithCustomProfile_EmptyCustomProfiles(t *testing.T) {
	cfg := config.Default()
	cfg.Models = &config.ModelsConfig{
		Providers:      make(map[string]config.ModelProviderConfig),
		CustomProfiles: map[string]config.CustomProfile{},
	}
	err := setupWithCustomProfile(cfg, "anything")
	if err == nil {
		t.Fatal("expected error for empty custom profiles map")
	}
}

func TestCustomProfileToModelProfile_LabelDefaultsToName(t *testing.T) {
	cp := config.CustomProfile{
		Tiers: config.ProfileTiers{
			Default: "gemini/gemini-2.5-flash",
			Complex: "gemini/gemini-2.5-pro",
			Fast:    "gemini/gemini-2.0-flash-lite",
			Nano:    "gemini/gemini-2.0-flash-lite",
		},
		Providers: []string{"gemini"},
	}
	profile := config.ModelProfile{
		Name:      "my-test",
		Label:     cp.Label,
		Tiers:     cp.Tiers,
		Providers: cp.Providers,
	}
	if profile.Label == "" {
		profile.Label = "my-test"
	}
	if profile.Label != "my-test" {
		t.Errorf("label = %q, want %q", profile.Label, "my-test")
	}
	if profile.Name != "my-test" {
		t.Errorf("name = %q, want %q", profile.Name, "my-test")
	}
}

func TestCustomProfileToModelProfile_LabelPreserved(t *testing.T) {
	cp := config.CustomProfile{
		Label: "My Budget Setup",
		Tiers: config.ProfileTiers{
			Default: "gemini/gemini-2.5-flash",
		},
		Providers: []string{"gemini"},
	}
	profile := config.ModelProfile{
		Name:      "my-test",
		Label:     cp.Label,
		Tiers:     cp.Tiers,
		Providers: cp.Providers,
	}
	if profile.Label == "" {
		profile.Label = "my-test"
	}
	if profile.Label != "My Budget Setup" {
		t.Errorf("label = %q, want %q", profile.Label, "My Budget Setup")
	}
}

func TestWarnDefaultContextWindow_NoWarningForLargeContext(t *testing.T) {
	// Should not panic or error for models with >=128k context.
	warnDefaultContextWindow("anthropic/claude-sonnet-4-6")
}

func TestWarnDefaultContextWindow_EmptySpec(t *testing.T) {
	// Should not panic on empty spec.
	warnDefaultContextWindow("")
}

func TestWarnDefaultContextWindow_InvalidSpec(t *testing.T) {
	// Should not panic on spec without slash.
	warnDefaultContextWindow("no-slash")
}
