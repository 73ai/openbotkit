package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/priyanshujain/openbotkit/source/slack"
)

type SlackSendTool struct {
	deps     SlackToolDeps
	resolver *slack.Resolver
}

func NewSlackSendTool(deps SlackToolDeps) *SlackSendTool {
	return &SlackSendTool{
		deps:     deps,
		resolver: slack.NewResolver(deps.Client),
	}
}

func (t *SlackSendTool) Name() string { return "slack_send" }
func (t *SlackSendTool) Description() string {
	return "Send a message to a Slack channel (requires user approval)"
}
func (t *SlackSendTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"channel": {
				"type": "string",
				"description": "Channel name, ID, or URL"
			},
			"text": {
				"type": "string",
				"description": "Message text to send"
			},
			"thread_ts": {
				"type": "string",
				"description": "Reply to this thread (optional)"
			}
		},
		"required": ["channel", "text"]
	}`)
}

type slackSendInput struct {
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	ThreadTS string `json:"thread_ts"`
}

func (t *SlackSendTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackSendInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Channel == "" || in.Text == "" {
		return "", fmt.Errorf("channel and text are required")
	}

	channelID, err := t.resolver.ResolveChannel(ctx, in.Channel)
	if err != nil {
		return "", err
	}

	preview := in.Text
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	desc := fmt.Sprintf("Send message to %s: %s", in.Channel, preview)

	return GuardedWrite(ctx, t.deps.Interactor, desc, func() (string, error) {
		ts, err := t.deps.Client.PostMessage(ctx, channelID, in.Text, in.ThreadTS)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`{"ts":"%s"}`, ts), nil
	})
}
