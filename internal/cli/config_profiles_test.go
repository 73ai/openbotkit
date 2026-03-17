package cli

import (
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestBuildTierOptions_RecommendedFirst(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic"})
	options := buildTierOptions(models, "complex")

	if len(options) == 0 {
		t.Fatal("expected options for complex tier")
	}

	// Recommended options have " *" suffix in label.
	first := options[0]
	if !strings.HasSuffix(first.Key, " *") {
		t.Errorf("first option should be recommended (have * suffix), got label %q", first.Key)
	}
}

func TestBuildTierOptions_RecommendedHaveStarMarker(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic", "gemini"})
	options := buildTierOptions(models, "fast")

	recommended := config.ModelsForTier(models, "fast")
	recommendedSpecs := make(map[string]bool)
	for _, m := range recommended {
		recommendedSpecs[m.Provider+"/"+m.ID] = true
	}

	for _, opt := range options {
		isRecommended := recommendedSpecs[opt.Value]
		hasStar := strings.HasSuffix(opt.Key, " *")
		if isRecommended && !hasStar {
			t.Errorf("recommended model %q should have * marker", opt.Value)
		}
		if !isRecommended && hasStar {
			t.Errorf("non-recommended model %q should not have * marker", opt.Value)
		}
	}
}

func TestBuildTierOptions_NoDuplicates(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic", "gemini"})
	options := buildTierOptions(models, "default")

	seen := make(map[string]bool)
	for _, opt := range options {
		if seen[opt.Value] {
			t.Errorf("duplicate option value: %q", opt.Value)
		}
		seen[opt.Value] = true
	}
}

func TestBuildTierOptions_AllModelsIncluded(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic"})
	options := buildTierOptions(models, "default")

	// Should include all anthropic models, not just recommended ones.
	if len(options) < len(models) {
		t.Errorf("expected at least %d options, got %d", len(models), len(options))
	}
}

func TestBuildTierOptions_EmptyModels(t *testing.T) {
	options := buildTierOptions(nil, "default")
	if len(options) != 0 {
		t.Errorf("expected no options for nil models, got %d", len(options))
	}
}

func TestBuildTierOptions_AllTiers(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic", "gemini", "openai"})
	for _, tier := range []string{"default", "complex", "fast", "nano"} {
		options := buildTierOptions(models, tier)
		if len(options) == 0 {
			t.Errorf("no options for tier %q", tier)
		}
		// All models should appear (recommended first, rest after).
		if len(options) != len(models) {
			t.Errorf("tier %q: expected %d options (all models), got %d", tier, len(models), len(options))
		}
	}
}

func TestConfigProfilesDelete_BuiltInProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

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
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

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
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	// Create a config with a custom profile that is active.
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
					Fast:    "gemini/gemini-2.0-flash-lite",
					Nano:    "gemini/gemini-2.0-flash-lite",
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
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

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
					Fast:    "gemini/gemini-2.0-flash-lite",
					Nano:    "gemini/gemini-2.0-flash-lite",
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

	// Should not error even with no config file.
	err := configProfilesListCmd.RunE(configProfilesListCmd, nil)
	if err != nil {
		t.Fatalf("list with no config: %v", err)
	}
}
