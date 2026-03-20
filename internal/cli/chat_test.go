package cli

import (
	"strings"
	"testing"

	historysrc "github.com/73ai/openbotkit/service/history"
)

func TestOpenHistoryStore(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	s, err := openHistoryStore("test-session")
	if err != nil {
		t.Fatalf("openHistoryStore: %v", err)
	}

	// Save messages like the chat loop does.
	if err := s.SaveMessage("test-session", "user", "Hello"); err != nil {
		t.Fatalf("save user message: %v", err)
	}
	if err := s.SaveMessage("test-session", "assistant", "Hi there!"); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}

	// Verify messages were persisted.
	msgs, err := s.LoadSessionMessages("test-session", 100)
	if err != nil {
		t.Fatalf("load messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "Hello" {
		t.Errorf("msg[0] = %q/%q", msgs[0].Role, msgs[0].Content)
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "Hi there!" {
		t.Errorf("msg[1] = %q/%q", msgs[1].Role, msgs[1].Content)
	}
}

func TestOpenHistoryStore_ReturnsStoreType(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	s, err := openHistoryStore("test-session")
	if err != nil {
		t.Fatalf("openHistoryStore: %v", err)
	}
	// Verify it returned the right type.
	var _ *historysrc.Store = s
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if !strings.HasPrefix(id1, "obk-chat-") {
		t.Errorf("expected obk-chat- prefix, got %q", id1)
	}
	if id1 == id2 {
		t.Error("expected unique session IDs")
	}
}
