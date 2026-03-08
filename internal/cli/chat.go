package cli

import (
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

		if cfg.Models == nil || cfg.Models.Default == "" {
			return fmt.Errorf("no model configured — add models.default to ~/.obk/config.yaml (e.g. models:\n  default: anthropic/claude-sonnet-4-6)")
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

			ch.Send(response)
			fmt.Println()
		}
	},
}

func buildSystemPrompt() string {
	system := `You are a personal AI assistant powered by OpenBotKit. You help users with email, messaging, notes, and other tasks.

You have core tools available: bash (run commands), file_read, file_write, file_edit, load_skills, search_skills.

To handle domain-specific tasks (email, WhatsApp, notes, etc.), first use search_skills to find relevant skills, then use load_skills to get detailed instructions. Skills teach you how to use bash and sqlite3 for specific domains.
`

	// Append skill index if available.
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
