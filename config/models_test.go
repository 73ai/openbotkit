package config

import "testing"

func TestModelInfo_HasFields(t *testing.T) {
	m := ModelInfo{
		Provider:       "anthropic",
		ID:             "claude-sonnet-4-6",
		Label:          "Claude Sonnet 4.6",
		ContextWindow:  200000,
		RecommendedFor: []string{"default", "complex"},
	}
	if m.Provider == "" || m.ID == "" || m.Label == "" {
		t.Error("ModelInfo fields should not be empty")
	}
	if m.ContextWindow <= 0 {
		t.Error("ContextWindow should be positive")
	}
	if len(m.RecommendedFor) == 0 {
		t.Error("RecommendedFor should not be empty")
	}
}
