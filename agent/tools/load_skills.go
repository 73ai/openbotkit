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

	var parts []string
	for _, name := range in.Names {
		skillPath := filepath.Join(skills.SkillsDir(), name, "SKILL.md")
		content, err := os.ReadFile(skillPath)
		if err != nil {
			parts = append(parts, fmt.Sprintf("--- %s ---\nError: skill not found", name))
			continue
		}
		parts = append(parts, fmt.Sprintf("--- %s ---\n%s", name, string(content)))
	}

	return strings.Join(parts, "\n\n"), nil
}
