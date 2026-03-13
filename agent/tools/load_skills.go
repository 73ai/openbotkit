package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/priyanshujain/openbotkit/internal/skills"
)

// LoadSkillsTool reads full SKILL.md content for named skills.
type LoadSkillsTool struct{}

func (l *LoadSkillsTool) Name() string        { return "load_skills" }
func (l *LoadSkillsTool) Description() string { return "Load full skill instructions for the specified skills" }
func (l *LoadSkillsTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"names": {
				"type": "array",
				"items": {"type": "string"},
				"description": "List of skill names to load"
			}
		},
		"required": ["names"]
	}`)
}

func (l *LoadSkillsTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	// Auto-include gws-shared when loading any gws-* skill.
	in.Names = ensureGWSShared(in.Names)

	var parts []string
	for _, name := range in.Names {
		// Prefer REFERENCE.md (full instructions), fall back to SKILL.md.
		refPath := filepath.Join(skills.SkillsDir(), name, "REFERENCE.md")
		content, err := os.ReadFile(refPath)
		if err != nil {
			skillPath := filepath.Join(skills.SkillsDir(), name, "SKILL.md")
			content, err = os.ReadFile(skillPath)
		}
		if err != nil {
			parts = append(parts, fmt.Sprintf("--- %s ---\nError: skill not found", name))
			continue
		}
		parts = append(parts, fmt.Sprintf("--- %s ---\n%s", name, string(content)))
	}

	return strings.Join(parts, "\n\n"), nil
}

// ensureGWSShared prepends gws-shared to the list when any gws-* skill is
// requested and gws-shared isn't already included. The shared skill contains
// critical CLI syntax (--params, --json flags) that all gws services need.
func ensureGWSShared(names []string) []string {
	hasGWS, hasShared := false, false
	for _, n := range names {
		if strings.HasPrefix(n, "gws-") {
			hasGWS = true
		}
		if n == "gws-shared" {
			hasShared = true
		}
	}
	if hasGWS && !hasShared {
		return append([]string{"gws-shared"}, names...)
	}
	return names
}
