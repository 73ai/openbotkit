package telegram

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/audit"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/internal/skills"
	"github.com/73ai/openbotkit/service/memory"
	"github.com/73ai/openbotkit/oauth/google"
	"github.com/73ai/openbotkit/provider"
	historysrc "github.com/73ai/openbotkit/service/history"
	"github.com/73ai/openbotkit/service/scheduler"
	slacksrc "github.com/73ai/openbotkit/source/slack"
	usagesrc "github.com/73ai/openbotkit/service/usage"
	"github.com/73ai/openbotkit/store"
)

const sessionTimeout = 15 * time.Minute

// SessionManager manages Telegram conversations with agent sessions.
type SessionManager struct {
	cfg          *config.Config
	channel      *Channel
	provider     provider.Provider
	providerName string
	model        string

	interactor  tools.Interactor
	scopeWaiter *google.ScopeWaiter
	tokenBridge *tools.TokenBridge
	googleAuth  *google.Google
	account     string
	manifest    *skills.Manifest

	taskTracker *tools.TaskTracker

	webSearch    tools.WebSearcher
	fastProvider provider.Provider
	fastModel    string
	nanoProvider provider.Provider
	nanoModel    string

	mu        sync.Mutex
	sessionID string
	timer     *time.Timer
	messages  []string           // user messages collected in this session
	history   []provider.Message // conversation history for context
}

// SessionManagerDeps holds optional GWS-related dependencies.
type SessionManagerDeps struct {
	Interactor  tools.Interactor
	ScopeWaiter *google.ScopeWaiter
	TokenBridge *tools.TokenBridge
	GoogleAuth  *google.Google
	Account     string
}

func NewSessionManager(cfg *config.Config, ch *Channel, p provider.Provider, providerName, model string, deps ...SessionManagerDeps) *SessionManager {
	sm := &SessionManager{
		cfg:          cfg,
		channel:      ch,
		provider:     p,
		providerName: providerName,
		model:        model,
		taskTracker:  tools.NewTaskTracker(),
	}
	if len(deps) > 0 {
		d := deps[0]
		sm.interactor = d.Interactor
		sm.scopeWaiter = d.ScopeWaiter
		sm.tokenBridge = d.TokenBridge
		sm.googleAuth = d.GoogleAuth
		sm.account = d.Account
	}
	if sm.gwsEnabled() {
		sm.manifest, _ = skills.LoadManifest()
	}
	sm.initWebSearch()
	return sm
}

func (sm *SessionManager) Run(ctx context.Context) {
	for {
		msg, err := sm.channel.ReceiveMessage()
		if err == io.EOF {
			sm.endSession()
			return
		}
		if err != nil {
			slog.Error("telegram session: receive error", "error", err)
			continue
		}

		sm.handleMessage(ctx, msg.text, msg.messageID)
	}
}

func (sm *SessionManager) handleMessage(ctx context.Context, text string, messageID int) {
	if strings.HasPrefix(text, "/start") {
		sm.endSession()
		sm.channel.Send("Session reset. Starting fresh.")
		return
	}

	sm.touchSession()

	sm.mu.Lock()
	priorHistory := make([]provider.Message, len(sm.history))
	copy(priorHistory, sm.history)
	sm.mu.Unlock()

	fb := newProcessingFeedback(
		sm.channel.bot, sm.channel.chatID, messageID,
		text, sm.nanoProvider, sm.nanoModel,
	)
	fb.Start(ctx)

	a, recorder, auditLogger, err := sm.newAgent(priorHistory, fb.Signal)
	if err != nil {
		fb.Stop()
		slog.Error("telegram session: create agent", "error", err)
		sm.channel.Send(fmt.Sprintf("Error: %v", err))
		return
	}
	if recorder != nil {
		defer recorder.Close()
	}
	if auditLogger != nil {
		defer auditLogger.Close()
	}

	response, err := a.Run(ctx, text)
	fb.Stop()
	if err != nil {
		slog.Error("telegram session: agent error", "error", err)
		sm.channel.Send(fmt.Sprintf("Error: %v", err))
		return
	}

	if strings.TrimSpace(response) == "" {
		slog.Warn("telegram session: empty response from agent")
		response = "I wasn't able to generate a response. Could you try again?"
	}

	if err := sm.channel.Send(response); err != nil {
		slog.Error("telegram session: send response", "error", err)
	}

	sm.mu.Lock()
	sid := sm.sessionID
	sm.messages = append(sm.messages, text)
	sm.history = append(sm.history,
		provider.NewTextMessage(provider.RoleUser, text),
		provider.NewTextMessage(provider.RoleAssistant, response),
	)
	sm.mu.Unlock()

	sm.saveHistory(sid, text, response)
}

