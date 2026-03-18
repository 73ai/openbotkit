package channel

import (
	"fmt"

	"github.com/73ai/openbotkit/service/scheduler"
)

func NewPusher(channelType string, meta scheduler.ChannelMeta) (Pusher, error) {
	switch channelType {
	case "telegram":
		return NewTelegramPusher(meta.BotToken, meta.OwnerID)
	case "slack":
		return NewSlackPusher(meta.Workspace, meta.ChannelID)
	default:
		return nil, fmt.Errorf("unsupported channel type: %q", channelType)
	}
}
