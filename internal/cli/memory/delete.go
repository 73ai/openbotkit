package memory

import (
	"fmt"
	"strconv"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a personal memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
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
