package websearch

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/73ai/openbotkit/config"
	wssrc "github.com/73ai/openbotkit/source/websearch"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

type historyEntry struct {
	Query       string `json:"query"`
	Category    string `json:"category"`
	ResultCount int    `json:"result_count"`
	Backends    string `json:"backends"`
	SearchMs    int64  `json:"search_ms"`
	CreatedAt   string `json:"created_at"`
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recent search history",
	Example: `  obk websearch history
  obk websearch history --limit 50`,
	Args: cobra.NoArgs,
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

		limit, _ := cmd.Flags().GetInt("limit")

		rows, err := db.Query("SELECT query, category, result_count, backends, search_ms, created_at FROM search_history ORDER BY created_at DESC LIMIT ?", limit)
		if err != nil {
			return fmt.Errorf("query history: %w", err)
		}
		defer rows.Close()

		var entries []historyEntry
		for rows.Next() {
			var e historyEntry
			if err := rows.Scan(&e.Query, &e.Category, &e.ResultCount, &e.Backends, &e.SearchMs, &e.CreatedAt); err != nil {
				return fmt.Errorf("scan row: %w", err)
			}
			entries = append(entries, e)
		}
		if entries == nil {
			entries = []historyEntry{}
		}

		return json.NewEncoder(os.Stdout).Encode(entries)
	},
}

func init() {
	historyCmd.Flags().Int("limit", 20, "Maximum number of entries to show")
}
