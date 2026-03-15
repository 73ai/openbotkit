package config

import "testing"

func TestModelsForProviders_FiltersByProvider(t *testing.T) {
	models := ModelsForProviders([]string{"anthropic"})
	for _, m := range models {
		if m.Provider != "anthropic" {
			t.Errorf("expected provider anthropic, got %q", m.Provider)
		}
	}
	if len(models) == 0 {
		t.Fatal("expected at least one anthropic model")
	}
}

func TestModelsForProviders_MultipleProviders(t *testing.T) {
	models := ModelsForProviders([]string{"anthropic", "gemini"})
	providers := make(map[string]bool)
	for _, m := range models {
		providers[m.Provider] = true
	}
	if !providers["anthropic"] || !providers["gemini"] {
		t.Errorf("expected anthropic and gemini, got %v", providers)
	}
}

func TestModelsForProviders_Empty(t *testing.T) {
	models := ModelsForProviders(nil)
	if len(models) != 0 {
		t.Errorf("expected no models for nil providers, got %d", len(models))
	}
}

func TestModelsForTier_ReturnsRecommended(t *testing.T) {
	all := ModelsForProviders([]string{"anthropic", "gemini"})
	fast := ModelsForTier(all, "fast")
	if len(fast) == 0 {
		t.Fatal("expected at least one fast-tier model")
	}
	for _, m := range fast {
		found := false
		for _, r := range m.RecommendedFor {
			if r == "fast" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("model %q not recommended for fast tier", m.ID)
		}
	}
}

func TestModelsForTier_UnknownTier(t *testing.T) {
	all := ModelsForProviders([]string{"anthropic"})
	result := ModelsForTier(all, "nonexistent")
	if len(result) != 0 {
		t.Errorf("expected no models for unknown tier, got %d", len(result))
	}
}

func TestModelCatalog_AllHaveRequiredFields(t *testing.T) {
	for _, m := range ModelCatalog {
		if m.Provider == "" {
			t.Errorf("model %q: empty Provider", m.ID)
		}
		if m.ID == "" {
			t.Error("empty model ID")
		}
		if m.Label == "" {
			t.Errorf("model %q: empty Label", m.ID)
		}
		if m.ContextWindow <= 0 {
			t.Errorf("model %q: invalid ContextWindow %d", m.ID, m.ContextWindow)
		}
		if len(m.RecommendedFor) == 0 {
			t.Errorf("model %q: empty RecommendedFor", m.ID)
		}
	}
}

func TestModelCatalog_NoDuplicateIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, m := range ModelCatalog {
		key := m.Provider + "/" + m.ID
		if seen[key] {
			t.Errorf("duplicate model in catalog: %s", key)
		}
		seen[key] = true
	}
}

func TestModelCatalog_ValidTierNames(t *testing.T) {
	validTiers := map[string]bool{
		"default": true,
		"complex": true,
		"fast":    true,
		"nano":    true,
	}
	for _, m := range ModelCatalog {
		for _, tier := range m.RecommendedFor {
			if !validTiers[tier] {
				t.Errorf("model %q: invalid tier %q in RecommendedFor", m.ID, tier)
			}
		}
	}
}

func TestModelCatalog_AllTiersHaveCoverage(t *testing.T) {
	tiers := []string{"default", "complex", "fast", "nano"}
	for _, tier := range tiers {
		models := ModelsForTier(ModelCatalog, tier)
		if len(models) == 0 {
			t.Errorf("no models recommended for tier %q", tier)
		}
	}
}

func TestModelsForProviders_UnknownProvider(t *testing.T) {
	models := ModelsForProviders([]string{"nonexistent"})
	if len(models) != 0 {
		t.Errorf("expected no models for unknown provider, got %d", len(models))
	}
}
