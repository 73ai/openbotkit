package whatsapp

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type sentMessage struct {
	jid  string
	text string
}

type mockSender struct {
	mu     sync.Mutex
	sent   []sentMessage
	notify chan struct{}
}

func (m *mockSender) SendText(_ context.Context, jid, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentMessage{jid: jid, text: text})
	if m.notify != nil {
		select {
		case m.notify <- struct{}{}:
		default:
		}
	}
	return nil
}

func TestSend_CallsSenderWithOwnerJID(t *testing.T) {
	ms := &mockSender{}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	if err := ch.Send("hello"); err != nil {
		t.Fatalf("send: %v", err)
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()
	if len(ms.sent) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(ms.sent))
	}
	if ms.sent[0].jid != "owner@s.whatsapp.net" {
		t.Fatalf("jid = %q, want owner@s.whatsapp.net", ms.sent[0].jid)
	}
	if ms.sent[0].text != "hello" {
		t.Fatalf("text = %q, want hello", ms.sent[0].text)
	}
}

func TestReceive_ReturnsIncomingMessage(t *testing.T) {
	ms := &mockSender{}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	ch.HandleIncoming("hello", "sender@s.whatsapp.net", "msg1")

	text, err := ch.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if text != "hello" {
		t.Fatalf("got %q, want hello", text)
	}
}

func TestReceive_EOFOnClose(t *testing.T) {
	ms := &mockSender{}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	ch.Close()

	_, err := ch.Receive()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestRequestApproval_SendsPromptAndBlocksUntilReply(t *testing.T) {
	ms := &mockSender{notify: make(chan struct{}, 1)}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

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
	case <-ms.notify:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for approval message")
	}

	ms.mu.Lock()
	if len(ms.sent) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(ms.sent))
	}
	if !strings.Contains(ms.sent[0].text, "Reply 1 to approve") {
		t.Fatalf("expected approval prompt, got %q", ms.sent[0].text)
	}
	ms.mu.Unlock()

	ch.HandleIncoming("1", "owner@s.whatsapp.net", "msg2")

	select {
	case approved := <-done:
		if !approved {
			t.Fatal("expected approval to be true")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for approval result")
	}
}

func TestRequestApproval_DenyWithReply2(t *testing.T) {
	ms := &mockSender{notify: make(chan struct{}, 1)}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	done := make(chan bool, 1)
	go func() {
		approved, _ := ch.RequestApproval("risky action")
		done <- approved
	}()

	select {
	case <-ms.notify:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	ch.HandleIncoming("2", "owner@s.whatsapp.net", "msg3")

	select {
	case approved := <-done:
		if approved {
			t.Fatal("expected denial")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

func TestHandleIncoming_RoutesApprovalWhenPending(t *testing.T) {
	ms := &mockSender{notify: make(chan struct{}, 1)}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	done := make(chan bool, 1)
	go func() {
		approved, _ := ch.RequestApproval("action")
		done <- approved
	}()

	select {
	case <-ms.notify:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	// Non-approval text goes to incoming
	ch.HandleIncoming("other text", "owner@s.whatsapp.net", "msg4")

	// Verify it appears on the incoming channel
	select {
	case msg := <-ch.incoming:
		if msg.text != "other text" {
			t.Fatalf("expected 'other text', got %q", msg.text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("non-approval message not queued")
	}

	// Now approve
	ch.HandleIncoming("1", "owner@s.whatsapp.net", "msg5")
	select {
	case approved := <-done:
		if !approved {
			t.Fatal("expected approval")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

func TestHandleIncoming_QueuesNormalMessageWhenNoApproval(t *testing.T) {
	ms := &mockSender{}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	ch.HandleIncoming("hello", "sender@s.whatsapp.net", "msg6")
	// "1" and "2" go to incoming when no approval pending
	ch.HandleIncoming("1", "sender@s.whatsapp.net", "msg7")
	ch.HandleIncoming("2", "sender@s.whatsapp.net", "msg8")

	for _, want := range []string{"hello", "1", "2"} {
		select {
		case msg := <-ch.incoming:
			if msg.text != want {
				t.Fatalf("got %q, want %q", msg.text, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}
}

func TestCancelPendingApproval_UnblocksRequestApproval(t *testing.T) {
	ms := &mockSender{notify: make(chan struct{}, 1)}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	done := make(chan bool, 1)
	go func() {
		approved, _ := ch.RequestApproval("risky action")
		done <- approved
	}()

	select {
	case <-ms.notify:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	ch.CancelPendingApproval()

	select {
	case approved := <-done:
		if approved {
			t.Fatal("expected denied after cancel")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

func TestCancelPendingApproval_NoOpWhenNoPending(t *testing.T) {
	ms := &mockSender{}
	ch := NewChannel(ms, "owner@s.whatsapp.net")
	ch.CancelPendingApproval() // should not panic
}

func TestSendLink_SendsTextPlusURL(t *testing.T) {
	ms := &mockSender{}
	ch := NewChannel(ms, "owner@s.whatsapp.net")

	if err := ch.SendLink("Open Google", "https://google.com"); err != nil {
		t.Fatalf("SendLink: %v", err)
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()
	if len(ms.sent) != 1 {
		t.Fatalf("expected 1 sent, got %d", len(ms.sent))
	}
	want := "Open Google\nhttps://google.com"
	if ms.sent[0].text != want {
		t.Fatalf("text = %q, want %q", ms.sent[0].text, want)
	}
}
