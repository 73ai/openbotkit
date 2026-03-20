package websearch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/73ai/openbotkit/config"
	wssrc "github.com/73ai/openbotkit/source/websearch"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recent search history",
	Example: `  obk websearch history
  obk websearch history --limit 50`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		historyPath := filepath.Join(config.SourceDir("websearch"), "search_history.jsonl")

		limit, _ := cmd.Flags().GetInt("limit")
		entries, err := wssrc.LoadSearchHistory(historyPath, limit)
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}
		if entries == nil {
			entries = []wssrc.SearchHistoryEntry{}
		}

		return json.NewEncoder(os.Stdout).Encode(entries)
	},
}

func init() {
	historyCmd.Flags().Int("limit", 20, "Maximum number of entries to show")
}
