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

var newsCmd = &cobra.Command{
	Use:   "news [query]",
	Short: "Search for recent news",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		maxResults, _ := cmd.Flags().GetInt("max-results")
		backend, _ := cmd.Flags().GetString("backend")
		timeLimit, _ := cmd.Flags().GetString("time-limit")
		region, _ := cmd.Flags().GetString("region")
		noCache, _ := cmd.Flags().GetBool("no-cache")

		var opts []wssrc.Option
		db, err := store.Open(store.SQLiteConfig(cfg.WebSearchDataDSN()))
		if err == nil {
			defer db.Close()
			if err := wssrc.Migrate(db); err == nil {
				opts = append(opts, wssrc.WithDB(db))
			}
		}

		ws := wssrc.New(wssrc.Config{WebSearch: cfg.WebSearch}, opts...)
		result, err := ws.News(cmd.Context(), args[0], wssrc.SearchOptions{
			MaxResults: maxResults,
			Backend:    backend,
			TimeLimit:  timeLimit,
			Region:     region,
			NoCache:    noCache,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		return json.NewEncoder(os.Stdout).Encode(result)
	},
}

func init() {
	newsCmd.Flags().IntP("max-results", "n", 10, "Maximum number of results")
	newsCmd.Flags().StringP("backend", "b", "auto", "News backend (auto, duckduckgo, yahoo)")
	newsCmd.Flags().StringP("time-limit", "t", "", "Time limit (d=day, w=week, m=month)")
	newsCmd.Flags().StringP("region", "r", "us-en", "Region for news results")
	newsCmd.Flags().Bool("no-cache", false, "Bypass result cache")
}
