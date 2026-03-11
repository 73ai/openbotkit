package channel

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramPusher struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

var _ Pusher = (*TelegramPusher)(nil)

func NewTelegramPusher(botToken string, chatID int64) (*TelegramPusher, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	return &TelegramPusher{bot: bot, chatID: chatID}, nil
}

func (p *TelegramPusher) Push(_ context.Context, message string) error {
	m := tgbotapi.NewMessage(p.chatID, message)
	m.ParseMode = "Markdown"
	_, err := p.bot.Send(m)
	return err
}
