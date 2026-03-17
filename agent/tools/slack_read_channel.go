package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/73ai/openbotkit/source/slack"
)

type SlackReadChannelTool struct {
	deps     SlackToolDeps
	resolver *slack.Resolver
}

func NewSlackReadChannelTool(deps SlackToolDeps) *SlackReadChannelTool {
	return &SlackReadChannelTool{
		deps:     deps,
		resolver: deps.SlackResolver(),
	}
}

func (t *SlackReadChannelTool) Name() string { return "slack_read_channel" }
func (t *SlackReadChannelTool) Description() string {
	return "Read recent messages from a Slack channel"
}
func (t *SlackReadChannelTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"channel": {
				"type": "string",
				"description": "Channel name (#general), ID (C0123ABC), or URL"
			},
			"limit": {
				"type": "integer",
				"description": "Number of messages to fetch (default: 20)"
			},
			"oldest": {
				"type": "string",
				"description": "Only messages after this Unix timestamp"
			},
			"latest": {
				"type": "string",
				"description": "Only messages before this Unix timestamp"
			}
		},
		"required": ["channel"]
	}`)
}

type slackReadChannelInput struct {
	Channel string `json:"channel"`
	Limit   int    `json:"limit"`
	Oldest  string `json:"oldest"`
	Latest  string `json:"latest"`
}

func (t *SlackReadChannelTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackReadChannelInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Channel == "" {
		return "", fmt.Errorf("channel is required")
	}
	if in.Limit <= 0 {
		in.Limit = 20
	}

	channelID, err := t.resolver.ResolveChannel(ctx, in.Channel)
	if err != nil {
		return "", err
	}

	msgs, err := t.deps.Client.ConversationsHistory(ctx, channelID, slack.HistoryOptions{
		Limit:  in.Limit,
		Oldest: in.Oldest,
		Latest: in.Latest,
	})
	if err != nil {
		return "", fmt.Errorf("fetch history: %w", err)
	}

	out, err := compactMarshal(msgs)
	if err != nil {
		return "", err
	}
	out = TruncateHead(out, MaxLinesSlack)
	out = TruncateBytes(out, MaxOutputBytes)
	return out, nil
}
