package cli

import (
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	historysrc "github.com/73ai/openbotkit/service/history"
)

func TestOpenHistoryDB(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cfg := config.Default()

	db, convID, err := openHistoryDB(cfg, "test-session")
	if err != nil {
		t.Fatalf("openHistoryDB: %v", err)
	}
	defer db.Close()

	if convID == 0 {
		t.Fatal("expected non-zero conversation ID")
	}

	// Save messages like the chat loop does.
	if err := historysrc.SaveMessage(db, convID, "user", "Hello"); err != nil {
		t.Fatalf("save user message: %v", err)
	}
	if err := historysrc.SaveMessage(db, convID, "assistant", "Hi there!"); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}

	// Verify messages were persisted.
	var msgCount int
	err = db.QueryRow(
		db.Rebind("SELECT COUNT(*) FROM history_messages WHERE conversation_id = ?"),
		convID,
	).Scan(&msgCount)
	if err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if msgCount != 2 {
		t.Fatalf("expected 2 messages, got %d", msgCount)
	}
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
