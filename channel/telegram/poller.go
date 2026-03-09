package telegram

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Poller receives updates from Telegram and routes them to the channel.
type Poller struct {
	bot     *tgbotapi.BotAPI
	ownerID int64
	channel *Channel
}

func NewPoller(bot *tgbotapi.BotAPI, ownerID int64, ch *Channel) *Poller {
	return &Poller{bot: bot, ownerID: ownerID, channel: ch}
}

func (p *Poller) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := p.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			p.bot.StopReceivingUpdates()
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			p.handleUpdate(update)
		}
	}
}

func (p *Poller) handleUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		if update.CallbackQuery.From == nil || update.CallbackQuery.From.ID != p.ownerID {
			return
		}
		p.channel.HandleCallback(update.CallbackQuery.Data)
		return
	}

	if update.Message == nil {
		return
	}

	if update.Message.From == nil || update.Message.From.ID != p.ownerID {
		slog.Warn("telegram: ignoring message from non-owner", "user_id", update.Message.From.ID)
		return
	}

	if update.Message.Text == "" {
		return
	}

	p.channel.PushMessage(update.Message.Text)
}
