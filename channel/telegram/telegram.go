package telegram

import (
	"fmt"
	"io"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// botSender abstracts the Telegram bot API for testing.
type botSender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
}

type approvalResponse struct {
	approved bool
	err      error
}

// Channel implements channel.Channel for Telegram.
type Channel struct {
	bot      botSender
	chatID   int64
	incoming chan string
	done     chan struct{}

	approvalMu sync.Mutex
	approvalCh chan approvalResponse
}

func NewChannel(bot botSender, chatID int64) *Channel {
	return &Channel{
		bot:      bot,
		chatID:   chatID,
		incoming: make(chan string, 16),
		done:     make(chan struct{}),
	}
}

func (c *Channel) ChatID() int64 { return c.chatID }

func (c *Channel) Send(msg string) error {
	m := tgbotapi.NewMessage(c.chatID, msg)
	m.ParseMode = "Markdown"
	_, err := c.bot.Send(m)
	return err
}

func (c *Channel) Receive() (string, error) {
	text, ok := <-c.incoming
	if !ok {
		return "", io.EOF
	}
	return text, nil
}

func (c *Channel) SendLink(text string, url string) error {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(text, url),
		),
	)
	msg := tgbotapi.NewMessage(c.chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := c.bot.Send(msg)
	return err
}

func (c *Channel) RequestApproval(action string) (bool, error) {
	c.approvalMu.Lock()
	c.approvalCh = make(chan approvalResponse, 1)
	c.approvalMu.Unlock()

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Approve", "approve"),
			tgbotapi.NewInlineKeyboardButtonData("Deny", "deny"),
		),
	)

	msg := tgbotapi.NewMessage(c.chatID, fmt.Sprintf("Approve action?\n\n%s", action))
	msg.ReplyMarkup = keyboard
	if _, err := c.bot.Send(msg); err != nil {
		return false, fmt.Errorf("send approval request: %w", err)
	}

	resp := <-c.approvalCh
	return resp.approved, resp.err
}

// HandleCallback processes an inline keyboard callback.
func (c *Channel) HandleCallback(data string) {
	c.approvalMu.Lock()
	ch := c.approvalCh
	c.approvalMu.Unlock()

	if ch != nil {
		ch <- approvalResponse{approved: data == "approve"}
	}
}

// PushMessage enqueues an incoming message from the poller.
func (c *Channel) PushMessage(text string) {
	c.incoming <- text
}

// Close shuts down the incoming channel.
func (c *Channel) Close() {
	close(c.incoming)
}
