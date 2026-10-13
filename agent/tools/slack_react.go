package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type SlackReactTool struct {
	deps       SlackToolDeps
	workspaces slackRouter
}

func NewSlackReactTool(deps SlackToolDeps) *SlackReactTool {
	return &SlackReactTool{
		deps:       deps,
		workspaces: deps.router(),
	}
}

func (t *SlackReactTool) Name() string { return "slack_react" }
func (t *SlackReactTool) Description() string {
	return t.workspaces.describe("Add or remove a reaction on a Slack message (requires user approval)")
}
func (t *SlackReactTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			` + slackWorkspaceProp + `
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
	Workspace string `json:"workspace"`
	Channel   string `json:"channel"`
	TS        string `json:"ts"`
	Emoji     string `json:"emoji"`
	Action    string `json:"action"`
}

func (t *SlackReactTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackReactInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	ws, err := t.workspaces.pick(in.Workspace)
	if err != nil {
		return "", err
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

	channelID, err := ws.Resolver.ResolveChannel(ctx, in.Channel)
	if err != nil {
		return "", err
	}

	action := "Add"
	if in.Action == "remove" {
		action = "Remove"
	}
	desc := fmt.Sprintf("%s :%s: reaction in %s", action, in.Emoji, t.workspaces.channelLabel(ws, in.Channel))

	return GuardedAction(ctx, t.deps.Interactor, RiskLow, desc, func() (string, error) {
		if in.Action == "remove" {
			return `{"ok":true}`, ws.Client.RemoveReaction(ctx, channelID, in.TS, in.Emoji)
		}
		return `{"ok":true}`, ws.Client.AddReaction(ctx, channelID, in.TS, in.Emoji)
	})
}
