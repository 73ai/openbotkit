package telegram

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/internal/testutil"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/provider/gemini"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	"github.com/priyanshujain/openbotkit/store"
)

// TestIntegration_SessionWithRealLLM tests the full session lifecycle:
// message → agent with real Gemini API → response → history saved.
func TestIntegration_SessionWithRealLLM(t *testing.T) {
	key := testutil.RequireGeminiKey(t)

	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	// Create source dirs
	for _, src := range []string{"history", "user_memory"} {
		os.MkdirAll(filepath.Join(dir, src), 0700)
	}

	cfg := config.Default()

	// Migrate history DB
	histDB, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open history db: %v", err)
	}
	historysrc.Migrate(histDB)
	histDB.Close()

	// Migrate memory DB
	memDB, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.UserMemoryDataDSN()})
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	memory.Migrate(memDB)
	memDB.Close()

	// Create real Gemini provider
	p := gemini.New(key)
	model := "gemini-2.5-flash"

	bot := &mockBot{}
	ch := NewChannel(bot, 123)

	sm := NewSessionManager(cfg, ch, p, model)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send a simple message through the session manager
	sm.handleMessage(ctx, "What is 2 + 2? Reply with just the number.")

	// Verify the bot sent a response
	bot.mu.Lock()
	sentCount := len(bot.sent)
	bot.mu.Unlock()

	if sentCount == 0 {
		t.Fatal("expected bot to send a response")
	}

	// Check the response contains "4"
	bot.mu.Lock()
	lastMsg := bot.sent[sentCount-1]
	bot.mu.Unlock()

	msg, ok := lastMsg.(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", lastMsg)
	}
	if !strings.Contains(msg.Text, "4") {
		t.Errorf("expected response to contain '4', got: %q", msg.Text)
	}

	// Verify history was saved
	histDB, err = store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("reopen history db: %v", err)
	}
	defer histDB.Close()

	var count int
	err = histDB.QueryRow("SELECT COUNT(*) FROM history_messages").Scan(&count)
	if err != nil {
		t.Fatalf("count history messages: %v", err)
	}
	if count < 2 {
		t.Fatalf("expected at least 2 history messages (user + assistant), got %d", count)
	}
}

// TestIntegration_SessionWithMemoryInjection tests that user memories are injected
// into the system prompt when the agent processes a message.
func TestIntegration_SessionWithMemoryInjection(t *testing.T) {
	key := testutil.RequireGeminiKey(t)

	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	for _, src := range []string{"history", "user_memory"} {
		os.MkdirAll(filepath.Join(dir, src), 0700)
	}

	cfg := config.Default()

	// Migrate and seed memory DB
	memDB, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.UserMemoryDataDSN()})
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	memory.Migrate(memDB)
	memory.Add(memDB, "User's name is TestBot42", memory.CategoryIdentity, "manual", "")
	memDB.Close()

	// Migrate history DB
	histDB, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open history db: %v", err)
	}
	historysrc.Migrate(histDB)
	histDB.Close()

	p := gemini.New(key)
	model := "gemini-2.5-flash"

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := NewSessionManager(cfg, ch, p, model)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ask the agent something that requires memory
	sm.handleMessage(ctx, "What is my name? Reply with just the name.")

	bot.mu.Lock()
	sentCount := len(bot.sent)
	bot.mu.Unlock()

	if sentCount == 0 {
		t.Fatal("expected bot to send a response")
	}

	bot.mu.Lock()
	lastMsg := bot.sent[sentCount-1]
	bot.mu.Unlock()

	msg, ok := lastMsg.(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", lastMsg)
	}

	if !strings.Contains(msg.Text, "TestBot42") {
		t.Errorf("expected response to contain 'TestBot42' (from memory), got: %q", msg.Text)
	}
}

// TestIntegration_SessionWithToolUse tests that the agent can use tools
// (like bash) when processing a Telegram message.
func TestIntegration_SessionWithToolUse(t *testing.T) {
	key := testutil.RequireGeminiKey(t)

	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	for _, src := range []string{"history", "user_memory"} {
		os.MkdirAll(filepath.Join(dir, src), 0700)
	}

	cfg := config.Default()

	histDB, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.HistoryDataDSN()})
	if err != nil {
		t.Fatalf("open history db: %v", err)
	}
	historysrc.Migrate(histDB)
	histDB.Close()

	memDB, err := store.Open(store.Config{Driver: "sqlite", DSN: cfg.UserMemoryDataDSN()})
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	memory.Migrate(memDB)
	memDB.Close()

	p := gemini.New(key)
	model := "gemini-2.5-flash"

	bot := &mockBot{}
	ch := NewChannel(bot, 123)
	sm := NewSessionManager(cfg, ch, p, model)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Ask something that requires tool use (bash)
	sm.handleMessage(ctx, "Run the command 'echo telegram-integration-ok' and tell me the output.")

	bot.mu.Lock()
	sentCount := len(bot.sent)
	bot.mu.Unlock()

	if sentCount == 0 {
		t.Fatal("expected bot to send a response")
	}

	bot.mu.Lock()
	lastMsg := bot.sent[sentCount-1]
	bot.mu.Unlock()

	msg, ok := lastMsg.(tgbotapi.MessageConfig)
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", lastMsg)
	}

	if !strings.Contains(msg.Text, "telegram-integration-ok") {
		t.Errorf("expected response to contain 'telegram-integration-ok', got: %q", msg.Text)
	}
}
