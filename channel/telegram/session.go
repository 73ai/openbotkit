package telegram

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/priyanshujain/openbotkit/agent"
	"github.com/priyanshujain/openbotkit/agent/tools"
	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/provider"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	usagesrc "github.com/priyanshujain/openbotkit/source/usage"
	"github.com/priyanshujain/openbotkit/store"
)

const sessionTimeout = 15 * time.Minute

// SessionManager manages Telegram conversations with agent sessions.
type SessionManager struct {
	cfg          *config.Config
	channel      *Channel
	provider     provider.Provider
	providerName string
	model        string

	mu        sync.Mutex
	sessionID string
	timer     *time.Timer
	messages  []string // user messages collected in this session
}

func NewSessionManager(cfg *config.Config, ch *Channel, p provider.Provider, providerName, model string) *SessionManager {
	return &SessionManager{
		cfg:          cfg,
		channel:      ch,
		provider:     p,
		providerName: providerName,
		model:        model,
	}
}

func (sm *SessionManager) Run(ctx context.Context) {
	for {
		text, err := sm.channel.Receive()
		if err == io.EOF {
			sm.endSession()
			return
		}
		if err != nil {
			slog.Error("telegram session: receive error", "error", err)
			continue
		}

		sm.handleMessage(ctx, text)
	}
}

func (sm *SessionManager) handleMessage(ctx context.Context, text string) {
	sm.touchSession()

	a, recorder, err := sm.newAgent()
	if err != nil {
		slog.Error("telegram session: create agent", "error", err)
		sm.channel.Send(fmt.Sprintf("Error: %v", err))
		return
	}
	if recorder != nil {
		defer recorder.Close()
	}

	response, err := a.Run(ctx, text)
	if err != nil {
		slog.Error("telegram session: agent error", "error", err)
		sm.channel.Send(fmt.Sprintf("Error: %v", err))
		return
	}

	sm.channel.Send(response)

	sm.mu.Lock()
	sid := sm.sessionID
	sm.messages = append(sm.messages, text)
	sm.mu.Unlock()

	sm.saveHistory(sid, text, response)
}

// touchSession resets the inactivity timer, starting a new session if needed.
func (sm *SessionManager) touchSession() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.sessionID == "" {
		sm.sessionID = generateSessionID()
		sm.messages = nil
	}

	if sm.timer != nil {
		sm.timer.Stop()
	}
	sm.timer = time.AfterFunc(sessionTimeout, func() {
		sm.endSession()
	})
}

// endSession finalizes the current session and runs async memory extraction.
func (sm *SessionManager) endSession() {
	sm.mu.Lock()
	if sm.sessionID == "" {
		sm.mu.Unlock()
		return
	}
	if sm.timer != nil {
		sm.timer.Stop()
		sm.timer = nil
	}
	messages := sm.messages
	sm.sessionID = ""
	sm.messages = nil
	sm.mu.Unlock()

	if len(messages) > 0 {
		go sm.extractMemories(context.Background(), messages)
	}
}

func (sm *SessionManager) extractMemories(ctx context.Context, messages []string) {
	memDB, err := store.Open(store.Config{
		Driver: sm.cfg.UserMemory.Storage.Driver,
		DSN:    sm.cfg.UserMemoryDataDSN(),
	})
	if err != nil {
		slog.Error("telegram session: open memory db for extraction", "error", err)
		return
	}
	defer memDB.Close()

	if err := memory.Migrate(memDB); err != nil {
		slog.Error("telegram session: migrate memory db", "error", err)
		return
	}

	llm, err := sm.buildLLM()
	if err != nil {
		slog.Error("telegram session: build LLM for extraction", "error", err)
		return
	}

	candidates, err := memory.Extract(ctx, llm, messages)
	if err != nil {
		slog.Error("telegram session: extract memories", "error", err)
		return
	}

	if len(candidates) == 0 {
		return
	}

	result, err := memory.Reconcile(ctx, memDB, llm, candidates)
	if err != nil {
		slog.Error("telegram session: reconcile memories", "error", err)
		return
	}

	slog.Info("telegram session: memory extraction complete",
		"added", result.Added, "updated", result.Updated,
		"deleted", result.Deleted, "skipped", result.Skipped)
}

