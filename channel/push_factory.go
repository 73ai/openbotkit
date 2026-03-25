package channel

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/scheduler"
)

func NewPusher(channelType string, meta scheduler.ChannelMeta) (Pusher, error) {
	switch channelType {
	case "telegram":
		return NewTelegramPusher(meta.BotToken, meta.OwnerID)
	case "slack":
		return NewSlackPusher(meta.Workspace, meta.ChannelID)
	case "whatsapp":
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("load config for whatsapp pusher: %w", err)
		}
		return NewWhatsAppPusher(cfg, meta.WAAccountLabel, meta.WAOwnerJID)
	default:
		return nil, fmt.Errorf("unsupported channel type: %q", channelType)
	}
}
