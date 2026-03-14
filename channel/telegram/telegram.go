package telegram

import (
	"errors"
	"fmt"
	"io"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/priyanshujain/openbotkit/channel/tghtml"
)

// botSender abstracts the Telegram bot API for testing.
type botSender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	MakeRequest(endpoint string, params tgbotapi.Params) (*tgbotapi.APIResponse, error)
}

type approvalResponse struct {
	approved bool
	err      error
}

// incomingMessage carries both the text and the Telegram message ID.
type incomingMessage struct {
	text      string
	messageID int
}

// Channel implements channel.Channel for Telegram.
type Channel struct {
	bot      botSender
	chatID   int64
	incoming chan incomingMessage
	done     chan struct{}

	approvalMu sync.Mutex
	approvalCh chan approvalResponse
}

func NewChannel(bot botSender, chatID int64) *Channel {
	return &Channel{
		bot:      bot,
		chatID:   chatID,
		incoming: make(chan incomingMessage, 16),
		done:     make(chan struct{}),
	}
}

func (c *Channel) ChatID() int64 { return c.chatID }

func (c *Channel) Send(msg string) error {
	html := tghtml.Convert(msg)
	m := tgbotapi.NewMessage(c.chatID, html)
	m.ParseMode = "HTML"
	_, err := c.bot.Send(m)
	if isTelegramBadRequest(err) {
		m.Text = msg
		m.ParseMode = ""
		_, err = c.bot.Send(m)
	}
	return err
}

func (c *Channel) Receive() (string, error) {
	msg, ok := <-c.incoming
	if !ok {
		return "", io.EOF
	}
	return msg.text, nil
}

// ReceiveMessage returns the next incoming message with its Telegram message ID.
func (c *Channel) ReceiveMessage() (incomingMessage, error) {
	msg, ok := <-c.incoming
	if !ok {
		return incomingMessage{}, io.EOF
	}
	return msg, nil
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
func (c *Channel) PushMessage(text string, messageID int) {
	c.incoming <- incomingMessage{text: text, messageID: messageID}
}

// isTelegramBadRequest returns true if the error is a Telegram API 400 error
// (e.g. HTML parse failure). Other errors (network, rate limit) are not retried
// to avoid sending duplicate messages.
func isTelegramBadRequest(err error) bool {
	var apiErr *tgbotapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == 400
}

// Close shuts down the incoming channel.
func (c *Channel) Close() {
	close(c.incoming)
}
