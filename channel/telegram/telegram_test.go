package telegram

import (
	"io"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type mockBot struct {
	mu     sync.Mutex
	sent   []tgbotapi.Chattable
	notify chan struct{}
}

func (m *mockBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, c)
	if m.notify != nil {
		select {
		case m.notify <- struct{}{}:
		default:
		}
	}
	return tgbotapi.Message{}, nil
}

func TestSend_FormatsMarkdown(t *testing.T) {
	bot := &mockBot{}
	ch := NewChannel(bot, 123)

	if err := ch.Send("hello **world**"); err != nil {
		t.Fatalf("send: %v", err)
	}

	bot.mu.Lock()
	defer bot.mu.Unlock()
	if len(bot.sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(bot.sent))
	}

	msg, ok := bot.sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[0])
	}
	if msg.ParseMode != "Markdown" {
		t.Fatalf("expected Markdown parse mode, got %q", msg.ParseMode)
	}
	if msg.Text != "hello **world**" {
		t.Fatalf("expected 'hello **world**', got %q", msg.Text)
	}
}

func TestReceive_ReturnsIncomingMessage(t *testing.T) {
	bot := &mockBot{}
	ch := NewChannel(bot, 123)

	ch.PushMessage("hello")

	text, err := ch.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if text != "hello" {
		t.Fatalf("expected 'hello', got %q", text)
	}
}

func TestReceive_EOFOnClose(t *testing.T) {
	bot := &mockBot{}
	ch := NewChannel(bot, 123)

	ch.Close()

	_, err := ch.Receive()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestSendLink_SendsURLButton(t *testing.T) {
	bot := &mockBot{}
	ch := NewChannel(bot, 123)

	if err := ch.SendLink("Open Google", "https://google.com"); err != nil {
		t.Fatalf("SendLink: %v", err)
	}

	bot.mu.Lock()
	defer bot.mu.Unlock()
	if len(bot.sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(bot.sent))
	}
	msg, ok := bot.sent[0].(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sent[0])
	}
	if msg.ReplyMarkup == nil {
		t.Fatal("expected inline keyboard markup")
	}
	kb, ok := msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("expected InlineKeyboardMarkup, got %T", msg.ReplyMarkup)
	}
	if len(kb.InlineKeyboard) != 1 || len(kb.InlineKeyboard[0]) != 1 {
		t.Fatal("expected 1 row with 1 button")
	}
	btn := kb.InlineKeyboard[0][0]
	if btn.URL == nil || *btn.URL != "https://google.com" {
		t.Errorf("button URL = %v, want https://google.com", btn.URL)
	}
}

func TestRequestApproval_SendsKeyboard(t *testing.T) {
	bot := &mockBot{notify: make(chan struct{}, 1)}
	ch := NewChannel(bot, 123)

	done := make(chan bool, 1)
	go func() {
		approved, err := ch.RequestApproval("delete all files")
		if err != nil {
			t.Errorf("approval: %v", err)
			return
		}
		done <- approved
	}()

	select {
	case <-bot.notify:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for approval message")
	}

	bot.mu.Lock()
	msg, ok := bot.sent[0].(tgbotapi.MessageConfig)
	bot.mu.Unlock()
	if !ok {
		t.Fatalf("expected MessageConfig")
	}
	if msg.ReplyMarkup == nil {
		t.Fatal("expected inline keyboard markup")
	}

	// Simulate approve callback
	ch.HandleCallback("approve")

	approved := <-done
	if !approved {
		t.Fatal("expected approval to be true")
	}
}

func TestOwnerFilter_RejectsNonOwner(t *testing.T) {
	bot := &mockBot{}
	ch := NewChannel(bot, 123)

	p := &Poller{
		bot:     nil, // Not used in handleUpdate
		ownerID: 123,
		channel: ch,
	}

	// Non-owner message should be dropped
	p.handleUpdate(tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{ID: 999},
			Text: "should be ignored",
		},
	})

	// Owner message should be accepted
	p.handleUpdate(tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{ID: 123},
			Text: "hello owner",
		},
	})

	text, err := ch.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if text != "hello owner" {
		t.Fatalf("expected 'hello owner', got %q", text)
	}
}
