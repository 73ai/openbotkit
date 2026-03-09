package telegram

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/priyanshujain/openbotkit/agent"
	"github.com/priyanshujain/openbotkit/agent/tools"
	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/provider"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	"github.com/priyanshujain/openbotkit/store"
)

// SessionManager manages Telegram conversations with agent sessions.
type SessionManager struct {
	cfg      *config.Config
	channel  *Channel
	provider provider.Provider
	model    string
}

func NewSessionManager(cfg *config.Config, ch *Channel, p provider.Provider, model string) *SessionManager {
	return &SessionManager{
		cfg:      cfg,
		channel:  ch,
		provider: p,
		model:    model,
	}
}

func (sm *SessionManager) Run(ctx context.Context) {
	for {
		text, err := sm.channel.Receive()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("telegram session: receive error: %v", err)
			continue
		}

		go sm.handleMessage(ctx, text)
	}
}

func (sm *SessionManager) handleMessage(ctx context.Context, text string) {
	a, err := sm.newAgent()
	if err != nil {
		log.Printf("telegram session: create agent: %v", err)
		sm.channel.Send(fmt.Sprintf("Error: %v", err))
		return
	}

	response, err := a.Run(ctx, text)
	if err != nil {
		log.Printf("telegram session: agent error: %v", err)
		sm.channel.Send(fmt.Sprintf("Error: %v", err))
		return
	}

	sm.channel.Send(response)
	sm.saveHistory(text, response)
}

func (sm *SessionManager) newAgent() (*agent.Agent, error) {
	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.FileWriteTool{})
	toolReg.Register(&tools.FileEditTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})

	system := sm.buildSystemPrompt()
	return agent.New(sm.provider, sm.model, toolReg, agent.WithSystem(system)), nil
}

func (sm *SessionManager) buildSystemPrompt() string {
	system := `You are a personal AI assistant powered by OpenBotKit, communicating via Telegram. You help users with email, messaging, notes, and other tasks.

You have core tools available: bash (run commands), file_read, file_write, file_edit, load_skills, search_skills.

To handle domain-specific tasks (email, WhatsApp, notes, etc.), first use search_skills to find relevant skills, then use load_skills to get detailed instructions.
`

	idx, err := skills.LoadIndex()
	if err == nil && len(idx.Skills) > 0 {
		system += "\nAvailable skills:\n"
		for _, s := range idx.Skills {
			system += fmt.Sprintf("- %s: %s\n", s.Name, s.Description)
		}
	}

	// Inject user memories if available.
	memDB, err := store.Open(store.Config{
		Driver: sm.cfg.UserMemory.Storage.Driver,
		DSN:    sm.cfg.UserMemoryDataDSN(),
	})
	if err == nil {
		defer memDB.Close()
		if err := memory.Migrate(memDB); err == nil {
			memories, err := memory.List(memDB)
			if err == nil && len(memories) > 0 {
				system += "\n" + memory.FormatForPrompt(memories)
			}
		}
	}

	return system
}

func (sm *SessionManager) saveHistory(userMsg, assistantMsg string) {
	histDB, err := store.Open(store.Config{
		Driver: sm.cfg.History.Storage.Driver,
		DSN:    sm.cfg.HistoryDataDSN(),
	})
	if err != nil {
		log.Printf("telegram session: open history db: %v", err)
		return
	}
	defer histDB.Close()

	if err := historysrc.Migrate(histDB); err != nil {
		log.Printf("telegram session: migrate history: %v", err)
		return
	}

	sessionID := generateSessionID()
	convID, err := historysrc.UpsertConversation(histDB, sessionID, "telegram")
	if err != nil {
		log.Printf("telegram session: create conversation: %v", err)
		return
	}

	historysrc.SaveMessage(histDB, convID, "user", userMsg)
	historysrc.SaveMessage(histDB, convID, "assistant", assistantMsg)
}

func generateSessionID() string {
	var b [16]byte
	rand.Read(b[:])
	return fmt.Sprintf("tg-%x", b[:])
}
