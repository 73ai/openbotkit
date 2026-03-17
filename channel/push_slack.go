package channel

import (
	"context"
	"fmt"

	slacksrc "github.com/73ai/openbotkit/source/slack"
)

type SlackPusher struct {
	client    *slacksrc.Client
	channelID string
}

var _ Pusher = (*SlackPusher)(nil)

func NewSlackPusher(workspace, channelID string) (*SlackPusher, error) {
	creds, err := slacksrc.LoadCredentials(workspace)
	if err != nil {
		return nil, fmt.Errorf("load slack credentials: %w", err)
	}
	client := slacksrc.NewClient(creds.Token, creds.Cookie)
	return &SlackPusher{client: client, channelID: channelID}, nil
}

func (p *SlackPusher) Push(ctx context.Context, message string) error {
	_, err := p.client.PostMessage(ctx, p.channelID, message, "")
	return err
}