func (sm *SessionManager) restoreSession() bool {
	histDB, err := store.Open(store.Config{
		Driver: sm.cfg.History.Storage.Driver,
		DSN:    sm.cfg.HistoryDataDSN(),
	})
	if err != nil {
		slog.Warn("telegram session: open history db for restore", "error", err)
		return false
	}
	defer histDB.Close()

	if err := historysrc.Migrate(histDB); err != nil {
		slog.Warn("telegram session: migrate history for restore", "error", err)
		return false
	}

	recent, err := historysrc.LoadRecentSession(histDB, "telegram", sessionTimeout)
	if err != nil {
		slog.Warn("telegram session: load recent session", "error", err)
		return false
	}
	if recent == nil {
		return false
	}

	msgs, err := historysrc.LoadSessionMessages(histDB, recent.SessionID, 100)
	if err != nil {
		slog.Warn("telegram session: load session messages", "error", err)
		return false
	}
	if len(msgs) == 0 {
		return false
	}

	sm.sessionID = recent.SessionID
	sm.messages = nil
	sm.history = nil
	for _, m := range msgs {
		role := provider.RoleUser
		if m.Role == "assistant" {
			role = provider.RoleAssistant
		}
		sm.history = append(sm.history, provider.NewTextMessage(role, m.Content))
		if m.Role == "user" {
			sm.messages = append(sm.messages, m.Content)
		}
	}
	return true
}

// touchSession resets the inactivity timer, starting a new session if needed.
func (sm *SessionManager) touchSession() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.sessionID == "" {
		if !sm.restoreSession() {
			sm.sessionID = generateSessionID()
			sm.messages = nil
		}
		if err := config.EnsureScratchDir(sm.sessionID); err != nil {
			slog.Warn("scratch dir creation failed", "error", err)
		}
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
	sid := sm.sessionID
	sm.sessionID = ""
	sm.messages = nil
	sm.history = nil
	sm.mu.Unlock()

	config.CleanScratch(sid)

	// Mark session ended in DB so restoreSession won't revive it.
	if histDB, err := store.Open(store.Config{
		Driver: sm.cfg.History.Storage.Driver,
		DSN:    sm.cfg.HistoryDataDSN(),
	}); err == nil {
		historysrc.EndSession(histDB, sid)
		histDB.Close()
	}

	if len(messages) > 0 {
		go sm.extractMemories(context.Background(), messages)
	}
}

func (sm *SessionManager) extractMemories(ctx context.Context, messages []string) {
	if sm.cfg.Models == nil || sm.cfg.Models.Default == "" {
		return
	}

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

	extractLLM, reconcileLLM, err := sm.buildMemoryLLMs()
	if err != nil {
		slog.Error("telegram session: build LLM for extraction", "error", err)
		return
	}

	candidates, err := memory.Extract(ctx, extractLLM, messages)
	if err != nil {
		slog.Error("telegram session: extract memories", "error", err)
		return
	}

	if len(candidates) == 0 {
		return
	}

	result, err := memory.Reconcile(ctx, memDB, reconcileLLM, candidates)
	if err != nil {
		slog.Error("telegram session: reconcile memories", "error", err)
		return
	}

	slog.Info("telegram session: memory extraction complete",
		"added", result.Added, "updated", result.Updated,
		"deleted", result.Deleted, "skipped", result.Skipped)
}

