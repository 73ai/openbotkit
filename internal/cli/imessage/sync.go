package imessage

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	imsrc "github.com/73ai/openbotkit/source/imessage"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync messages from iMessage into local storage",
	Example: `  obk imessage sync
  obk imessage sync --full`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openIMessageDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		full, _ := cmd.Flags().GetBool("full")

		result, err := imsrc.Sync(db, imsrc.SyncOptions{
			Full: full,
		})
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		if err := config.LinkSource("imessage"); err != nil {
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
