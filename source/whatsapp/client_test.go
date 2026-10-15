package whatsapp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow"
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

func drain(qr chan string) []string {
	var codes []string
	for code := range qr {
		codes = append(codes, code)
	}
	return codes
}

func TestForwardQREventsSuccess(t *testing.T) {
	qr := make(chan string, 2)
	err := forwardQREvents(qrEvents(whatsmeow.QRChannelItem{Event: "code", Code: "2@abc"}, whatsmeow.QRChannelSuccess), qr)
	if err != nil {
		t.Fatalf("success: %v", err)
	}
	if codes := drain(qr); len(codes) != 1 || codes[0] != "2@abc" {
		t.Fatalf("codes = %v", codes)
	}
}

func TestForwardQREventsFailures(t *testing.T) {
	code := whatsmeow.QRChannelItem{Event: "code", Code: "2@abc"}
	cases := []struct {
		name   string
		events <-chan whatsmeow.QRChannelItem
		want   string
	}{
		{"client outdated", qrEvents(code, whatsmeow.QRChannelClientOutdated), "err-client-outdated"},
		{"scanned without multidevice", qrEvents(code, whatsmeow.QRChannelScannedWithoutMultidevice), "err-scanned-without-multidevice"},
		{"timeout", qrEvents(code, whatsmeow.QRChannelTimeout), "timed out"},
		{"channel closed early", qrEvents(code), "ended before"},
	}
	for _, c := range cases {
		qr := make(chan string, 2)
		err := forwardQREvents(c.events, qr)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to mention %q", c.name, err, c.want)
		}
		if codes := drain(qr); len(codes) != 1 {
			t.Errorf("%s: forwarded codes = %v", c.name, codes)
		}
	}
}