func (sm *SessionManager) buildMemoryLLMs() (extract memory.LLM, reconcile memory.LLM, err error) {
	registry, err := provider.NewRegistry(sm.cfg.Models)
	if err != nil {
		return nil, nil, fmt.Errorf("create provider registry: %w", err)
	}
	router := provider.NewRouter(registry, sm.cfg.Models)
	extractLLM := &memory.RouterLLM{Router: router, Tier: provider.TierFast}
	reconcileLLM := &memory.RouterLLM{Router: router, Tier: provider.TierNano}
	return extractLLM, reconcileLLM, nil
}

func (sm *SessionManager) resolveContextWindow() int {
	if sm.cfg.Models != nil && sm.cfg.Models.ContextWindow > 0 {
		return sm.cfg.Models.ContextWindow
	}
	if w := provider.DefaultContextWindow(sm.model); w > 0 {
		return w
	}
	return 200000
}

func (sm *SessionManager) resolveCompactionThreshold() float64 {
	if sm.cfg.Models != nil && sm.cfg.Models.CompactionThreshold > 0 {
		return sm.cfg.Models.CompactionThreshold
	}
	return 0.30
}

func (sm *SessionManager) gwsEnabled() bool {
	return sm.cfg.Integrations != nil && sm.cfg.Integrations.GWS != nil && sm.cfg.Integrations.GWS.Enabled
}

func (sm *SessionManager) newAgent(history []provider.Message, onToolStart func(string)) (*agent.Agent, *usagesrc.Recorder, *audit.Logger, error) {
	toolReg := tools.NewStandardRegistry(sm.interactor, nil)

	sm.mu.Lock()
	sid := sm.sessionID
	sm.mu.Unlock()
	if sid != "" {
		toolReg.SetScratchDir(config.ScratchDir(sid))
	}

	al := sm.openAuditLogger()
	if al != nil {
		toolReg.SetAudit(al, "telegram")
	}

	if sm.gwsEnabled() && sm.interactor != nil {
		toolReg.Register(tools.NewGWSExecuteTool(tools.GWSToolConfig{
			Interactor:   sm.interactor,
			ScopeChecker: &tools.GoogleScopeChecker{TokenDBPath: sm.cfg.GoogleTokenDBPath()},
			Bridge:       sm.tokenBridge,
			ScopeWaiter:  sm.scopeWaiter,
			Google:       sm.googleAuth,
			Account:      sm.account,
			Manifest:     sm.manifest,
			Runner:       tools.NewGWSRunner(),
		}))
	}

	sm.registerSlackTools(toolReg)
	sm.registerDelegateTool(toolReg)
	sm.registerScheduleTools(toolReg)
	sm.registerWebTools(toolReg)

	scratchDir := config.ScratchDir(sid)
	toolReg.Register(tools.NewSubagentTool(tools.SubagentConfig{
		Provider: sm.provider,
		Model:    sm.model,
		ToolFactory: func() *tools.Registry {
			r := tools.NewStandardRegistry(nil, nil)
			r.SetScratchDir(scratchDir)
			return r
		},
		System: "You are a focused sub-agent. Complete the given task and return a concise result.",
	}))

	identity := "You are a personal AI assistant communicating via Telegram.\n"
	extras := `
Talk like a helpful human, not a robot. Be casual, warm, and direct.
- Answer the question first. Don't restate what the user already knows (like today's date).
- Keep it short — one or two sentences when possible.
- Summarize by default, but include specifics when the user asks for details.
- Use natural language, not structured output. Say "you're free after 2" not "you have availability from 14:00-17:00".
` + sm.userMemoriesPrompt()
	blocks := tools.BuildSystemBlocks(identity, toolReg, extras)

	opts := []agent.Option{agent.WithSystemBlocks(blocks)}
	if len(history) > 0 {
		opts = append(opts, agent.WithHistory(history))
	}
	opts = append(opts, agent.WithContextWindow(sm.resolveContextWindow()))
	opts = append(opts, agent.WithCompactionThreshold(sm.resolveCompactionThreshold()))
	opts = append(opts, agent.WithSummarizer(&agent.LLMSummarizer{
		Provider: sm.fastProvider,
		Model:    sm.fastModel,
	}))
	recorder := sm.openUsageRecorder()
	if recorder != nil {
		opts = append(opts, agent.WithUsageRecorder(recorder))
	}
	var executor agent.ToolExecutor = toolReg
	if onToolStart != nil {
		executor = &notifyingExecutor{delegate: toolReg, onToolStart: onToolStart}
	}
	return agent.New(sm.provider, sm.model, executor, opts...), recorder, al, nil
}

