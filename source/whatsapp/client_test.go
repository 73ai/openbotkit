package whatsapp

import (
	"context"
	"go.mau.fi/whatsmeow"
	"path/filepath"
	"testing"
)

func TestNewClient_CreatesSessionDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "session.db")

	ctx := context.Background()
	client, err := NewClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("client is nil")
	}

	if client.IsAuthenticated() {
		t.Error("new client should not be authenticated")
	}
}

func TestClient_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "session.db")

	ctx := context.Background()
	client, err := NewClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Close should not panic and should release the SQLite connection.
	client.Close()

	// After Close, creating a new client on the same DB should succeed
	// (proves the SQLite lock was released).
	client2, err := NewClient(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewClient after Close failed: %v", err)
	}
	client2.Close()
}

func qrEvents(items ...whatsmeow.QRChannelItem) <-chan whatsmeow.QRChannelItem {
	ch := make(chan whatsmeow.QRChannelItem, len(items))
	for _, it := range items {
		ch <- it
	}
	close(ch)
	return ch
}

func TestForwardQREventsSuccess(t *testing.T) {
	qr := make(chan string, 2)
	err := forwardQREvents(qrEvents(whatsmeow.QRChannelItem{Event: "code", Code: "abc"}, whatsmeow.QRChannelSuccess), qr)
	if err != nil {
		t.Fatalf("success: %v", err)
	}
	if code := <-qr; code != "abc" {
		t.Fatalf("code = %q", code)
	}
}

func TestForwardQREventsFailures(t *testing.T) {
	cases := map[string]<-chan whatsmeow.QRChannelItem{
		"outdated": qrEvents(whatsmeow.QRChannelClientOutdated),
		"timeout":  qrEvents(whatsmeow.QRChannelItem{Event: "code", Code: "abc"}, whatsmeow.QRChannelTimeout),
		"closed":   qrEvents(),
	}
	for name, ch := range cases {
		if err := forwardQREvents(ch, make(chan string, 2)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
