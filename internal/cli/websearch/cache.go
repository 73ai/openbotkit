package websearch

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/priyanshujain/openbotkit/config"
	wssrc "github.com/priyanshujain/openbotkit/source/websearch"
	"github.com/priyanshujain/openbotkit/store"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage websearch cache",
}

var cacheClearCmd = &cobra.Command{
	Use:     "clear",
	Short:   "Clear all cached search results and fetched pages",
	Example: `  obk websearch cache clear`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := store.Open(store.SQLiteConfig(cfg.WebSearchDataDSN()))
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()

		if err := wssrc.Migrate(db); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}

		ws := wssrc.New(wssrc.Config{WebSearch: cfg.WebSearch}, wssrc.WithDB(db))
		if err := ws.ClearCaches(); err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "cleared"})
	},
}

func init() {
	cacheCmd.AddCommand(cacheClearCmd)
}
