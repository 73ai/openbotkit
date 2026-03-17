package channel

import (
	"context"
	"testing"

	"github.com/73ai/openbotkit/source/scheduler"
)

func TestNewPusherUnsupported(t *testing.T) {
	_, err := NewPusher("unknown", scheduler.ChannelMeta{})
	if err == nil {
		t.Fatal("expected error for unsupported channel type")
	}
}

func TestTelegramPusherImplementsInterface(t *testing.T) {
	var _ Pusher = (*TelegramPusher)(nil)
}

func TestSlackPusherImplementsInterface(t *testing.T) {
	var _ Pusher = (*SlackPusher)(nil)
}

type mockPusher struct {
	messages []string
}

func (m *mockPusher) Push(_ context.Context, message string) error {
	m.messages = append(m.messages, message)
	return nil
}

func TestMockPusherImplementsInterface(t *testing.T) {
	var p Pusher = &mockPusher{}
	if err := p.Push(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
}
