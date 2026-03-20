package memory

import (
	"fmt"
	"strconv"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/memory"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a personal memory",
	Example: `  obk memory delete 3
  obk memory delete 7 --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("About to delete memory #%d. Continue? (y/N): ", id)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.IsRemote() {
			client, err := newRemoteClient(cfg)
			if err != nil {
				return err
			}
			if err := client.MemoryDelete(id); err != nil {
				return fmt.Errorf("delete: %w", err)
			}
			fmt.Printf("Deleted memory #%d\n", id)
			return nil
		}

		dir := config.UserMemoryDir()
		if err := memory.EnsureDir(dir); err != nil {
			return fmt.Errorf("ensure user_memory dir: %w", err)
		}

		s := memory.NewStore(dir)
		if err := s.Delete(id); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		fmt.Printf("Deleted memory #%d\n", id)
		return nil
	},
}

func init() {
	deleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
