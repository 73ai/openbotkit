package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/73ai/openbotkit/source/slack"
)

type SlackReadThreadTool struct {
	deps     SlackToolDeps
	resolver *slack.Resolver
}

func NewSlackReadThreadTool(deps SlackToolDeps) *SlackReadThreadTool {
	return &SlackReadThreadTool{
		deps:     deps,
		resolver: deps.SlackResolver(),
	}
}

func (t *SlackReadThreadTool) Name() string { return "slack_read_thread" }
func (t *SlackReadThreadTool) Description() string {
	return "Read replies in a Slack thread"
}
func (t *SlackReadThreadTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"channel": {
				"type": "string",
				"description": "Channel name, ID, or URL"
			},
			"thread_ts": {
				"type": "string",
				"description": "Timestamp of the parent message"
			}
		},
		"required": ["channel", "thread_ts"]
	}`)
}

type slackReadThreadInput struct {
	Channel  string `json:"channel"`
	ThreadTS string `json:"thread_ts"`
}

func (t *SlackReadThreadTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackReadThreadInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Channel == "" || in.ThreadTS == "" {
		return "", fmt.Errorf("channel and thread_ts are required")
	}

	channelID, err := t.resolver.ResolveChannel(ctx, in.Channel)
	if err != nil {
		return "", err
	}

	msgs, err := t.deps.Client.ConversationsReplies(ctx, channelID, in.ThreadTS, slack.HistoryOptions{})
	if err != nil {
		return "", fmt.Errorf("fetch replies: %w", err)
	}

	out, err := compactMarshal(msgs)
	if err != nil {
		return "", err
	}
	out = TruncateHead(out, MaxLinesSlack)
	out = TruncateBytes(out, MaxOutputBytes)
	return out, nil
}
