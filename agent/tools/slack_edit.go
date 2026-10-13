package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type SlackEditTool struct {
	deps       SlackToolDeps
	workspaces slackRouter
}

func NewSlackEditTool(deps SlackToolDeps) *SlackEditTool {
	return &SlackEditTool{
		deps:       deps,
		workspaces: deps.router(),
	}
}

func (t *SlackEditTool) Name() string { return "slack_edit" }
func (t *SlackEditTool) Description() string {
	return t.workspaces.describe("Edit a Slack message (requires user approval)")
}
func (t *SlackEditTool) InputSchema() json.RawMessage {
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
				"description": "Timestamp of the message to edit"
			},
			"text": {
				"type": "string",
				"description": "New message text"
			}
		},
		"required": ["channel", "ts", "text"]
	}`)
}

type slackEditInput struct {
	Workspace string `json:"workspace"`
	Channel   string `json:"channel"`
	TS        string `json:"ts"`
	Text      string `json:"text"`
}

func (t *SlackEditTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackEditInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	ws, err := t.workspaces.pick(in.Workspace)
	if err != nil {
		return "", err
	}
	if in.Channel == "" || in.TS == "" || in.Text == "" {
		return "", fmt.Errorf("channel, ts, and text are required")
	}

	channelID, err := ws.Resolver.ResolveChannel(ctx, in.Channel)
	if err != nil {
		return "", err
	}

	preview := truncateUTF8(in.Text, 100)
	desc := fmt.Sprintf("Edit message in %s: %s", t.workspaces.channelLabel(ws, in.Channel), preview)

	var opts []GuardOption
	if t.deps.ApprovalRules != nil {
		opts = append(opts, WithApprovalRules(t.deps.ApprovalRules, "slack_edit", input))
	}

	return GuardedAction(ctx, t.deps.Interactor, RiskMedium, desc, func() (string, error) {
		if err := ws.Client.UpdateMessage(ctx, channelID, in.TS, in.Text); err != nil {
			return "", err
		}
		return `{"ok":true}`, nil
	}, opts...)
}
