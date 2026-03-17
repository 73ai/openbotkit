package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	imsrc "github.com/73ai/openbotkit/source/imessage"
	"github.com/73ai/openbotkit/store"
)

func testServerWithIMessageDB(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cfg := config.Default()

	imDir := filepath.Join(dir, "imessage")
	if err := os.MkdirAll(imDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	dsn := cfg.IMessageDataDSN()
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := imsrc.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Close()

	return &Server{cfg: cfg}
}

func TestIMessagePush_Empty(t *testing.T) {
	s := testServerWithIMessageDB(t)

	req := httptest.NewRequest("POST", "/api/imessage/push", strings.NewReader("[]"))
	rec := httptest.NewRecorder()

	s.handleIMessagePush(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["saved"] != 0 {
		t.Errorf("saved = %d, want 0", resp["saved"])
	}
}

func TestIMessagePush_SavesMessages(t *testing.T) {
	s := testServerWithIMessageDB(t)

	body := `[{
		"guid": "msg-push-001",
		"apple_rowid": 1,
		"text": "Hello from bridge",
		"chat_guid": "iMessage;-;+1234567890",
		"sender_id": "+1234567890",
		"sender_service": "iMessage",
		"is_from_me": false,
		"is_read": true,
		"date": "2024-01-15T12:00:00Z",
		"chat_display_name": "John"
	}]`

	req := httptest.NewRequest("POST", "/api/imessage/push", strings.NewReader(body))
	rec := httptest.NewRecorder()

	s.handleIMessagePush(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["saved"] != 1 {
		t.Errorf("saved = %d, want 1", resp["saved"])
	}

	// Verify it was actually saved
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: s.cfg.IMessageDataDSN()})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	msg, err := imsrc.GetMessage(db, "msg-push-001")
	if err != nil {
		t.Fatalf("get message: %v", err)
	}
	if msg.Text != "Hello from bridge" {
		t.Errorf("text = %q, want %q", msg.Text, "Hello from bridge")
	}
}

func TestIMessagePush_InvalidBody(t *testing.T) {
	s := testServerWithIMessageDB(t)

	req := httptest.NewRequest("POST", "/api/imessage/push", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	s.handleIMessagePush(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
