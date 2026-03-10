package cli

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/priyanshujain/openbotkit/agent"
	"github.com/priyanshujain/openbotkit/agent/tools"
	clicli "github.com/priyanshujain/openbotkit/channel/cli"
	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/provider"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
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

		// Open history DB for saving conversation.
		histDB, convID, err := openHistoryDB(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: history will not be saved: %v\n", err)
		}
		if histDB != nil {
			defer histDB.Close()
		}

		// Build tool registry.
		toolReg := tools.NewRegistry()
		toolReg.Register(tools.NewBashTool(30 * time.Second))
		toolReg.Register(&tools.FileReadTool{})
		toolReg.Register(&tools.FileWriteTool{})
		toolReg.Register(&tools.FileEditTool{})
		toolReg.Register(&tools.LoadSkillsTool{})
		toolReg.Register(&tools.SearchSkillsTool{})

		// Build system prompt.
		system := buildSystemPrompt()

		a := agent.New(p, modelName, toolReg, agent.WithSystem(system))
		ch := clicli.New(os.Stdin, os.Stdout)

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

func openHistoryDB(cfg *config.Config) (*store.DB, int64, error) {
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

	sessionID := generateSessionID()
	cwd, _ := os.Getwd()
	convID, err := historysrc.UpsertConversation(db, sessionID, cwd)
	if err != nil {
		db.Close()
		return nil, 0, fmt.Errorf("create conversation: %w", err)
	}

	return db, convID, nil
}

func generateSessionID() string {
	var b [16]byte
	rand.Read(b[:])
	return fmt.Sprintf("obk-chat-%x", b[:])
}

func buildSystemPrompt() string {
	system := `You are a personal AI assistant powered by OpenBotKit.

## Tools
Available: bash, file_read, file_write, file_edit, load_skills, search_skills.
Tool names are case-sensitive. Call tools exactly as listed.

Rules:
- ALWAYS use tools to perform actions. Never say you will do something without calling the tool.
- Never predict or claim results before receiving them. Wait for tool output.
- Do not narrate routine tool calls — just call the tool. Only explain when the step is non-obvious or the user asked for details.
- If a tool call fails, analyze the error before retrying with a different approach.

## Skills
Before replying to domain-specific requests (email, WhatsApp, memories, notes, etc.):
1. Scan the "Available skills" list below for matching skill names
2. Use load_skills to read the skill's instructions
3. Use bash to run the commands from those instructions
4. If the request spans multiple domains, load and use ALL relevant skills
5. If no skill matches, use search_skills to discover one by keyword
`

	idx, err := skills.LoadIndex()
	if err == nil && len(idx.Skills) > 0 {
		system += "\nAvailable skills:\n"
		for _, s := range idx.Skills {
			system += fmt.Sprintf("- %s: %s\n", s.Name, s.Description)
		}
	}

	return system
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
