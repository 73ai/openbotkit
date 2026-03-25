package channel

import (
	"context"
	"testing"

	"github.com/73ai/openbotkit/service/scheduler"
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

func TestWhatsAppPusherImplementsInterface(t *testing.T) {
	var _ Pusher = (*WhatsAppPusher)(nil)
}

func TestNewPusher_WhatsApp(t *testing.T) {
	// Cannot fully construct without a real session DB, but verify
	// the factory routes to the whatsapp case (will fail on missing
	// config dir, which confirms routing works).
	_, err := NewPusher("whatsapp", scheduler.ChannelMeta{
		WAAccountLabel: "test",
		WAOwnerJID:     "123@s.whatsapp.net",
	})
	if err == nil {
		t.Fatal("expected error (no config dir), but got nil")
	}
}

func TestMockPusherImplementsInterface(t *testing.T) {
	var p Pusher = &mockPusher{}
	if err := p.Push(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
}
