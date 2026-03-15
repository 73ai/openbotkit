package config

import (
	"strings"
	"testing"
)

func TestProfilesHaveValidModelSpecs(t *testing.T) {
	for name, profile := range Profiles {
		for _, spec := range []string{profile.Tiers.Default, profile.Tiers.Complex, profile.Tiers.Fast, profile.Tiers.Nano} {
			if spec == "" {
				continue
			}
			parts := strings.SplitN(spec, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				t.Errorf("profile %q: invalid model spec %q", name, spec)
			}
		}
	}
}

func TestProfilesHaveRequiredProviders(t *testing.T) {
	for name, profile := range Profiles {
		providerSet := make(map[string]bool)
		for _, p := range profile.Providers {
			providerSet[p] = true
		}

		for _, spec := range []string{profile.Tiers.Default, profile.Tiers.Complex, profile.Tiers.Fast, profile.Tiers.Nano} {
			if spec == "" {
				continue
			}
			parts := strings.SplitN(spec, "/", 2)
			provName := parts[0]
			if !providerSet[provName] {
				t.Errorf("profile %q: tier spec %q uses provider %q not in Providers list", name, spec, provName)
			}
		}
	}
}

func TestProfileNamesMatchKeys(t *testing.T) {
	for _, name := range ProfileNames {
		profile, ok := Profiles[name]
		if !ok {
			t.Errorf("ProfileNames contains %q but Profiles map does not", name)
			continue
		}
		if profile.Name != name {
			t.Errorf("profile %q: Name field = %q, want %q", name, profile.Name, name)
		}
	}
}

func TestAllProfilesInProfileNames(t *testing.T) {
	nameSet := make(map[string]bool)
	for _, n := range ProfileNames {
		nameSet[n] = true
	}
	for name := range Profiles {
		if !nameSet[name] {
			t.Errorf("profile %q in Profiles map but not in ProfileNames", name)
		}
	}
}

func TestProfilesHaveCategory(t *testing.T) {
	for name, profile := range Profiles {
		if profile.Category != "single" && profile.Category != "multi" {
			t.Errorf("profile %q: Category = %q, want \"single\" or \"multi\"", name, profile.Category)
		}
	}
}

func TestSingleProviderProfilesHaveOneProvider(t *testing.T) {
	for name, profile := range Profiles {
		if profile.Category != "single" {
			continue
		}
		if len(profile.Providers) != 1 {
			t.Errorf("profile %q: single-provider profile has %d providers, want 1", name, len(profile.Providers))
		}
	}
}

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"my-setup", false},
		{"budget", false},
		{"ab", false},
		{"a23456789012345678901234567890", false}, // 30 chars

		{"", true},          // too short
		{"a", true},         // too short (min 2)
		{"A-upper", true},   // uppercase
		{"1start", true},    // starts with number
		{"-start", true},    // starts with hyphen
		{"has space", true}, // contains space
		{"has_under", true}, // contains underscore

		// Reserved names
		{"custom", true},
		{"gemini", true},     // built-in
		{"anthropic", true},  // built-in
		{"starter", true},    // built-in
		{"standard", true},   // built-in
		{"premium", true},    // built-in
		{"groq", true},       // built-in
		{"openrouter", true}, // built-in
		{"openai", true},     // built-in
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfileName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
