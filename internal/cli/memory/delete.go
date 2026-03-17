package memory

import (
	"fmt"
	"strconv"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/memory"
	"github.com/73ai/openbotkit/store"
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

		if err := config.EnsureSourceDir("user_memory"); err != nil {
			return fmt.Errorf("ensure user_memory dir: %w", err)
		}

		db, err := store.Open(store.Config{
			Driver: cfg.UserMemory.Storage.Driver,
			DSN:    cfg.UserMemoryDataDSN(),
		})
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		if err := memory.Migrate(db); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}

		if err := memory.Delete(db, id); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		fmt.Printf("Deleted memory #%d\n", id)
		return nil
	},
}

func init() {
	deleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
