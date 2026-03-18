package memory

import (
	"strings"
	"testing"
)

func TestFormatForPromptMixed(t *testing.T) {
	memories := []Memory{
		{Content: "Name is Priyanshu", Category: CategoryIdentity},
		{Content: "Software engineer", Category: CategoryIdentity},
		{Content: "Prefers Go over Python", Category: CategoryPreference},
		{Content: "Building OpenBotKit", Category: CategoryProject},
	}

	result := FormatForPrompt(memories)

	if !strings.Contains(result, "## About the user") {
		t.Error("missing header")
	}
	if !strings.Contains(result, "### Identity") {
		t.Error("missing Identity section")
	}
	if !strings.Contains(result, "- Name is Priyanshu") {
		t.Error("missing identity fact")
	}
	if !strings.Contains(result, "### Preferences") {
		t.Error("missing Preferences section")
	}
	if !strings.Contains(result, "- Prefers Go over Python") {
		t.Error("missing preference fact")
	}
	if !strings.Contains(result, "### Relationships") {
		t.Error("missing Relationships section")
	}
	if !strings.Contains(result, "- (none)") {
		t.Error("missing (none) for empty relationships")
	}
	if !strings.Contains(result, "### Projects & Context") {
		t.Error("missing Projects section")
	}
	if !strings.Contains(result, "- Building OpenBotKit") {
		t.Error("missing project fact")
	}
}

func TestFormatForPromptEmpty(t *testing.T) {
	result := FormatForPrompt(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFormatForPromptSingleCategory(t *testing.T) {
	memories := []Memory{
		{Content: "Prefers dark mode", Category: CategoryPreference},
	}

	result := FormatForPrompt(memories)

	if !strings.Contains(result, "- Prefers dark mode") {
		t.Error("missing preference")
	}
	// Other categories should show (none).
	count := strings.Count(result, "- (none)")
	if count != 3 {
		t.Errorf("expected 3 (none) entries, got %d", count)
	}
}
