package memory

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/memory"
	"github.com/spf13/cobra"
)

var (
	addCategory string
	addSource   string
)

var addCmd = &cobra.Command{
	Use:   "add <content>",
	Short: "Add a personal memory",
	Example: `  obk memory add "I prefer dark mode"
  obk memory add "Lives in Da Nang" --category identity
  obk memory add "Likes Go" --source chat --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		content := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")

		if cfg.IsRemote() {
			client, err := newRemoteClient(cfg)
			if err != nil {
				return err
			}
			id, err := client.MemoryAdd(content, addCategory, addSource)
			if err != nil {
				return fmt.Errorf("add: %w", err)
			}
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]any{"id": id, "content": content, "category": addCategory})
			}
			fmt.Printf("Added memory #%d\n", id)
			return nil
		}

		dir := config.UserMemoryDir()
		if err := memory.EnsureDir(dir); err != nil {
			return fmt.Errorf("ensure user_memory dir: %w", err)
		}

		s := memory.NewStore(dir)
		id, err := s.Add(content, memory.Category(addCategory), addSource, "")
		if err != nil {
			return fmt.Errorf("add: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"id": id, "content": content, "category": addCategory})
		}
		fmt.Printf("Added memory #%d\n", id)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addCategory, "category", "preference", "category (identity, preference, relationship, project)")
	addCmd.Flags().StringVar(&addSource, "source", "manual", "source of the memory")
	addCmd.Flags().Bool("json", false, "Output as JSON")
}
