package x

import (
	"context"
	"fmt"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/twitter"
	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync timeline from X into local storage",
	Example: `  obk x sync
  obk x sync --type following
  obk x sync --full`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		session, err := client.LoadSession()
		if err != nil {
			return fmt.Errorf("not authenticated — run 'obk x auth login' first")
		}

		xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		dsn := cfg.XDataDSN()
		if err := config.EnsureSourceDir("x"); err != nil {
			return fmt.Errorf("create source dir: %w", err)
		}

		db, err := store.Open(store.Config{
			Driver: cfg.X.Storage.Driver,
			DSN:    dsn,
		})
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		full, _ := cmd.Flags().GetBool("full")
		tlType, _ := cmd.Flags().GetString("type")

		result, err := twitter.Sync(context.Background(), db, xClient, twitter.SyncOptions{
			Full:         full,
			TimelineType: tlType,
		})
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		if err := config.LinkSource("x"); err != nil {
			return fmt.Errorf("link source: %w", err)
		}

		fmt.Printf("Sync complete: %d fetched, %d skipped", result.Fetched, result.Skipped)
		if result.Errors > 0 {
			fmt.Printf(", %d errors", result.Errors)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("full", false, "Re-sync everything")
	syncCmd.Flags().String("type", "following", "Timeline type: foryou or following")
}
