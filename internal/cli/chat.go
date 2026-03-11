package cli

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"log/slog"

	"github.com/priyanshujain/openbotkit/agent"
	"github.com/priyanshujain/openbotkit/agent/tools"
	clicli "github.com/priyanshujain/openbotkit/channel/cli"
	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	slacksrc "github.com/priyanshujain/openbotkit/source/slack"
	usagesrc "github.com/priyanshujain/openbotkit/source/usage"
	"github.com/priyanshujain/openbotkit/store"

	// Register provider factories.
	_ "github.com/priyanshujain/openbotkit/provider/anthropic"
	_ "github.com/priyanshujain/openbotkit/provider/gemini"
	_ "github.com/priyanshujain/openbotkit/provider/openai"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat with the AI assistant",
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

		router := provider.NewRouter(registry, cfg.Models)

		// Resolve the default model's provider and model name.
		providerName, modelName, err := provider.ParseModelSpec(cfg.Models.Default)
		if err != nil {
			return fmt.Errorf("parse model spec: %w", err)
		}
		p, ok := registry.Get(providerName)
		if !ok {
			return fmt.Errorf("provider %q not found", providerName)
		}

		_ = router // Will be used for tier-based routing later.

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

		// Build tool registry.
		toolReg := tools.NewStandardRegistry()
		toolReg.Register(tools.NewSubagentTool(tools.SubagentConfig{
			Provider:    p,
			Model:       modelName,
			ToolFactory: tools.NewStandardRegistry,
			System:      "You are a focused sub-agent. Complete the given task and return a concise result.",
		}))

		// Register delegate_task if external AI CLIs are available.
		registerDelegateTool(toolReg, ch)

		// Register Slack tools if configured.
		registerSlackTools(cfg, toolReg, ch)

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

// cliInteractor adapts a CLI channel to the tools.Interactor interface.
type cliInteractor struct {
	ch *clicli.Channel
}

func (c *cliInteractor) Notify(msg string) error              { return c.ch.Send(msg) }
func (c *cliInteractor) NotifyLink(text, url string) error    { return c.ch.SendLink(text, url) }
func (c *cliInteractor) RequestApproval(desc string) (bool, error) { return c.ch.RequestApproval(desc) }

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
	inter := &cliInteractor{ch: ch}
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
	inter := &cliInteractor{ch: ch}
	tracker := tools.NewTaskTracker()
	reg.Register(tools.NewDelegateTaskTool(tools.DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Tracker:    tracker,
	}))
	reg.Register(tools.NewCheckTaskTool(tracker))
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
