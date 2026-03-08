package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/priyanshujain/openbotkit/internal/skills"
)

// SearchSkillsTool searches the skill index by keyword.
type SearchSkillsTool struct{}

func (s *SearchSkillsTool) Name() string        { return "search_skills" }
func (s *SearchSkillsTool) Description() string { return "Search available skills by keyword" }
func (s *SearchSkillsTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search term to find matching skills"
			}
		},
		"required": ["query"]
	}`)
}

func (s *SearchSkillsTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	idx, err := skills.LoadIndex()
	if err != nil {
		return "", fmt.Errorf("load index: %w", err)
	}

	query := strings.ToLower(in.Query)
	var matches []string
	for _, entry := range idx.Skills {
		if strings.Contains(strings.ToLower(entry.Name), query) ||
			strings.Contains(strings.ToLower(entry.Description), query) {
			matches = append(matches, fmt.Sprintf("- %s: %s", entry.Name, entry.Description))
		}
	}

	if len(matches) == 0 {
		return "No matching skills found.", nil
	}
	return strings.Join(matches, "\n"), nil
}
