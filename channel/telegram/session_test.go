package telegram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	"github.com/priyanshujain/openbotkit/store"
)

type stubProvider struct{}

func (s *stubProvider) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "stub response"}},
		StopReason: provider.StopEndTurn,
	}, nil
}

func (s *stubProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func setupTestEnv(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)
	for _, src := range []string{"history", "user_memory"} {
		os.MkdirAll(filepath.Join(dir, src), 0700)
	}
	return config.Default()
}

func seedHistory(t *testing.T, cfg *config.Config, sessionID string, msgs []historysrc.Message) {
	t.Helper()
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open history db: %v", err)
	}
	defer db.Close()
	if err := historysrc.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	convID, err := historysrc.UpsertConversation(db, sessionID, "telegram")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	for _, m := range msgs {
		if err := historysrc.SaveMessage(db, convID, m.Role, m.Content); err != nil {
			t.Fatalf("save message: %v", err)
		}
	}
}

func TestRestoreSession_RecentSession(t *testing.T) {
	cfg := setupTestEnv(t)
	seedHistory(t, cfg, "tg-abc", []historysrc.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
		{Role: "user", Content: "how are you"},
		{Role: "assistant", Content: "fine"},
	})

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.touchSession()

	if sm.sessionID != "tg-abc" {
		t.Fatalf("expected restored session tg-abc, got %q", sm.sessionID)
	}
	if len(sm.history) != 4 {
		t.Fatalf("expected 4 history messages, got %d", len(sm.history))
	}
	if sm.history[0].Content[0].Text != "hello" {
		t.Errorf("history[0] = %q, want 'hello'", sm.history[0].Content[0].Text)
	}
	if len(sm.messages) != 2 {
		t.Fatalf("expected 2 user messages, got %d", len(sm.messages))
	}
}

func TestRestoreSession_ExpiredSession(t *testing.T) {
	cfg := setupTestEnv(t)

	// Seed a session and manually set it to 1 hour ago
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	historysrc.Migrate(db)
	historysrc.UpsertConversation(db, "tg-old", "telegram")
	db.Exec("UPDATE history_conversations SET updated_at = datetime('now', '-1 hour') WHERE session_id = 'tg-old'")
	historysrc.SaveMessage(db, 1, "user", "old msg")
	db.Close()

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.touchSession()

	if sm.sessionID == "tg-old" {
		t.Fatal("should not restore expired session")
	}
	if sm.sessionID == "" {
		t.Fatal("should have generated new session ID")
	}
	if len(sm.history) != 0 {
		t.Fatalf("expected empty history, got %d", len(sm.history))
	}
}

func TestRestoreSession_EmptyDB(t *testing.T) {
	cfg := setupTestEnv(t)

	// Migrate but don't seed
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	historysrc.Migrate(db)
	db.Close()

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.touchSession()

	if sm.sessionID == "" {
		t.Fatal("should have generated new session ID")
	}
	if len(sm.history) != 0 {
		t.Fatalf("expected empty history, got %d", len(sm.history))
	}
}

func sentTexts(bot *mockBot) []string {
	bot.mu.Lock()
	defer bot.mu.Unlock()
	var texts []string
	for _, c := range bot.sent {
		if msg, ok := c.(tgbotapi.MessageConfig); ok {
			texts = append(texts, msg.Text)
		}
	}
	return texts
}

func TestStartCommand_ResetsSession(t *testing.T) {
	cfg := setupTestEnv(t)
	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}

	sm.sessionID = "tg-abc"
	sm.history = []provider.Message{
		provider.NewTextMessage(provider.RoleUser, "hello"),
		provider.NewTextMessage(provider.RoleAssistant, "hi"),
	}
	sm.messages = []string{"hello"}

	sm.handleMessage(context.Background(), "/start")

	if sm.sessionID != "" {
		t.Fatalf("expected cleared sessionID, got %q", sm.sessionID)
	}
	if sm.history != nil {
		t.Fatalf("expected nil history, got %d messages", len(sm.history))
	}

	texts := sentTexts(bot)
	found := false
	for _, txt := range texts {
		if strings.Contains(txt, "Session reset") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected reset confirmation message, got: %v", texts)
	}
}

func TestStartCommand_WithSuffix(t *testing.T) {
	cfg := setupTestEnv(t)
	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := &SessionManager{cfg: cfg, channel: ch}
	sm.sessionID = "tg-xyz"

	sm.handleMessage(context.Background(), "/start now")

	if sm.sessionID != "" {
		t.Fatalf("expected cleared sessionID, got %q", sm.sessionID)
	}
	texts := sentTexts(bot)
	found := false
	for _, txt := range texts {
		if strings.Contains(txt, "Session reset") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected reset confirmation, got: %v", texts)
	}
}

func TestNormalMessage_DoesNotReset(t *testing.T) {
	cfg := setupTestEnv(t)
	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sp := &stubProvider{}
	sm := &SessionManager{
		cfg:          cfg,
		channel:      ch,
		provider:     sp,
		model:        "test-model",
		fastProvider: sp,
		fastModel:    "test-model",
	}
	sm.sessionID = "tg-existing"
	sm.history = []provider.Message{
		provider.NewTextMessage(provider.RoleUser, "prior"),
	}
	sm.messages = []string{"prior"}

	sm.handleMessage(context.Background(), "hello")

	// Session was NOT reset — it should still be "tg-existing"
	if sm.sessionID != "tg-existing" {
		t.Fatalf("session should not be reset, got %q", sm.sessionID)
	}
	texts := sentTexts(bot)
	for _, txt := range texts {
		if strings.Contains(txt, "Session reset") {
			t.Fatal("normal message should not trigger session reset")
		}
	}
	// Should have received a response from the stub provider
	found := false
	for _, txt := range texts {
		if strings.Contains(txt, "stub response") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected stub response, got: %v", texts)
	}
}

func TestResolveContextWindow_FromConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Models = &config.ModelsConfig{ContextWindow: 150000}
	sm := &SessionManager{cfg: cfg, model: "claude-opus-4-6"}

	if got := sm.resolveContextWindow(); got != 150000 {
		t.Fatalf("expected 150000 from config, got %d", got)
	}
}

func TestResolveContextWindow_FromModelLookup(t *testing.T) {
	cfg := config.Default()
	sm := &SessionManager{cfg: cfg, model: "gemini-2.5-flash"}

	if got := sm.resolveContextWindow(); got != 1048576 {
		t.Fatalf("expected 1048576 from model lookup, got %d", got)
	}
}

func TestResolveContextWindow_Fallback(t *testing.T) {
	cfg := config.Default()
	sm := &SessionManager{cfg: cfg, model: "unknown-model"}

	if got := sm.resolveContextWindow(); got != 200000 {
		t.Fatalf("expected 200000 fallback, got %d", got)
	}
}

func TestResolveCompactionThreshold_FromConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Models = &config.ModelsConfig{CompactionThreshold: 0.25}
	sm := &SessionManager{cfg: cfg}

	if got := sm.resolveCompactionThreshold(); got != 0.25 {
		t.Fatalf("expected 0.25 from config, got %f", got)
	}
}

func TestResolveCompactionThreshold_Default(t *testing.T) {
	cfg := config.Default()
	sm := &SessionManager{cfg: cfg}

	if got := sm.resolveCompactionThreshold(); got != 0.30 {
		t.Fatalf("expected 0.30 default, got %f", got)
	}
}
