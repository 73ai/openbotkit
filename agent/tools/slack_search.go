package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/73ai/openbotkit/source/slack"
)

type SlackSearchTool struct {
	deps SlackToolDeps
}

func NewSlackSearchTool(deps SlackToolDeps) *SlackSearchTool {
	return &SlackSearchTool{deps: deps}
}

func (t *SlackSearchTool) Name() string { return "slack_search" }
func (t *SlackSearchTool) Description() string {
	return "Search Slack messages or files by query"
}
func (t *SlackSearchTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query"
			},
			"type": {
				"type": "string",
				"enum": ["messages", "files"],
				"description": "Type of search (default: messages)"
			},
			"limit": {
				"type": "integer",
				"description": "Maximum results to return (default: 20)"
			}
		},
		"required": ["query"]
	}`)
}

type slackSearchInput struct {
	Query string `json:"query"`
	Type  string `json:"type"`
	Limit int    `json:"limit"`
}

func (t *SlackSearchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackSearchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if in.Limit <= 0 {
		in.Limit = 20
	}

	opts := slack.SearchOptions{Count: in.Limit}

	var out string
	if in.Type == "files" {
		result, err := t.deps.Client.SearchFiles(ctx, in.Query, opts)
		if err != nil {
			return "", fmt.Errorf("search files: %w", err)
		}
		out, err = compactMarshal(result)
		if err != nil {
			return "", err
		}
	} else {
		result, err := t.deps.Client.SearchMessages(ctx, in.Query, opts)
		if err != nil {
			return "", fmt.Errorf("search messages: %w", err)
		}
		out, err = compactMarshal(result)
		if err != nil {
			return "", err
		}
	}
	out = TruncateHead(out, MaxLinesSlack)
	out = TruncateBytes(out, MaxOutputBytes)
	return out, nil
}

func compactMarshal(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	compacted, err := slack.CompactJSON(data)
	if err != nil {
		return string(data), nil
	}
	return string(compacted), nil
}
