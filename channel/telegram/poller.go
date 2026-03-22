package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Poller receives updates from Telegram and routes them to the channel.
type Poller struct {
	bot         *tgbotapi.BotAPI
	ownerID     int64
	channel     *Channel
	interrupter Interrupter

	mu               sync.Mutex
	interruptPending bool
	interruptMsgID   int
	pendingMsg       *incomingMessage
}

func NewPoller(bot *tgbotapi.BotAPI, ownerID int64, ch *Channel, interrupter Interrupter) *Poller {
	return &Poller{bot: bot, ownerID: ownerID, channel: ch, interrupter: interrupter}
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
		case cb := <-p.channel.interruptCh:
			p.handleInterruptCallback(cb)
		case cb := <-p.channel.killTaskCh:
			p.handleKillTaskCallback(cb)
		}
	}
}

func (p *Poller) handleUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		if update.CallbackQuery.From == nil || update.CallbackQuery.From.ID != p.ownerID {
			return
		}
		slog.Info("telegram: callback received", "data", update.CallbackQuery.Data)
		p.channel.HandleCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
		return
	}

	if update.Message == nil {
		return
	}

	if update.Message.From == nil || update.Message.From.ID != p.ownerID {
		slog.Warn("telegram: ignoring message from non-owner", "user_id", update.Message.From.ID)
		return
	}

	text := update.Message.Text
	if text == "" {
		return
	}

	msgID := update.Message.MessageID

	if text == "/kill" {
		p.handleKill()
		return
	}

	if p.interrupter != nil && p.interrupter.IsAgentRunning() {
		p.mu.Lock()
		if p.interruptPending {
			// Already showing confirmation — queue message normally.
			p.mu.Unlock()
			p.channel.PushMessage(text, msgID)
			return
		}
		p.mu.Unlock()
		p.handleInterrupt(text, msgID)
		return
	}

	p.channel.PushMessage(text, msgID)
}

func (p *Poller) handleKill() {
	if p.interrupter == nil {
		p.sendText("Nothing running to kill.")
		return
	}

	if p.interrupter.IsAgentRunning() {
		if !p.interrupter.Kill() {
			p.sendText("Already finished.")
		}
		return
	}

	tasks := p.interrupter.RunningDelegateTasks()
	if len(tasks) == 0 {
		p.sendText("Nothing running to kill.")
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, t := range tasks {
		preview := t.Task
		if len(preview) > 40 {
			preview = preview[:40] + "..."
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("Kill: %s", preview),
				fmt.Sprintf("kill_task:%s", t.ID),
			),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Kill all", "kill_task:all"),
	))

	msg := tgbotapi.NewMessage(p.channel.chatID,
		fmt.Sprintf("%d background task(s) running. Kill which?", len(tasks)))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	p.channel.bot.Send(msg)
}

func (p *Poller) handleInterrupt(text string, msgID int) {
	p.mu.Lock()
	p.interruptPending = true
	p.pendingMsg = &incomingMessage{text: text, messageID: msgID}
	p.mu.Unlock()

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Stop", "interrupt:stop"),
			tgbotapi.NewInlineKeyboardButtonData("Continue", "interrupt:continue"),
		),
	)
	msg := tgbotapi.NewMessage(p.channel.chatID, "Want me to stop?")
	msg.ReplyMarkup = keyboard
	sentMsg, err := p.channel.bot.Send(msg)
	if err != nil {
		slog.Error("telegram: send interrupt prompt", "error", err)
		return
	}

	p.mu.Lock()
	p.interruptMsgID = sentMsg.MessageID
	p.mu.Unlock()
}

func (p *Poller) handleInterruptCallback(cb callbackData) {
	action := strings.TrimPrefix(cb.Data, "interrupt:")

	p.mu.Lock()
	pending := p.pendingMsg
	interruptMsgID := p.interruptMsgID
	p.interruptPending = false
	p.pendingMsg = nil
	p.interruptMsgID = 0
	p.mu.Unlock()

	answer := tgbotapi.NewCallback(cb.ID, "")
	p.channel.bot.Request(answer)

	if interruptMsgID != 0 {
		edit := tgbotapi.NewEditMessageReplyMarkup(
			p.channel.chatID, interruptMsgID,
			tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}},
		)
		p.channel.bot.Request(edit)
	}

	if p.interrupter != nil && !p.interrupter.IsAgentRunning() {
		p.sendText("Already finished.")
		if pending != nil {
			p.channel.PushMessage(pending.text, pending.messageID)
		}
		return
	}

	switch action {
	case "stop":
		if p.interrupter != nil {
			p.interrupter.Kill()
		}
		// Drop the pending message.
	case "continue":
		if pending != nil {
			p.channel.PushMessage(pending.text, pending.messageID)
		}
	}
}

func (p *Poller) handleKillTaskCallback(cb callbackData) {
	taskID := strings.TrimPrefix(cb.Data, "kill_task:")

	answer := tgbotapi.NewCallback(cb.ID, "")
	p.channel.bot.Request(answer)

	if p.interrupter == nil {
		return
	}

	if taskID == "all" {
		tasks := p.interrupter.RunningDelegateTasks()
		for _, t := range tasks {
			p.interrupter.KillDelegateTask(t.ID)
		}
		p.sendText(fmt.Sprintf("Killed %d task(s).", len(tasks)))
		return
	}

	if p.interrupter.KillDelegateTask(taskID) {
		p.sendText(fmt.Sprintf("Killed task %s.", taskID[:min(8, len(taskID))]))
	} else {
		p.sendText("Task not found or already finished.")
	}
}

func (p *Poller) sendText(text string) {
	msg := tgbotapi.NewMessage(p.channel.chatID, text)
	p.channel.bot.Send(msg)
}
