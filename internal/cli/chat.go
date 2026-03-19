package cli

import (
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/audit"
	"github.com/73ai/openbotkit/agent/tools"
	clicli "github.com/73ai/openbotkit/channel/cli"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/learnings"
	"github.com/73ai/openbotkit/provider"
	historysrc "github.com/73ai/openbotkit/service/history"
	slacksrc "github.com/73ai/openbotkit/source/slack"
	usagesrc "github.com/73ai/openbotkit/service/usage"
	"github.com/73ai/openbotkit/store"

	// Register provider factories.
	_ "github.com/73ai/openbotkit/provider/anthropic"
	_ "github.com/73ai/openbotkit/provider/gemini"
	_ "github.com/73ai/openbotkit/provider/groq"
	_ "github.com/73ai/openbotkit/provider/openai"
	_ "github.com/73ai/openbotkit/provider/openrouter"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat with the AI assistant",
	Example: `  obk chat`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := cfg.RequireSetup(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		registry, err := provider.NewRegistry(cfg.Models)
		if err != nil {
			return fmt.Errorf("create provider registry: %w", err)
		}

		// Resolve the default model's provider and model name.
		providerName, modelName, err := provider.ParseModelSpec(cfg.Models.Default)
		if err != nil {
			return fmt.Errorf("parse model spec: %w", err)
		}
		p, ok := registry.Get(providerName)
		if !ok {
			return fmt.Errorf("provider %q not found", providerName)
		}

		sessionID := generateSessionID()

		// Open history DB for saving conversation.
		histDB, convID, err := openHistoryDB(cfg, sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: history will not be saved: %v\n", err)
		}
		if histDB != nil {
			defer histDB.Close()
		}

		ch := clicli.New(os.Stdin, os.Stdout)

		// Set up audit logging.
		auditLogger := openAuditLogger()
		if auditLogger != nil {
			defer auditLogger.Close()
		}

		// Build tool registry with CLI approval gate.
		inter := NewCLIInteractor(ch)
		approvalRules := tools.NewApprovalRuleSet()
		toolReg := tools.NewStandardRegistry(inter, approvalRules)
		if err := config.EnsureScratchDir(sessionID); err != nil {
			slog.Warn("scratch dir creation failed", "error", err)
		}
		toolReg.SetScratchDir(config.ScratchDir(sessionID))
		defer config.CleanScratch(sessionID)
		if auditLogger != nil {
			toolReg.SetAudit(auditLogger, "cli")
		}
		scratchDir := config.ScratchDir(sessionID)
		toolReg.Register(tools.NewSubagentTool(tools.SubagentConfig{
			Provider: p,
			Model:    modelName,
			ToolFactory: func() *tools.Registry {
				r := tools.NewStandardRegistry(nil, nil)
				r.SetScratchDir(scratchDir)
				return r
			},
			System: "You are a focused sub-agent. Complete the given task and return a concise result.",
		}))

		// Register delegate_task if external AI CLIs are available.
		registerDelegateTool(toolReg, ch)

		// Register Slack tools if configured.
		registerSlackTools(cfg, toolReg, ch)

		// Register learnings tools.
		registerLearningsTools(toolReg)

		// Register web search/fetch tools.
		wsDB := registerWebTools(cfg, toolReg, registry, p, modelName)
		if wsDB != nil {
			defer wsDB.Close()
		}

		// Set up usage recording.
		usageRecorder := openUsageRecorder(cfg, providerName, "cli", sessionID)
		if usageRecorder != nil {
			defer usageRecorder.Close()
		}

		// Build system prompt with structured blocks for cache optimization.
		identity := "You are a personal AI assistant powered by OpenBotKit.\n"
		blocks := tools.BuildSystemBlocks(identity, toolReg)

		var agentOpts []agent.Option
		agentOpts = append(agentOpts, agent.WithSystemBlocks(blocks))
		if usageRecorder != nil {
			agentOpts = append(agentOpts, agent.WithUsageRecorder(usageRecorder))
		}
		a := agent.New(p, modelName, toolReg, agentOpts...)

		fmt.Println("OpenBotKit Chat (Ctrl+D to exit)")
		fmt.Println()

		for {
			input, err := ch.Receive()
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return nil
			}
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			if input == "" {
				continue
			}

			response, err := a.Run(cmd.Context(), input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}

			if histDB != nil {
				historysrc.SaveMessage(histDB, convID, "user", input)
				historysrc.SaveMessage(histDB, convID, "assistant", response)
			}

			ch.Send(response)
			fmt.Println()
		}
	},
}

