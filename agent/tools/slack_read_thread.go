package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/73ai/openbotkit/source/slack"
)

type SlackReadThreadTool struct {
	workspaces slackRouter
}

func NewSlackReadThreadTool(deps SlackToolDeps) *SlackReadThreadTool {
	return &SlackReadThreadTool{
		workspaces: deps.router(),
	}
}

func (t *SlackReadThreadTool) Name() string { return "slack_read_thread" }
func (t *SlackReadThreadTool) Description() string {
	return t.workspaces.describe("Read a complete Slack thread, with user IDs resolved to names. Accepts a message permalink, or a channel plus thread_ts. Any message in the thread works: a link to a reply returns the whole thread.")
}
func (t *SlackReadThreadTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			` + slackWorkspaceProp + `
			"permalink": {
				"type": "string",
				"description": "Slack message permalink, e.g. https://acme.slack.com/archives/C01ABC/p1785334150236249"
			},
			"channel": {
				"type": "string",
				"description": "Channel name, ID, or URL (use with thread_ts)"
			},
			"thread_ts": {
				"type": "string",
				"description": "Timestamp of any message in the thread, dotted (1785334150.236249) or p-form (p1785334150236249)"
			}
		}
	}`)
}

type slackReadThreadInput struct {
	Workspace string `json:"workspace"`
	Permalink string `json:"permalink"`
	Channel   string `json:"channel"`
	ThreadTS  string `json:"thread_ts"`
}

func (t *SlackReadThreadTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in slackReadThreadInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	ws, channelID, ts, err := t.resolve(ctx, in)
	if err != nil {
		return "", err
	}

	thread, err := slack.FetchThread(ctx, ws.Client, ws.UserCache, channelID, ts)
	if err != nil {
		return "", err
	}

	out, err := compactMarshal(thread)
	if err != nil {
		return "", err
	}
	out = TruncateHead(out, MaxLinesSlack)
	out = TruncateBytes(out, MaxOutputBytes)
	return out, nil
}

func (t *SlackReadThreadTool) resolve(ctx context.Context, in slackReadThreadInput) (ws *SlackWorkspace, channelID, ts string, err error) {
	if in.Permalink != "" {
		ref, err := slack.ParsePermalink(in.Permalink)
		if err != nil {
			return nil, "", "", err
		}
		ts, err := ref.RootTS()
		if err != nil {
			return nil, "", "", err
		}
		ws, err := t.workspaces.forSubdomain(in.Workspace, ref.Subdomain)
		if err != nil {
			return nil, "", "", err
		}
		return ws, ref.ChannelID, ts, nil
	}

	if in.Channel == "" || in.ThreadTS == "" {
		return nil, "", "", fmt.Errorf("provide permalink, or both channel and thread_ts")
	}
	if ws, err = t.workspaces.pick(in.Workspace); err != nil {
		return nil, "", "", err
	}
	channelID, err = ws.Resolver.ResolveChannel(ctx, in.Channel)
	if err != nil {
		return nil, "", "", err
	}
	if ts, err = slack.MessageIDToTS(in.ThreadTS); err != nil {
		return nil, "", "", err
	}
	return ws, channelID, ts, nil
}
