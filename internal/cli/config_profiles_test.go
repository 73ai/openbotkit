package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
)

func setupTestCache(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cache := provider.NewModelCache(config.ModelsDir())
	cache.Save("anthropic", &provider.CachedModelList{
		Provider:  "anthropic",
		FetchedAt: time.Now(),
		Models: []provider.AvailableModel{
			{ID: "claude-sonnet-4-6", DisplayName: "Claude Sonnet 4.6", Provider: "anthropic"},
			{ID: "claude-haiku-4-5", DisplayName: "Claude Haiku 4.5", Provider: "anthropic"},
			{ID: "claude-opus-4-6", DisplayName: "Claude Opus 4.6", Provider: "anthropic"},
		},
	})
	cache.Save("gemini", &provider.CachedModelList{
		Provider:  "gemini",
		FetchedAt: time.Now(),
		Models: []provider.AvailableModel{
			{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash", Provider: "gemini"},
			{ID: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro", Provider: "gemini"},
			{ID: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash", Provider: "gemini"},
		},
	})
	cache.Save("openai", &provider.CachedModelList{
		Provider:  "openai",
		FetchedAt: time.Now(),
		Models: []provider.AvailableModel{
			{ID: "gpt-4o", DisplayName: "GPT-4o", Provider: "openai"},
			{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Provider: "openai"},
		},
	})
}

func TestBuildTierOptions_ReturnsModels(t *testing.T) {
	setupTestCache(t)
	options := buildTierOptions([]string{"anthropic"})
	if len(options) == 0 {
		t.Fatal("expected options from cache")
	}
	for _, opt := range options {
		if !strings.Contains(opt.Value, "anthropic/") {
			t.Errorf("option %q should be from anthropic", opt.Value)
		}
	}
}

func TestBuildTierOptions_MultipleProviders(t *testing.T) {
	setupTestCache(t)
	options := buildTierOptions([]string{"anthropic", "gemini"})
	if len(options) < 4 {
		t.Fatalf("expected at least 4 options, got %d", len(options))
	}
	providers := make(map[string]bool)
	for _, opt := range options {
		parts := strings.SplitN(opt.Value, "/", 2)
		if len(parts) == 2 {
			providers[parts[0]] = true
		}
	}
	if !providers["anthropic"] || !providers["gemini"] {
		t.Errorf("expected both providers, got %v", providers)
	}
}

func TestBuildTierOptions_NoDuplicates(t *testing.T) {
	setupTestCache(t)
	options := buildTierOptions([]string{"anthropic", "gemini"})
	seen := make(map[string]bool)
	for _, opt := range options {
		if seen[opt.Value] {
			t.Errorf("duplicate option value: %q", opt.Value)
		}
		seen[opt.Value] = true
	}
}

func TestBuildTierOptions_EmptyProviders(t *testing.T) {
	setupTestCache(t)
	options := buildTierOptions(nil)
	if len(options) != 0 {
		t.Errorf("expected no options for nil providers, got %d", len(options))
	}
}

func TestBuildTierOptions_UncachedProvider(t *testing.T) {
	setupTestCache(t)
	options := buildTierOptions([]string{"nonexistent"})
	if len(options) != 0 {
		t.Errorf("expected no options for uncached provider, got %d", len(options))
	}
}

func TestConfigProfilesDelete_BuiltInProfile(t *testing.T) {
	setupTestCache(t)

	cmd := configProfilesDeleteCmd
	cmd.SetArgs([]string{"gemini"})
	err := cmd.RunE(cmd, []string{"gemini"})
	if err == nil {
		t.Fatal("expected error deleting built-in profile")
	}
	if !strings.Contains(err.Error(), "cannot delete built-in") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigProfilesDelete_NonExistent(t *testing.T) {
	setupTestCache(t)

	configProfilesDeleteCmd.Flags().Set("force", "true")
	defer configProfilesDeleteCmd.Flags().Set("force", "false")
	err := configProfilesDeleteCmd.RunE(configProfilesDeleteCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error deleting non-existent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigProfilesDelete_ClearsActiveProfile(t *testing.T) {
	setupTestCache(t)

	cfg := config.Default()
	cfg.Models = &config.ModelsConfig{
		Default: "gemini/gemini-2.5-flash",
		Profile: "my-test",
		CustomProfiles: map[string]config.CustomProfile{
			"my-test": {
				Label: "Test",
				Tiers: config.ProfileTiers{
					Default: "gemini/gemini-2.5-flash",
					Complex: "gemini/gemini-2.5-pro",
					Fast:    "gemini/gemini-2.0-flash",
					Nano:    "gemini/gemini-2.0-flash",
				},
				Providers: []string{"gemini"},
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	configProfilesDeleteCmd.Flags().Set("force", "true")
	defer configProfilesDeleteCmd.Flags().Set("force", "false")
	err := configProfilesDeleteCmd.RunE(configProfilesDeleteCmd, []string{"my-test"})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Models.Profile != "" {
		t.Errorf("active profile should be cleared after delete, got %q", loaded.Models.Profile)
	}
	if loaded.Models.CustomProfiles != nil {
		t.Error("CustomProfiles should be nil after deleting last profile")
	}
}

func TestConfigProfilesShow_BuiltIn(t *testing.T) {
	err := configProfilesShowCmd.RunE(configProfilesShowCmd, []string{"gemini"})
	if err != nil {
		t.Fatalf("show built-in profile: %v", err)
	}
}

func TestConfigProfilesShow_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	err := configProfilesShowCmd.RunE(configProfilesShowCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for non-existent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigProfilesShow_CustomProfile(t *testing.T) {
	setupTestCache(t)

	cfg := config.Default()
	cfg.Models = &config.ModelsConfig{
		Default: "gemini/gemini-2.5-flash",
		CustomProfiles: map[string]config.CustomProfile{
			"my-custom": {
				Label:       "My Custom",
				Description: "Test description",
				Tiers: config.ProfileTiers{
					Default: "gemini/gemini-2.5-flash",
					Complex: "gemini/gemini-2.5-pro",
					Fast:    "gemini/gemini-2.0-flash",
					Nano:    "gemini/gemini-2.0-flash",
				},
				Providers: []string{"gemini"},
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	err := configProfilesShowCmd.RunE(configProfilesShowCmd, []string{"my-custom"})
	if err != nil {
		t.Fatalf("show custom profile: %v", err)
	}
}

func TestConfigProfilesList_NoConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	err := configProfilesListCmd.RunE(configProfilesListCmd, nil)
	if err != nil {
		t.Fatalf("list with no config: %v", err)
	}
}