func openHistoryDB(cfg *config.Config, sessionID string) (*store.DB, int64, error) {
	if err := config.EnsureSourceDir("history"); err != nil {
		return nil, 0, fmt.Errorf("ensure history dir: %w", err)
	}

	db, err := store.Open(store.Config{
		Driver: cfg.History.Storage.Driver,
		DSN:    cfg.HistoryDataDSN(),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("open history db: %w", err)
	}

	if err := historysrc.Migrate(db); err != nil {
		db.Close()
		return nil, 0, fmt.Errorf("migrate history: %w", err)
	}

	cwd, _ := os.Getwd()
	convID, err := historysrc.UpsertConversation(db, sessionID, cwd)
	if err != nil {
		db.Close()
		return nil, 0, fmt.Errorf("create conversation: %w", err)
	}

	return db, convID, nil
}

func openUsageRecorder(cfg *config.Config, providerName, channel, sessionID string) *usagesrc.Recorder {
	if err := config.EnsureSourceDir("usage"); err != nil {
		return nil
	}
	db, err := store.Open(store.Config{
		Driver: cfg.Usage.Storage.Driver,
		DSN:    cfg.UsageDataDSN(),
	})
	if err != nil {
		return nil
	}
	if err := usagesrc.Migrate(db); err != nil {
		db.Close()
		return nil
	}
	return usagesrc.NewRecorder(db, providerName, channel, sessionID)
}

func generateSessionID() string {
	var b [16]byte
	rand.Read(b[:])
	return fmt.Sprintf("obk-chat-%x", b[:])
}


func openAuditLogger() *audit.Logger {
	return audit.OpenDefault(config.AuditDBPath())
}

func registerSlackTools(cfg *config.Config, reg *tools.Registry, ch *clicli.Channel) {
	if cfg.Slack == nil || cfg.Slack.DefaultWorkspace == "" {
		return
	}
	creds, err := slacksrc.LoadCredentials(cfg.Slack.DefaultWorkspace)
	if err != nil {
		slog.Debug("slack tools not loaded: no credentials", "workspace", cfg.Slack.DefaultWorkspace)
		return
	}
	client := slacksrc.NewClient(creds.Token, creds.Cookie)
	inter := NewCLIInteractor(ch)
	deps := tools.SlackToolDeps{Client: client, Interactor: inter}

	reg.Register(tools.NewSlackSearchTool(deps))
	reg.Register(tools.NewSlackReadChannelTool(deps))
	reg.Register(tools.NewSlackReadThreadTool(deps))
	reg.Register(tools.NewSlackSendTool(deps))
	reg.Register(tools.NewSlackEditTool(deps))
	reg.Register(tools.NewSlackReactTool(deps))
}

func registerDelegateTool(reg *tools.Registry, ch *clicli.Channel) {
	agents := tools.DetectAgents()
	if len(agents) == 0 {
		return
	}
	inter := NewCLIInteractor(ch)
	tracker := tools.NewTaskTracker()
	reg.Register(tools.NewDelegateTaskTool(tools.DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Tracker:    tracker,
	}))
	reg.Register(tools.NewCheckTaskTool(tracker))
}

func registerLearningsTools(reg *tools.Registry) {
	st := learnings.New(config.LearningsDir())
	deps := tools.LearningsDeps{Store: st}
	reg.Register(tools.NewLearningSaveTool(deps))
	reg.Register(tools.NewLearningReadTool(deps))
	reg.Register(tools.NewLearningSearchTool(deps))
}

// registerWebTools adds web_search and web_fetch tools. Returns an optional
// DB handle that the caller must close when done.
func registerWebTools(cfg *config.Config, reg *tools.Registry, provRegistry *provider.Registry, defaultP provider.Provider, defaultModel string) *store.DB {
	ws, wsDB := tools.NewWebSearchInstance(tools.WebSearchSetup{
		WSConfig: cfg.WebSearch,
		DSN:      cfg.WebSearchDataDSN(),
	})
	fastP, fastModel := tools.ResolveFastProvider(cfg.Models, provRegistry, defaultP, defaultModel)
	deps := tools.WebToolDeps{WS: ws, Provider: fastP, Model: fastModel}
	reg.Register(tools.NewWebSearchTool(deps))
	reg.Register(tools.NewWebFetchTool(deps))
	return wsDB
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