func (sm *SessionManager) buildLLM() (memory.LLM, error) {
	registry, err := provider.NewRegistry(sm.cfg.Models)
	if err != nil {
		return nil, fmt.Errorf("create provider registry: %w", err)
	}
	router := provider.NewRouter(registry, sm.cfg.Models)
	return &memory.RouterLLM{Router: router, Tier: provider.TierFast}, nil
}

func (sm *SessionManager) newAgent() (*agent.Agent, *usagesrc.Recorder, error) {
	toolReg := tools.NewStandardRegistry()
	toolReg.Register(tools.NewSubagentTool(tools.SubagentConfig{
		Provider:    sm.provider,
		Model:       sm.model,
		ToolFactory: tools.NewStandardRegistry,
		System:      "You are a focused sub-agent. Complete the given task and return a concise result.",
	}))

	identity := "You are a personal AI assistant powered by OpenBotKit, communicating via Telegram.\n"
	extras := "\nBe concise and direct. Skip filler phrases.\n" + sm.userMemoriesPrompt()
	blocks := tools.BuildSystemBlocks(identity, toolReg, extras)

	opts := []agent.Option{agent.WithSystemBlocks(blocks)}
	recorder := sm.openUsageRecorder()
	if recorder != nil {
		opts = append(opts, agent.WithUsageRecorder(recorder))
	}
	return agent.New(sm.provider, sm.model, toolReg, opts...), recorder, nil
}

func (sm *SessionManager) userMemoriesPrompt() string {
	memDB, err := store.Open(store.Config{
		Driver: sm.cfg.UserMemory.Storage.Driver,
		DSN:    sm.cfg.UserMemoryDataDSN(),
	})
	if err != nil {
		return ""
	}
	defer memDB.Close()
	if err := memory.Migrate(memDB); err != nil {
		return ""
	}
	memories, err := memory.List(memDB)
	if err != nil || len(memories) == 0 {
		return ""
	}
	return "\n" + memory.FormatForPrompt(memories)
}

func (sm *SessionManager) saveHistory(sessionID, userMsg, assistantMsg string) {
	histDB, err := store.Open(store.Config{
		Driver: sm.cfg.History.Storage.Driver,
		DSN:    sm.cfg.HistoryDataDSN(),
	})
	if err != nil {
		slog.Error("telegram session: open history db", "error", err)
		return
	}
	defer histDB.Close()

	if err := historysrc.Migrate(histDB); err != nil {
		slog.Error("telegram session: migrate history", "error", err)
		return
	}

	convID, err := historysrc.UpsertConversation(histDB, sessionID, "telegram")
	if err != nil {
		slog.Error("telegram session: create conversation", "error", err)
		return
	}

	if err := historysrc.SaveMessage(histDB, convID, "user", userMsg); err != nil {
		slog.Error("telegram session: save user message", "error", err)
	}
	if err := historysrc.SaveMessage(histDB, convID, "assistant", assistantMsg); err != nil {
		slog.Error("telegram session: save assistant message", "error", err)
	}
}

func (sm *SessionManager) openUsageRecorder() *usagesrc.Recorder {
	if err := config.EnsureSourceDir("usage"); err != nil {
		return nil
	}
	db, err := store.Open(store.Config{
		Driver: sm.cfg.Usage.Storage.Driver,
		DSN:    sm.cfg.UsageDataDSN(),
	})
	if err != nil {
		return nil
	}
	if err := usagesrc.Migrate(db); err != nil {
		db.Close()
		return nil
	}

	sm.mu.Lock()
	sid := sm.sessionID
	sm.mu.Unlock()

	return usagesrc.NewRecorder(db, sm.providerName, "telegram", sid)
}

func generateSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("tg-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("tg-%x", b[:])
}