func (sm *SessionManager) registerDelegateTool(reg *tools.Registry) {
	agents := tools.DetectAgents()
	if len(agents) == 0 || sm.interactor == nil {
		return
	}
	sm.mu.Lock()
	delegateSID := sm.sessionID
	sm.mu.Unlock()
	reg.Register(tools.NewDelegateTaskTool(tools.DelegateTaskConfig{
		Interactor: sm.interactor,
		Agents:     agents,
		Tracker:    sm.taskTracker,
		ScratchDir: config.ScratchDir(delegateSID),
	}))
	reg.Register(tools.NewCheckTaskTool(sm.taskTracker))
}

func (sm *SessionManager) registerScheduleTools(reg *tools.Registry) {
	var botToken string
	var ownerID int64
	if sm.cfg.Channels != nil && sm.cfg.Channels.Telegram != nil {
		botToken = sm.cfg.Channels.Telegram.BotToken
		ownerID = sm.cfg.Channels.Telegram.OwnerID
	}
	if botToken == "" || ownerID == 0 {
		return
	}
	deps := tools.ScheduleToolDeps{
		Cfg:     sm.cfg,
		Channel: "telegram",
		ChannelMeta: scheduler.ChannelMeta{
			BotToken: botToken,
			OwnerID:  ownerID,
		},
	}
	reg.Register(tools.NewCreateScheduleTool(deps))
	reg.Register(tools.NewListSchedulesTool(deps))
	reg.Register(tools.NewDeleteScheduleTool(deps))
}

func (sm *SessionManager) initWebSearch() {
	ws, _ := tools.NewWebSearchInstance(tools.WebSearchSetup{
		WSConfig: sm.cfg.WebSearch,
		DSN:      sm.cfg.WebSearchDataDSN(),
	})
	sm.webSearch = ws

	reg, err := provider.NewRegistry(sm.cfg.Models)
	if err == nil {
		sm.fastProvider, sm.fastModel = tools.ResolveFastProvider(
			sm.cfg.Models, reg, sm.provider, sm.model,
		)
		sm.nanoProvider, sm.nanoModel = tools.ResolveNanoProvider(
			sm.cfg.Models, reg, sm.provider, sm.model,
		)
	} else {
		sm.fastProvider = sm.provider
		sm.fastModel = sm.model
		sm.nanoProvider = sm.provider
		sm.nanoModel = sm.model
	}
}

func (sm *SessionManager) registerWebTools(reg *tools.Registry) {
	deps := tools.WebToolDeps{
		WS:       sm.webSearch,
		Provider: sm.fastProvider,
		Model:    sm.fastModel,
	}
	reg.Register(tools.NewWebSearchTool(deps))
	reg.Register(tools.NewWebFetchTool(deps))
}

func (sm *SessionManager) registerSlackTools(reg *tools.Registry) {
	if sm.cfg.Slack == nil || sm.cfg.Slack.DefaultWorkspace == "" || sm.interactor == nil {
		return
	}
	creds, err := slacksrc.LoadCredentials(sm.cfg.Slack.DefaultWorkspace)
	if err != nil {
		return
	}
	client := slacksrc.NewClient(creds.Token, creds.Cookie)
	deps := tools.SlackToolDeps{Client: client, Interactor: sm.interactor}
	reg.Register(tools.NewSlackSearchTool(deps))
	reg.Register(tools.NewSlackReadChannelTool(deps))
	reg.Register(tools.NewSlackReadThreadTool(deps))
	reg.Register(tools.NewSlackSendTool(deps))
	reg.Register(tools.NewSlackEditTool(deps))
	reg.Register(tools.NewSlackReactTool(deps))
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

func (sm *SessionManager) openAuditLogger() *audit.Logger {
	return audit.OpenDefault(config.AuditDBPath())
}

func generateSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("tg-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("tg-%x", b[:])
}
