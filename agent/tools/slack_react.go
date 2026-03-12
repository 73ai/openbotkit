package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/priyanshujain/openbotkit/source/slack"
)

type SlackReactTool struct {
	deps     SlackToolDeps
	resolver *slack.Resolver
}

func NewSlackReactTool(deps SlackToolDeps) *SlackReactTool {
	return &SlackReactTool{
		deps:     deps,
		resolver: deps.SlackResolver(),
	}
}

func (t *SlackReactTool) Name() string { return "slack_react" }
func (t *SlackReactTool) Description() string {
	return "Add or remove a reaction on a Slack message (requires user approval)"
}
func (t *SlackReactTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"channel": {
				"type": "string",
				"description": "Channel name, ID, or URL"
			},
			"ts": {
				"type": "string",
				"description": "Timestamp of the message to react to"
			},
			"emoji": {
				"type": "string",
				"description": "Emoji name without colons (e.g. thumbsup)"
			},
			"action": {
				"type": "string",
				"enum": ["add", "remove"],
				"description": "Whether to add or remove the reaction (default: add)"
			}
		},
		"required": ["channel", "ts", "emoji"]
	}`)
}

type slackReactInput struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	Emoji   string `json:"emoji"`
	Action  string `json:"action"`
}

func (t *SlackReactTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackReactInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Channel == "" || in.TS == "" || in.Emoji == "" {
		return "", fmt.Errorf("channel, ts, and emoji are required")
	}
	if in.Action == "" {
		in.Action = "add"
	}
	if in.Action != "add" && in.Action != "remove" {
		return "", fmt.Errorf("action must be \"add\" or \"remove\", got %q", in.Action)
	}

	channelID, err := t.resolver.ResolveChannel(ctx, in.Channel)
	if err != nil {
		return "", err
	}

	action := "Add"
	if in.Action == "remove" {
		action = "Remove"
	}
	desc := fmt.Sprintf("%s :%s: reaction in %s", action, in.Emoji, in.Channel)

	return GuardedAction(ctx, t.deps.Interactor, RiskLow, desc, func() (string, error) {
		if in.Action == "remove" {
			return `{"ok":true}`, t.deps.Client.RemoveReaction(ctx, channelID, in.TS, in.Emoji)
		}
		return `{"ok":true}`, t.deps.Client.AddReaction(ctx, channelID, in.TS, in.Emoji)
	})
}
