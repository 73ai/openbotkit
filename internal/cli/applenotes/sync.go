package applenotes

import (
	"fmt"
	"runtime"

	"github.com/priyanshujain/openbotkit/config"
	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync notes from Apple Notes into local storage",
	Example: `  obk applenotes sync
  obk applenotes sync --full`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("Apple Notes is only available on macOS")
		}
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openAppleNotesDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		full, _ := cmd.Flags().GetBool("full")

		result, err := ansrc.Sync(db, ansrc.SyncOptions{
			Full: full,
		})
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		if err := config.LinkSource("applenotes"); err != nil {
			return fmt.Errorf("link source: %w", err)
		}

		fmt.Printf("\nSync complete: %d synced", result.Synced)
		if result.Skipped > 0 {
			fmt.Printf(", %d skipped", result.Skipped)
		}
		if result.Errors > 0 {
			fmt.Printf(", %d errors", result.Errors)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("full", false, "Re-sync everything (ignore existing)")
}
